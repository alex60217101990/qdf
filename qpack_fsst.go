package qdf

import (
	"sync"

	"github.com/alex60217101990/qdf/internal/fsst"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

const (
	// fsstMinElems: below this a per-column table cannot pay off.
	fsstMinElems = 16
	// fsstMaxDecompPerByte bounds decode expansion: each compressed byte
	// yields at most one symbol of ≤8 bytes.
	fsstMaxDecompPerByte = 8
	// fsstProbeRounds trains a coarser table for the columnar probe's size
	// estimate (a rough table is enough to decide columnar-vs-row-major); the
	// encoder still uses the full buildRounds for the table it emits. Keeps the
	// per-column probe training off the OptCompression hot path.
	fsstProbeRounds = 1
)

// fsstBuilderPool reuses FSST training scratch across the per-column probe
// estimates and encode attempts, so a column where FSST is evaluated but not
// chosen does not re-allocate the trainer's counter + first-byte index on every
// call (the source of the OptCompression encode-allocation regression).
var fsstBuilderPool = sync.Pool{New: func() any { return fsst.NewBuilder() }}

// tryWriteStringColumnFSST trains an FSST table on strs, compresses them, and
// emits a tagColStrFSST block — but only when the block is strictly smaller
// than the raw per-value size (sum(len)+n framing). It is invoked only after
// the dictionary codec has bailed (i.e. high-cardinality columns), where the
// per-value path is dominated by the raw bytes, so raw size is a sound
// never-larger baseline. Returns true iff the block was written.
func (e *Encoder) tryWriteStringColumnFSST(strs []string) bool {
	n := len(strs)
	if n < fsstMinElems {
		return false
	}

	rawTotal := 0
	for _, s := range strs {
		rawTotal += len(s)
	}

	// A pre-trained dictionary (FSSTDict.Marshal) skips the per-batch training,
	// which is the dominant FSST encode cost; otherwise train on this column.
	tbl := e.fsstDict
	if tbl == nil {
		// Reuse a pooled [][]byte across columns/encodes instead of allocating
		// a fresh slice per column (the dominant OptCompression columnar-encode
		// allocation). The views are zero-copy; the trainer only reads them.
		samples := e.state.fsstSamples[:0]
		for _, s := range strs {
			samples = append(samples, unsafestr.Bytes(s))
		}
		e.state.fsstSamples = samples
		bld := fsstBuilderPool.Get().(*fsst.Builder)
		defer fsstBuilderPool.Put(bld)
		tbl = bld.Build(samples) // aliases bld; used within this call only
	}

	// Compress all rows into one scratch buffer, recording per-row lengths.
	comp := e.state.fsstScratch[:0]
	compLens := e.state.fsstLens[:0]
	decompTotal := 0
	for _, s := range strs {
		before := len(comp)
		comp = tbl.Compress(unsafestr.Bytes(s), comp)
		compLens = append(compLens, len(comp)-before)
		decompTotal += len(s)
	}
	e.state.fsstScratch = comp
	e.state.fsstLens = compLens

	// Block size = tag + table + uvarint(n) + uvarint(decompTotal)
	//            + sum( uvarint(compLen) + compLen ). SerializedSize avoids
	//            materializing the table just to measure it.
	size := 1 + tbl.SerializedSize() + uvarintLen(uint64(n)) + uvarintLen(uint64(decompTotal))
	for _, cl := range compLens {
		size += uvarintLen(uint64(cl)) + cl
	}
	// Never-larger baseline: raw per-value bytes + one framing byte each.
	if size >= rawTotal+n {
		return false
	}

	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrFSST)
	out = tbl.MarshalTo(out) // serialize the table straight into the output
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(decompTotal))
	pos := 0
	for _, cl := range compLens {
		out = appendUvarint(out, uint64(cl))
		out = append(out, comp[pos:pos+cl]...)
		pos += cl
	}
	e.buf = out
	return true
}

// readStringColumnFSST decodes a tagColStrFSST block (tag at d.i) into n
// strings, all backed by a single per-column slab. Every length is validated
// before allocation.
func (d *Decoder) readStringColumnFSST(n int) ([]string, error) {
	d.i++ // consume tagColStrFSST
	tbl, used, err := fsst.UnmarshalSymbolTable(d.buf[d.i:])
	if err != nil {
		return nil, err
	}
	d.i += used

	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, ErrTypeMismatch
	}

	dt64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	rem := uint64(len(d.buf) - d.i)
	if dt64 > rem*fsstMaxDecompPerByte {
		return nil, ErrShortBuffer
	}
	// Hard cap: decompTotal must not exceed maxColumnarElems * maxSymLen (8).
	// This prevents a hostile varint from driving a huge slab alloc.
	if dt64 > uint64(maxColumnarElems)*8 {
		return nil, ErrShortBuffer
	}

	slab := make([]byte, 0, int(dt64))
	out := d.colStrScratch(n)
	for i := range n {
		cl64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if cl64 > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
		cl := int(cl64)
		start := len(slab)
		var ok bool
		// Bounded decode: never let the slab grow past its pre-sized capacity
		// (dt64), so a malformed row cannot trigger a transient over-allocation.
		slab, ok = tbl.DecompressN(d.buf[d.i:d.i+cl], slab, int(dt64))
		if !ok {
			return nil, ErrShortBuffer
		}
		d.i += cl
		out[i] = unsafestr.String(slab[start:])
	}
	return out, nil
}
