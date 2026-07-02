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
	buf   []byte
	epoch uint32
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
