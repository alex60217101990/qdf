package qdf

// Constant (single-distinct) string column codec: when every value in a string
// column is identical, store it once + the row count instead of n copies. See
// tagColStrConst (wire.go) for the format. Wire-minimal and decodes to a single
// allocation regardless of encoder mode, so it fixes the codegen/Fast path where
// the per-value fallback does not intern repeats.

// tryWriteStringColumnConst writes a constant string column if every element of
// strs is identical, returning true. The all-equal scan bails on the first
// mismatch, so a multi-distinct column costs O(1) here.
func (e *Encoder) tryWriteStringColumnConst(strs []string) bool {
	n := len(strs)
	if n < 2 {
		return false // a 0/1-element column is not worth a dedicated frame
	}
	first := strs[0]
	for i := 1; i < n; i++ {
		if strs[i] != first {
			return false
		}
	}
	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrConst)
	out = appendUvarint(out, uint64(len(first)))
	out = append(out, first...)
	out = appendUvarint(out, uint64(n))
	e.buf = out
	return true
}

// readStringColumnConst decodes a tagColStrConst block (tag at d.i) and returns
// n shares of the single owned string value. The block's row count must equal
// the columnar header's n (which bounds it), so a tiny input cannot drive a
// large allocation.
func (d *Decoder) readStringColumnConst(n int) ([]string, error) {
	d.i++ // consume tagColStrConst
	l64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if l64 > uint64(len(d.buf)-d.i) {
		return nil, ErrShortBuffer
	}
	l := int(l64)
	v := d.materializeStr(d.buf[d.i : d.i+l]) // shared by every row (aliases input under noCopy)
	d.i += l
	cnt64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if int(cnt64) != n {
		return nil, ErrTypeMismatch
	}
	out := make([]string, n)
	for i := range out {
		out[i] = v
	}
	return out, nil
}
