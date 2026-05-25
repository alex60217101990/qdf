package qdf

import "slices"

// QPack codec helpers. Each codec emits a single self-described tagged
// payload that replaces the per-element tag stream for one slice. The
// codecs are opt-in (Encoder.SetQPack); decoders accept the new tags
// unconditionally so a payload written with QPack can always be read.

// writePackedBool encodes a []bool as one bit per element, LSB-first within
// each byte. Wire form: tagPackBool, varuint(n), ceil(n/8) bytes. Two
// fewer bytes than the per-element tag form once n >= 2, and 8× smaller
// past n >= 16.
func (e *Encoder) writePackedBool(s []bool) {
	e.writeHeader()
	n := len(s)
	nBytes := (n + 7) >> 3
	// Worst-case header is 1 (tag) + 10 (varuint) bytes. Reserve once so
	// the body append is a single memmove past the grow.
	out := slices.Grow(e.buf, 11+nBytes)
	out = append(out, tagPackBool)
	out = appendUvarint(out, uint64(n))
	start := len(out)
	out = out[:start+nBytes]
	body := out[start : start+nBytes]
	clear(body)
	// Eight-element unroll keeps the inner branch out of the hot loop.
	// The bool layout in Go is one byte per element, so we can read 8
	// booleans at a time and build the packed byte without a per-bit
	// branch.
	i := 0
	for ; i+8 <= n; i += 8 {
		var b byte
		if s[i] {
			b |= 1 << 0
		}
		if s[i+1] {
			b |= 1 << 1
		}
		if s[i+2] {
			b |= 1 << 2
		}
		if s[i+3] {
			b |= 1 << 3
		}
		if s[i+4] {
			b |= 1 << 4
		}
		if s[i+5] {
			b |= 1 << 5
		}
		if s[i+6] {
			b |= 1 << 6
		}
		if s[i+7] {
			b |= 1 << 7
		}
		body[i>>3] = b
	}
	for ; i < n; i++ {
		if s[i] {
			body[i>>3] |= 1 << uint(i&7)
		}
	}
	e.buf = out
}

// readPackedBool decodes a bool slice written by writePackedBool. The tag
// byte must already have been consumed.
func (d *Decoder) readPackedBool() ([]bool, error) {
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if n64 > uint64(len(d.buf)-d.i)*8 {
		// Length is in elements, payload is in bytes. Reject claims that
		// cannot possibly fit even if every byte carried 8 valid bits.
		return nil, ErrShortBuffer
	}
	n := int(n64)
	nBytes := (n + 7) >> 3
	if d.i+nBytes > len(d.buf) {
		return nil, ErrShortBuffer
	}
	out := make([]bool, n)
	base := d.i
	i := 0
	for ; i+8 <= n; i += 8 {
		b := d.buf[base+(i>>3)]
		out[i] = b&(1<<0) != 0
		out[i+1] = b&(1<<1) != 0
		out[i+2] = b&(1<<2) != 0
		out[i+3] = b&(1<<3) != 0
		out[i+4] = b&(1<<4) != 0
		out[i+5] = b&(1<<5) != 0
		out[i+6] = b&(1<<6) != 0
		out[i+7] = b&(1<<7) != 0
	}
	for ; i < n; i++ {
		out[i] = d.buf[base+(i>>3)]&(1<<uint(i&7)) != 0
	}
	d.i += nBytes
	return out, nil
}
