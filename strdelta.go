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
