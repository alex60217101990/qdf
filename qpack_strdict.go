package qdf

import "github.com/alex60217101990/qdf/internal/bitpack"

// Dictionary-coded string columns for the columnar container. A string
// column whose values are drawn from a small set (enum-like dimensions —
// log level, service, region, status) is cheaper to store as a distinct
// table plus a bitpacked index per row than as M interned values, because
// each interned reference still costs at least a byte while an index costs
// ceil(log2 distinct) bits. Wire format documented at tagColStrDict.
//
// The codec is gated so it never grows the wire: it is chosen only when the
// bitpacked index body is smaller than the per-value run cost (one byte per
// run boundary is the floor for the interned/repeat path). Run-heavy
// (clustered) columns keep the per-value path, which repeat-codes runs more
// cheaply than a flat index.

const (
	// qpackStrDictMaxDistinct caps the distinct count. 256 keeps the index
	// at <= 8 bits and the distinct probe bounded.
	qpackStrDictMaxDistinct = 256
	// qpackStrDictMinElems skips the probe for tiny columns where the table
	// overhead cannot pay off.
	qpackStrDictMinElems = 16
	// Cheap high-cardinality bail: if the first sample rows are mostly
	// distinct, the column is ID/text-like — abandon before the full scan.
	qpackStrDictSampleN      = 64
	qpackStrDictSampleMaxPct = 70 // distinct% over the sample above which we bail
)

// tryWriteStringColumnDict attempts to emit strs as a tagColStrDict block.
// It returns true when the dictionary form was written (and is strictly
// smaller than the per-value run floor), false when the caller should fall
// back to per-value WriteString. Encode-side scratch is reused across
// columns to avoid per-column allocation.
func (e *Encoder) tryWriteStringColumnDict(strs []string) bool {
	n := len(strs)
	if n < qpackStrDictMinElems {
		return false
	}
	m := e.state.strDictMap
	if m == nil {
		m = make(map[string]uint32, qpackStrDictMaxDistinct)
		e.state.strDictMap = m
	} else {
		clear(m)
	}
	table := e.state.colDictTable[:0]
	idx := e.state.colScratchU64[:0]
	runs := 0
	prev := ^uint32(0)
	for i, s := range strs {
		id, ok := m[s]
		if !ok {
			if len(m) >= qpackStrDictMaxDistinct {
				e.state.colDictTable = table
				e.state.colScratchU64 = idx
				return false
			}
			id = uint32(len(m))
			m[s] = id
			table = append(table, s)
		}
		idx = append(idx, uint64(id))
		if id != prev {
			runs++
		}
		prev = id
		// High-cardinality bail: after the sample, if the column is mostly
		// distinct it will not dict-compress — stop before scanning the rest.
		if i+1 == qpackStrDictSampleN && len(m)*100 > qpackStrDictSampleN*qpackStrDictSampleMaxPct {
			e.state.colDictTable = table
			e.state.colScratchU64 = idx
			return false
		}
	}
	e.state.colDictTable = table
	e.state.colScratchU64 = idx

	d := len(table)
	if d < 2 {
		// A single distinct value never beats the per-value path (one interned
		// string + a state-repeat run). The gate below would reject it anyway,
		// but make the invariant explicit: the decoder relies on count >= 2 (so
		// bits >= 1, so a non-empty index body) to keep the row count bounded by
		// the buffer. A count==1 dict would carry no body and let a tiny input
		// claim a huge n.
		return false
	}
	bits := bitsForDistinct(d)
	bodyBytes := (n*bits + 7) >> 3
	// Never-worse gate. The distinct table bytes are paid by both forms (the
	// per-value path emits each distinct string once too), so they cancel;
	// compare only the dict's fixed overhead + index body against the
	// per-value run floor (>= 1 byte per run boundary).
	overhead := 1 + uvarintLen(uint64(d)) + uvarintLen(uint64(n))
	if bodyBytes+overhead >= runs {
		return false
	}

	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrDict)
	out = appendUvarint(out, uint64(d))
	for _, s := range table {
		out = appendUvarint(out, uint64(len(s)))
		out = append(out, s...)
	}
	out = appendUvarint(out, uint64(n))
	if bits > 0 {
		start := len(out)
		out = append(out, make([]byte, bodyBytes)...)
		body := out[start : start+bodyBytes]
		var chunk [64]uint64
		for i := 0; i < n; i += len(chunk) {
			end := min(i+len(chunk), n)
			bitpack.PackChunk(body, idx[i:end], bits, i)
		}
	}
	e.buf = out
	return true
}

// readStringColumn decodes a string column of n values written by
// writeStringColumn: a tagColStrDict dictionary block, or n per-value strings.
func (d *Decoder) readStringColumn(n int) ([]string, error) {
	out := make([]string, n)
	if n > 0 && d.i < len(d.buf) && d.buf[d.i] == tagColStrDict {
		table, idx, err := d.readStringColumnDict(n)
		if err != nil {
			return nil, err
		}
		for i := range n {
			out[i] = table[idx[i]]
		}
		return out, nil
	}
	for i := range n {
		sb, err := d.readStringBytes()
		if err != nil {
			return nil, err
		}
		out[i] = string(sb) // owned copy
	}
	return out, nil
}

// readStringColumnDict decodes a tagColStrDict block (tag at d.i) into a
// distinct table and a per-row index slice of length n. Each row's string
// is table[idx[i]]; the distinct strings are allocated once and shared by
// every row that references them. n is the row count the caller expects
// from the columnar header; the block's own count must match.
func (d *Decoder) readStringColumnDict(n int) (table []string, idx []uint32, err error) {
	d.i++ // consume tagColStrDict
	c64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if c64 < 2 || c64 > qpackStrDictMaxDistinct {
		// The encoder never emits a single-distinct (count==1) dictionary —
		// per-value wins, so the never-worse gate rejects it. Requiring
		// count >= 2 here is therefore lossless for valid streams and, crucially,
		// guarantees bits >= 1 so the index body is non-empty and the row count n
		// is bounded by the remaining buffer below. A count==1 dict would carry
		// no body and let a tiny hostile input drive a huge n / allocation.
		return nil, nil, ErrBadTag
	}
	count := int(c64)
	table = make([]string, count)
	for i := range count {
		l64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, nil, ErrInvalidLength
		}
		d.i += nr
		if l64 > uint64(len(d.buf)-d.i) {
			return nil, nil, ErrShortBuffer
		}
		l := int(l64)
		table[i] = string(d.buf[d.i : d.i+l]) // owned copy
		d.i += l
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, nil, ErrTypeMismatch
	}
	// count >= 2 ⇒ bits >= 1 ⇒ the index body is non-empty, so this check
	// bounds n by the remaining buffer BEFORE any n-sized allocation; a tiny
	// input can no longer drive a large idx/tmp/output slice.
	bits := bitsForDistinct(count)
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bits) {
		return nil, nil, ErrShortBuffer
	}
	bodyBytes := (n*bits + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx = make([]uint32, n)
	tmp := make([]uint64, n)
	bitpack.Unpack(tmp, body, bits)
	for i, v := range tmp {
		if v >= c64 {
			return nil, nil, ErrBadTag
		}
		idx[i] = uint32(v)
	}
	return table, idx, nil
}
