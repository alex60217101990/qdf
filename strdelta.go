package qdf

// Front-delta string values (tagStrDelta) for the row-major struct path.
//
// The columnar path has its own front-delta codec (qpack_strfrontdelta.go): a
// block with an anchor interval, because a column is contiguous and a reader
// may want to seek into it. This one is per value, because row-major
// interleaves fields — there is no column to anchor and no seek to preserve,
// so the form carries no header at all.
//
// The prefix compare is shared with that codec (frontDeltaCommonPrefix): a
// 64-bit XOR is zero exactly when eight bytes match, and TrailingZeros64>>3
// turns the first differing bit into the first differing byte. Measured 3x
// faster than a byte loop on this codebase's string lengths.

// strDeltaCost is the exact byte count appendStrDelta will write. The caller
// compares it against the first-sighting form — 1 + uvarintLen(len(s)) + len(s)
// — and emits the delta only when it is strictly smaller, which is what makes
// the form never worse by construction rather than by a gate.
func strDeltaCost(base, s string) int {
	p := frontDeltaCommonPrefix(base, s)
	return 1 + uvarintLen(uint64(p)) + uvarintLen(uint64(len(s)-p)) + (len(s) - p)
}

func appendStrDelta(buf []byte, base, s string) []byte {
	p := frontDeltaCommonPrefix(base, s)
	buf = append(buf, tagStrDelta)
	buf = appendUvarint(buf, uint64(p))
	buf = appendUvarint(buf, uint64(len(s)-p))
	return appendString(buf, s[p:])
}

// readStrDelta decodes one tagStrDelta value against base and returns the
// reconstructed string, the bytes consumed, and an error.
//
// Every field is validated before a byte is copied. A hostile wire can claim a
// prefix longer than the base — which would slice past the end of a string the
// reader does not own — or a middle longer than the buffer. Both are rejected
// here rather than deeper, where the copy would already have happened.
func readStrDelta(buf []byte, base string) (string, int, error) {
	if len(buf) == 0 || buf[0] != tagStrDelta {
		return "", 0, ErrBadTag
	}
	i := 1
	p64, n := readUvarint(buf[i:])
	if n <= 0 {
		return "", 0, ErrShortBuffer
	}
	i += n
	m64, n := readUvarint(buf[i:])
	if n <= 0 {
		return "", 0, ErrShortBuffer
	}
	i += n
	if p64 > uint64(len(base)) {
		return "", 0, ErrInvalidLength
	}
	if m64 > uint64(len(buf)-i) {
		return "", 0, ErrShortBuffer
	}
	p, m := int(p64), int(m64)
	out := make([]byte, p+m)
	copy(out, base[:p])
	copy(out[p:], buf[i:i+m])
	return string(out), i + m, nil
}

// --- per-field base storage -------------------------------------------------
//
// tagStrDelta codes against the previous value of the SAME field, so the base
// has to outlive the row: each row is its own encodeStruct call. It therefore
// lives on the pooled state, beside the shape bindings that key it.
//
// The encoder keys by *typeDesc, which it always has. The decoder cannot: a
// field the target struct does not declare has no typeDesc, and that field's
// base still has to advance or the next value of it decodes against a stale
// one. So the decoder keys by wire shape ID instead.

// strDeltaBases returns this type's per-field base slice, growing it if the
// type is seen with more fields than before.
//
// The one-entry cache is the whole point: a slice of one struct type — the case
// that matters — hits it on every row, so the lookup is a pointer compare
// rather than a scan.
func (e *encState) strDeltaBases(td *typeDesc, nFields int) []string {
	if e.lastDeltaTd == td && len(e.lastDeltaBases) >= nFields {
		return e.lastDeltaBases
	}
	for i, t := range e.strDeltaTd {
		if t == td {
			b := e.strDeltaBase[i]
			if len(b) < nFields {
				b = append(b, make([]string, nFields-len(b))...)
				e.strDeltaBase[i] = b
			}
			e.lastDeltaTd, e.lastDeltaBases = td, b
			return b
		}
	}
	b := make([]string, nFields)
	e.strDeltaTd = append(e.strDeltaTd, td)
	e.strDeltaBase = append(e.strDeltaBase, b)
	e.lastDeltaTd, e.lastDeltaBases = td, b
	return b
}

// strDeltaBases returns the per-field base slice for a wire shape ID.
func (d *decState) strDeltaBases(shapeID uint32, nFields int) []string {
	if int(shapeID) >= len(d.strDeltaBase) {
		grow := int(shapeID) + 1 - len(d.strDeltaBase)
		d.strDeltaBase = append(d.strDeltaBase, make([][]string, grow)...)
	}
	b := d.strDeltaBase[shapeID]
	if len(b) < nFields {
		b = append(b, make([]string, nFields-len(b))...)
		d.strDeltaBase[shapeID] = b
	}
	return b
}

// strDeltaResetEnc clears the encoder's bases in place. The slices hold string
// headers from the previous message; a bare truncation would keep them live in
// the tail and pin that message's memory for the lifetime of the pooled state —
// the same hazard decState.reset documents for stringValues.
func (e *encState) strDeltaResetEnc() {
	for i := range e.strDeltaBase {
		clear(e.strDeltaBase[i])
	}
	clear(e.strDeltaTd)
	e.strDeltaTd = e.strDeltaTd[:0]
	e.strDeltaBase = e.strDeltaBase[:0]
	e.lastDeltaTd, e.lastDeltaBases = nil, nil
}

func (d *decState) strDeltaResetDec() {
	for i := range d.strDeltaBase {
		clear(d.strDeltaBase[i])
	}
}
