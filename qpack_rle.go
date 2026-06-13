package qdf

import "slices"

// Run-length encoded integer slice codec. Compact when many consecutive
// values repeat — common shape in telemetry: HTTP Status columns, log
// Level enum-likes, sparse counter snapshots where most readings sit at
// zero. Wire format documented at tagPackRLE in wire.go.
//
// The encoder writes one (value, runLen) pair per run; the decoder
// unrolls until the declared element count is consumed.

// writePackedRLEUint64Slice emits s as a tagPackRLE payload. n == 0
// still writes the tag/kind/n triplet so a round-trip distinguishes an
// empty slice from no slice. Caller already decided via pickU64Codec
// that RLE wins; no probe is repeated here.
func (e *Encoder) writePackedRLEUint64Slice(s []uint64) {
	e.writeHeader()
	n := len(s)
	// Conservative pre-grow: 2 byte tag/kind + uvarint(n) + at most
	// 20 bytes per run. nRuns ≤ n so the cap is bounded.
	out := slices.Grow(e.buf, 2+10+20*n)
	out = append(out, tagPackRLE, qpackKindUint64)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	runLen := uint64(1)
	prev := s[0]
	for i := 1; i < n; i++ {
		if s[i] == prev {
			runLen++
			continue
		}
		out = appendUvarint(out, prev)
		out = appendUvarint(out, runLen)
		runLen = 1
		prev = s[i]
	}
	out = appendUvarint(out, prev)
	out = appendUvarint(out, runLen)
	e.buf = out
}

// writePackedRLEInt64Slice mirrors the unsigned variant but zigzag-
// encodes each run value so small negatives stay one byte.
func (e *Encoder) writePackedRLEInt64Slice(s []int64) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+20*n)
	out = append(out, tagPackRLE, qpackKindInt64)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	runLen := uint64(1)
	prev := s[0]
	for i := 1; i < n; i++ {
		if s[i] == prev {
			runLen++
			continue
		}
		out = appendUvarint(out, zigzagEncode64(prev))
		out = appendUvarint(out, runLen)
		runLen = 1
		prev = s[i]
	}
	out = appendUvarint(out, zigzagEncode64(prev))
	out = appendUvarint(out, runLen)
	e.buf = out
}

// readPackedRLEHeader parses the kind byte and element count. The tag
// itself must already be consumed; the caller dispatches on kind to
// pick the signed / unsigned body decoder.
func (d *Decoder) readPackedRLEHeader(expectKind byte) (n int, err error) {
	if d.i >= len(d.buf) {
		return 0, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, ErrTypeMismatch
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, ErrInvalidLength
	}
	d.i += nr
	// In a columnar column every codec must yield exactly the struct count
	// (colMaxLen); RLE can legitimately claim far more elements than remaining
	// bytes (a long run is 2 bytes), so unlike the body-bounded codecs it MUST
	// be gated by colLenOK or a tiny body could claim a multi-GB element count.
	if !d.colLenOK(n64) {
		return 0, ErrInvalidLength
	}
	if n64 > uint64(len(d.buf)-d.i) {
		// Standalone decode (colMaxLen == 0): no per-element body bound exists
		// for RLE, so cap the obvious lie. Each run is ≥ 2 bytes; the body loop
		// catches subtler over-claims. Shares the constant-codec ceiling so a
		// single 2-byte run can't claim a multi-GB element count (OOM-DoS).
		if n64 > qpackMaxStandaloneCount {
			return 0, ErrInvalidLength
		}
	}
	return int(n64), nil
}

// readPackedRLEUint64Slice consumes a tagPackRLE body emitted with
// qpackKindUint64 and returns the unrolled []uint64.
func (d *Decoder) readPackedRLEUint64Slice() ([]uint64, error) {
	n, err := d.readPackedRLEHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	idx := 0
	for idx < n {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		runLen, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if runLen == 0 || uint64(idx)+runLen > uint64(n) {
			return nil, ErrInvalidLength
		}
		end := idx + int(runLen)
		for i := idx; i < end; i++ {
			out[i] = v
		}
		idx = end
	}
	return out, nil
}

// readPackedRLEInt64Slice consumes a tagPackRLE body emitted with
// qpackKindInt64 and returns the unrolled []int64. Each value comes
// back through zigzagDecode64.
func (d *Decoder) readPackedRLEInt64Slice() ([]int64, error) {
	n, err := d.readPackedRLEHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	idx := 0
	for idx < n {
		v64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		runLen, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if runLen == 0 || uint64(idx)+runLen > uint64(n) {
			return nil, ErrInvalidLength
		}
		v := zigzagDecode64(v64)
		end := idx + int(runLen)
		for i := idx; i < end; i++ {
			out[i] = v
		}
		idx = end
	}
	return out, nil
}
