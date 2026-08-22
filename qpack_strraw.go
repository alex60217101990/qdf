package qdf

import "github.com/alex60217101990/qdf/internal/unsafestr"

// Bulk-materialized ("raw") string columns for the columnar container — the
// high-cardinality counterpart to the dictionary codec in qpack_strdict.go.
//
// A mostly-distinct string column (IDs, GUIDs, emails, free text) cannot
// dict-compress: dictionary coding wins only when a small distinct set repeats.
// Written per value it also costs one heap allocation per row on decode, because
// each distinct string is materialized into its own backing array. tagColStrRaw
// instead lays every value down once, length-prefixed, so the decoder copies the
// whole column into ONE slab and every row is a sub-slice of it: the distinct
// bytes are still stored once (wire-neutral against the per-value form, which
// also stores each distinct value once and cannot intern-dedup a distinct
// column) but decode drops from n allocations to one slab plus the result slice.
//
// The codec is chosen automatically (no option) for high-cardinality columns;
// low/medium-cardinality columns keep the per-value path, where intern dedup
// shrinks the wire more than a flat layout. See tagColStrRaw for the wire format.

// qpackStrRawMinElems skips the bulk form for tiny columns, where the per-value
// path's handful of allocations is already cheap.
const qpackStrRawMinElems = 16

// tryWriteStringColumnRaw emits strs as a tagColStrRaw block when the bulk form
// is no larger on the wire than the per-value path, returning true on emit. The
// bulk form drops decode from n allocations to one slab, so it is preferred
// wherever it is wire-safe — that is the high-cardinality case, where per-value
// interning has few repeats to dedup. A single pass estimates both sizes (a
// distinct value costs its bytes once plus an intern tag; a repeat costs a state
// ref) and the bulk form is declined when interning would pack the column
// smaller, so the codec never grows the wire.
func (e *Encoder) tryWriteStringColumnRaw(strs []string) bool {
	n := len(strs)
	if n < qpackStrRawMinElems {
		return false
	}
	m := e.state.strDictMap
	if m == nil {
		m = make(map[string]uint32, min(n, 1024))
		e.state.strDictMap = m
	} else {
		clear(m)
	}
	total := 0
	bulkBytes := 0 // sum over strings of one length varuint plus the bytes
	perVal := 0    // distinct: intern tag + uvarint(len) + len; repeat: state ref (>= 1)
	for _, s := range strs {
		l := len(s)
		total += l
		lp := uvarintLen(uint64(l))
		bulkBytes += lp + l
		if l < e.minIntern {
			// WriteString does NOT intern sub-minIntern strings: it emits each
			// occurrence inline (fixstr/str8/...) with no dedup. Charge that real
			// cost per occurrence — modeling them as interned (distinct tag /
			// 1-byte repeat) over-counts perVal and could fire the bulk form even
			// though it is larger by the bulk header, breaking never-larger.
			perVal += stringInlineHeaderLen(l) + l
			continue
		}
		if _, seen := m[s]; seen {
			perVal++ // a Dense state ref is at least one byte
		} else {
			m[s] = 0
			perVal += 1 + lp + l // first occurrence: tag + length + bytes
		}
	}
	bulkHdr := 1 + uvarintLen(uint64(n)) + uvarintLen(uint64(total))
	if bulkBytes+bulkHdr > perVal {
		return false // interning packs this column smaller — keep per-value
	}

	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrRaw)
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(total))
	for _, s := range strs {
		out = appendUvarint(out, uint64(len(s)))
		out = append(out, s...)
	}
	e.buf = out
	return true
}

// writeStringColumnRawForced ALWAYS emits strs as a tagColStrRaw block, with no
// size/cardinality decline. It is the unconditional emit half of
// tryWriteStringColumnRaw (same wire shape, decoded by readStringColumnRaw for
// any n >= 0). Used by the column-diff path, which needs a wire-stateless string
// codec (raw is fully self-contained) it can build on a shared encoder state
// without polluting the intern state table.
func (e *Encoder) writeStringColumnRawForced(strs []string) {
	n := len(strs)
	total := 0
	for _, s := range strs {
		total += len(s)
	}
	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrRaw)
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(total))
	for _, s := range strs {
		out = appendUvarint(out, uint64(len(s)))
		out = append(out, s...)
	}
	e.buf = out
}

// readStringColumnRaw decodes a tagColStrRaw block (tag at d.i) into n strings.
// Under noCopy each row aliases the input buffer directly (zero allocation past
// the result slice); otherwise every row is a sub-slice of one owned slab pre-
// sized from the block's total, so the column materializes in a single backing
// allocation. Every length is validated against the remaining buffer and the
// running slab bound before use, so a hostile block can neither read out of
// range nor force the slab to reallocate (which would dangle an earlier alias).
func (d *Decoder) readStringColumnRaw(n int) ([]string, error) {
	d.i++ // consume tagColStrRaw
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, ErrTypeMismatch
	}
	total64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	// The value bytes (total) plus their length prefixes must fit in what is
	// left, so total alone cannot exceed the remaining buffer — bound it before
	// the slab allocation so a hostile varint cannot drive a huge make.
	if total64 > uint64(len(d.buf)-d.i) {
		return nil, ErrShortBuffer
	}
	total := int(total64)
	out := d.colStrScratch(n)

	if d.noCopy {
		for i := range n {
			l64, nr := readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return nil, ErrInvalidLength
			}
			d.i += nr
			if l64 > uint64(len(d.buf)-d.i) {
				return nil, ErrShortBuffer
			}
			l := int(l64)
			out[i] = unsafestr.String(d.buf[d.i : d.i+l]) // alias the input
			d.i += l
		}
		return out, nil
	}

	// One owned slab; appends are bounded to cap == total so the backing never
	// reallocates and every prior sub-slice alias stays valid.
	slab := make([]byte, 0, total)
	for i := range n {
		l64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if l64 > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
		l := int(l64)
		start := len(slab)
		if start+l > total {
			return nil, ErrShortBuffer // lengths sum past the declared total
		}
		slab = append(slab, d.buf[d.i:d.i+l]...)
		out[i] = unsafestr.String(slab[start : start+l])
		d.i += l
	}
	return out, nil
}
