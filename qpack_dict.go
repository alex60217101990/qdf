package qdf

import (
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
)

// Dictionary-coded integer slice codec. Wire format documented at
// tagPackDict in wire.go. Wins on enum-like columns where the
// distinct count is small (≤ qpackDictMaxDistinct) but the values
// are spread far enough that the FOR / Delta+FOR bitpack can't
// reach competitive density.
//
// The encoder re-runs the distinct probe because the picker discards
// the table after the size estimate; the probe is O(n * distinct)
// and stays well under the bitpack-body work it replaces.

// writePackedDictUint64Slice emits s as a tagPackDict payload. The
// caller has already decided via pickU64Codec that dict wins.
func (e *Encoder) writePackedDictUint64Slice(s []uint64) {
	e.writeHeader()
	n := len(s)
	var table [qpackDictMaxDistinct + 1]uint64
	count, ok := probeDistinctU64(s, &table)
	if !ok || count == 0 {
		// Fall back to raw — picker mis-selected or the slice grew
		// distinct values between probe and encode. The fallback
		// keeps the wire well-formed; the only cost is a missed
		// dict win, not a panic.
		e.writePackedUint64Slice(s)
		return
	}
	bp := bitsForDistinct(count)
	bodyBytes := (n*bp + 7) >> 3
	out := slices.Grow(e.buf, 2+10+10*count+10+bodyBytes)
	out = append(out, tagPackDict, qpackKindUint64)
	out = appendUvarint(out, uint64(count))
	for i := range count {
		out = appendUvarint(out, table[i])
	}
	out = appendUvarint(out, uint64(n))
	if bp == 0 {
		// Single distinct value — body is empty, decoder broadcasts.
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	// Stage indices in a chunk buffer so bitpack.PackChunk can
	// reuse its existing LSB-first packing routine.
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			// Resolve dict index via linear scan over the small
			// table — count is bounded by qpackDictMaxDistinct so
			// this stays well inside the cache.
			for k := range count {
				if table[k] == v {
					chunk[j] = uint64(k)
					break
				}
			}
		}
		bitpack.PackChunk(body, chunk[:end-i], bp, i)
	}
	e.buf = out
}

// writePackedDictInt64Slice mirrors the unsigned variant. Dictionary
// entries are zigzag-encoded.
func (e *Encoder) writePackedDictInt64Slice(s []int64) {
	e.writeHeader()
	n := len(s)
	var table [qpackDictMaxDistinct + 1]int64
	count, ok := probeDistinctI64(s, &table)
	if !ok || count == 0 {
		e.writePackedInt64Slice(s)
		return
	}
	bp := bitsForDistinct(count)
	bodyBytes := (n*bp + 7) >> 3
	out := slices.Grow(e.buf, 2+10+10*count+10+bodyBytes)
	out = append(out, tagPackDict, qpackKindInt64)
	out = appendUvarint(out, uint64(count))
	for i := range count {
		out = appendUvarint(out, zigzagEncode64(table[i]))
	}
	out = appendUvarint(out, uint64(n))
	if bp == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			for k := range count {
				if table[k] == v {
					chunk[j] = uint64(k)
					break
				}
			}
		}
		bitpack.PackChunk(body, chunk[:end-i], bp, i)
	}
	e.buf = out
}

// readPackedDictHeader parses the kind byte and distinct count after
// a tagPackDict tag. The tag itself must already be consumed. The
// caller walks the distinct values and the element count next, then
// the bitsPer-packed body — distinct value parsing is caller-specific
// because of signed zigzag, and bitsPer is derivable from count.
func (d *Decoder) readPackedDictHeader(expectKind byte) (count int, bitsPer int, err error) {
	if d.i >= len(d.buf) {
		return 0, 0, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, 0, ErrTypeMismatch
	}
	c64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, ErrInvalidLength
	}
	d.i += nr
	if c64 == 0 || c64 > qpackDictMaxDistinct {
		return 0, 0, ErrBadTag
	}
	count = int(c64)
	bitsPer = bitsForDistinct(count)
	return count, bitsPer, nil
}

// readPackedDictUint64Slice consumes a tagPackDict body emitted with
// qpackKindUint64.
func (d *Decoder) readPackedDictUint64Slice() ([]uint64, error) {
	count, bitsPer, err := d.readPackedDictHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	var table [qpackDictMaxDistinct]uint64
	for i := range count {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		table[i] = v
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
	n := int(n64)
	out := make([]uint64, n)
	if bitsPer == 0 {
		v := table[0]
		for i := range out {
			out[i] = v
		}
		return out, nil
	}
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bitsPer) {
		return nil, ErrShortBuffer
	}
	bodyBytes := (n*bitsPer + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx := make([]uint64, n)
	bitpack.Unpack(idx, body, bitsPer)
	for i, k := range idx {
		if k >= uint64(count) {
			return nil, ErrBadTag
		}
		out[i] = table[k]
	}
	return out, nil
}

// readPackedDictInt64Slice consumes a tagPackDict body emitted with
// qpackKindInt64.
func (d *Decoder) readPackedDictInt64Slice() ([]int64, error) {
	count, bitsPer, err := d.readPackedDictHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	var table [qpackDictMaxDistinct]int64
	for i := range count {
		v, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		table[i] = zigzagDecode64(v)
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
	n := int(n64)
	out := make([]int64, n)
	if bitsPer == 0 {
		v := table[0]
		for i := range out {
			out[i] = v
		}
		return out, nil
	}
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bitsPer) {
		return nil, ErrShortBuffer
	}
	bodyBytes := (n*bitsPer + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx := make([]uint64, n)
	bitpack.Unpack(idx, body, bitsPer)
	for i, k := range idx {
		if k >= uint64(count) {
			return nil, ErrBadTag
		}
		out[i] = table[k]
	}
	return out, nil
}
