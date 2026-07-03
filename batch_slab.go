package qdf

import (
	"sync"
	"unsafe"
)

// batchSlab owns every byte a Batch's handles point into: one contiguous
// grow-copy buffer. Growth copies the buffer; handles store OFFSETS, so they
// survive growth (only base moves). Pooled: Release returns it for reuse and
// bumps epoch so debug builds can catch stale handles.
type batchSlab struct {
	buf     []byte
	rowsBuf []byte // rows backing (see takeRows); NOT part of buf, never grow-copied
	epoch   uint32
}

const batchSlabInitCap = 4096

var batchSlabPool = sync.Pool{New: func() any {
	return &batchSlab{buf: make([]byte, 0, batchSlabInitCap)}
}}

func newBatchSlab() *batchSlab {
	s := batchSlabPool.Get().(*batchSlab)
	s.buf = s.buf[:0]
	return s
}

// append copies b into the slab and returns its (offset, length). The zero
// handle (0,0) is reserved for the empty string: appends never return len 0.
func (s *batchSlab) append(b []byte) (off, ln uint32) {
	if len(b) == 0 {
		return 0, 0
	}
	off = uint32(len(s.buf))
	s.buf = append(s.buf, b...)
	return off, uint32(len(b))
}

// grow reserves n more bytes of capacity up front (bulk paths).
func (s *batchSlab) grow(n int) {
	if cap(s.buf)-len(s.buf) < n {
		nb := make([]byte, len(s.buf), len(s.buf)+n+len(s.buf)/2)
		copy(nb, s.buf)
		s.buf = nb
	}
}

// takeRows returns a pointer to a ZEROED nbytes-long region backing the
// caller's []T (the generic wrapper builds the view via unsafe.Slice). The
// region is pooled ON THE SLAB across decodes: if the slab's previous rowsBuf
// already has enough capacity it is resliced and cleared in place; otherwise
// a fresh (already-zeroed) []byte is allocated. Zeroing on the reuse branch
// matters for correctness, not just hygiene: wire fields the columnar/mirror
// path never touches (schema evolution — a plan field absent from this
// message) must read back as the zero value, exactly like a fresh make would
// produce.
//
// GC-safety: rowsBuf is a []byte (noscan) reused as the backing store for
// []T. This is only safe because T is validated pointer-free by batchPlanOf
// (scalars, qdf.Str, qdf.Bytes, qdf.Time only) — the GC never needs to scan
// it. The unsafe.Slice view the wrapper builds over this region must not
// outlive Release: once the slab is pooled, the next decode may reslice and
// overwrite rowsBuf in place (same contract as buf/handles).
func (s *batchSlab) takeRows(nbytes int) unsafe.Pointer {
	if nbytes == 0 {
		return nil
	}
	if cap(s.rowsBuf) >= nbytes {
		s.rowsBuf = s.rowsBuf[:nbytes]
		clear(s.rowsBuf)
	} else {
		s.rowsBuf = make([]byte, nbytes)
	}
	return unsafe.Pointer(unsafe.SliceData(s.rowsBuf))
}

func (s *batchSlab) base() unsafe.Pointer {
	if len(s.buf) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.SliceData(s.buf))
}

func (s *batchSlab) release() {
	s.epoch++
	batchSlabPool.Put(s)
}
