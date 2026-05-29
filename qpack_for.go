package qdf

import (
	"math/bits"
	"slices"
)

// Frame-of-Reference (FOR) bit-packing for integer slices.
//
// For a slice with min and max values, every element fits in
// ceil(log2(max-min+1)) bits once min has been subtracted. The encoded
// payload is one min reference plus a tightly packed bit-stream of the
// deltas, LSB-first within each byte. With clustered integer data this
// crushes the per-element cost from 64 bits to anywhere from 0 (all
// equal) to ~16-24 bits without measurably slowing the decoder.
//
// The packer caps bits at 56 so the running 64-bit accumulator never
// overflows during the partial-byte drain (worst case: 7 carry-over bits
// + 56 new bits = 63 < 64). Slices that genuinely need 57-64 bits per
// delta should pick the raw-LE codec, which strictly beats FOR there.

const qpackForMaxBits = 56

// bitPackU64LE writes len(vals)*bits bits into out, LSB-first within each
// byte. out must have len >= ceil(len(vals)*bits/8). bits must be in
// [1, 56].
func bitPackU64LE(out []byte, vals []uint64, bitsPer int) {
	if bitsPer == 0 || len(vals) == 0 {
		return
	}
	mask := uint64(1)<<uint(bitsPer) - 1
	var acc uint64
	var have uint
	pos := 0
	for _, v := range vals {
		acc |= (v & mask) << have
		have += uint(bitsPer)
		for have >= 8 {
			out[pos] = byte(acc)
			acc >>= 8
			have -= 8
			pos++
		}
	}
	if have > 0 {
		out[pos] = byte(acc)
	}
}

// bitUnpackU64LE reads len(out)*bits bits from in. bits must be in
// [1, 56]. in must have len >= ceil(len(out)*bits/8). Byte-aligned
// widths take a dedicated zero-extend fast path (asm under qdf_simd on
// amd64, otherwise scalar memcpy-with-mask); other widths use the
// 128-bit sliding window decoder.
func bitUnpackU64LE(out []uint64, in []byte, bitsPer int) {
	switch bitsPer {
	case 8:
		unpackBits8(out, in)
		return
	case 16:
		unpackBits16(out, in)
		return
	case 32:
		unpackBits32(out, in)
		return
	}
	bitUnpackU64LEFast(out, in, bitsPer)
}

// bitUnpackU64LEScalar is the original byte-at-a-time decoder, kept as
// a parity reference for the fast path's tests.
func bitUnpackU64LEScalar(out []uint64, in []byte, bitsPer int) {
	if bitsPer == 0 {
		for i := range out {
			out[i] = 0
		}
		return
	}
	mask := uint64(1)<<uint(bitsPer) - 1
	var acc uint64
	var have uint
	pos := 0
	for i := range out {
		for have < uint(bitsPer) {
			acc |= uint64(in[pos]) << have
			have += 8
			pos++
		}
		out[i] = acc & mask
		acc >>= uint(bitsPer)
		have -= uint(bitsPer)
	}
}

// zigzagEncode64 maps a signed int64 to an unsigned int64 with
// magnitude-preserving low-bit cost: |v| small => result small.
func zigzagEncode64(v int64) uint64 {
	return uint64((v << 1) ^ (v >> 63))
}

func zigzagDecode64(u uint64) int64 {
	return int64((u >> 1) ^ -(u & 1))
}

// qpackForSizeUnsigned estimates the wire size, in bytes, of a FOR-packed
// encoding for n unsigned values whose delta range needs bits per slot
// and whose minimum value is m. Used to choose between raw and FOR.
func qpackForSizeUnsigned(n int, bitsPer int, m uint64) int {
	// tag(1) + kind(1) + bits(1) + min varuint (1..10) + count varuint (1..5 for n<=2^28)
	// + ceil(n*bits/8) body.
	hdr := 3 + uvarintLen(m) + uvarintLen(uint64(n))
	body := (n*bitsPer + 7) >> 3
	return hdr + body
}

func qpackForSizeSigned(n int, bitsPer int, m int64) int {
	hdr := 3 + uvarintLen(zigzagEncode64(m)) + uvarintLen(uint64(n))
	body := (n*bitsPer + 7) >> 3
	return hdr + body
}

func uvarintLen(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

// minMaxU64 returns the (min, max) of s. Caller must ensure len(s) > 0.
func minMaxU64(s []uint64) (uint64, uint64) {
	mn, mx := s[0], s[0]
	for _, v := range s[1:] {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

func minMaxI64(s []int64) (int64, int64) {
	mn, mx := s[0], s[0]
	for _, v := range s[1:] {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

// writePackedForUint64Slice emits a FOR-packed []uint64. Caller should
// already have chosen FOR over raw (e.g. via qpackForSizeUnsigned <
// raw-size). bitsPer must be in [0, 56]. min must equal the slice's min.
func (e *Encoder) writePackedForUint64Slice(s []uint64, mn uint64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyBytes := (n*bitsPer + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackFor, qpackKindUint64, byte(bitsPer))
	out = appendUvarint(out, mn)
	out = appendUvarint(out, uint64(n))
	if bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	// Subtract min on the fly to avoid an extra allocation.
	// Reuse a small stack-allocated chunk buffer for the delta values.
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = v - mn
		}
		bitPackChunkInto(body, chunk[:end-i], bitsPer, i)
	}
	e.buf = out
}

// bitPackChunkInto writes a chunk of delta values starting at element
// offset elemOff in the output bit-stream. It is bitPackU64LE generalised
// to write into the middle of an existing buffer.
func bitPackChunkInto(out []byte, vals []uint64, bitsPer int, elemOff int) {
	if bitsPer == 0 || len(vals) == 0 {
		return
	}
	bitOff := elemOff * bitsPer
	// Byte-aligned widths land on whole-byte boundaries (bitsPer multiple
	// of 8 ⇒ bitOff multiple of 8), so each value is an independent LE
	// store with no cross-byte accumulator. These dedicated packers mirror
	// the byte-aligned unpack fast paths and get a SIMD variant under
	// qdf_simd.
	switch bitsPer {
	case 8:
		packBits8(out[bitOff>>3:], vals)
		return
	case 16:
		packBits16(out[bitOff>>3:], vals)
		return
	case 32:
		packBits32(out[bitOff>>3:], vals)
		return
	}
	mask := uint64(1)<<uint(bitsPer) - 1
	pos := bitOff >> 3
	bitInByte := uint(bitOff & 7)
	var acc uint64
	if bitInByte > 0 {
		acc = uint64(out[pos])
	}
	have := bitInByte
	for _, v := range vals {
		acc |= (v & mask) << have
		have += uint(bitsPer)
		for have >= 8 {
			out[pos] = byte(acc)
			acc >>= 8
			have -= 8
			pos++
		}
	}
	if have > 0 {
		// merge with any existing high bits in out[pos] (zero on a freshly
		// cleared body, but safe in case the caller passed a populated buf)
		out[pos] = byte(acc) | (out[pos] &^ byte((1<<have)-1))
	}
}

// writePackedForInt64Slice mirrors writePackedForUint64Slice with a
// signed min stored as zigzag varuint.
func (e *Encoder) writePackedForInt64Slice(s []int64, mn int64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyBytes := (n*bitsPer + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackFor, qpackKindInt64, byte(bitsPer))
	out = appendUvarint(out, zigzagEncode64(mn))
	out = appendUvarint(out, uint64(n))
	if bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	mnU := uint64(mn)
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = uint64(v) - mnU
		}
		bitPackChunkInto(body, chunk[:end-i], bitsPer, i)
	}
	e.buf = out
}

// readPackedForHeader consumes the kind, bits, min, count fields after a
// tagPackFor tag. The tag itself must already have been consumed. The
// returned signedMin is meaningful only when the kind family is signed.
func (d *Decoder) readPackedForHeader(expectKind byte) (bitsPer int, unsignedMin uint64, signedMin int64, n int, body []byte, err error) {
	if d.i+2 > len(d.buf) {
		return 0, 0, 0, 0, nil, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, 0, 0, 0, nil, ErrTypeMismatch
	}
	b := d.buf[d.i]
	d.i++
	if int(b) > qpackForMaxBits {
		return 0, 0, 0, 0, nil, ErrBadTag
	}
	bitsPer = int(b)
	mn64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	if k&(1<<2) != 0 {
		signedMin = zigzagDecode64(mn64)
	} else {
		unsignedMin = mn64
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	// Validate body size in uint64 BEFORE the signed cast. A hostile
	// varuint with n64 > MaxInt would otherwise wrap int(n64) to
	// negative and the bounds check in the original form
	// (`d.i + bodyBytes > len(d.buf)`) accepts a negative bodyBytes,
	// then `d.buf[d.i : d.i+bodyBytes]` panics with a reverse-range
	// slice. Same shape as the Skip-path overflow fixed earlier;
	// applies to the body-reading path here too.
	rem := uint64(len(d.buf) - d.i)
	if bitsPer > 0 && n64 > rem*8/uint64(bitsPer) {
		return 0, 0, 0, 0, nil, ErrShortBuffer
	}
	n = int(n64)
	bodyBytes := int((n64*uint64(bitsPer) + 7) / 8)
	body = d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	return bitsPer, unsignedMin, signedMin, n, body, nil
}

func (d *Decoder) readPackedForUint64Slice() ([]uint64, error) {
	bitsPer, mn, _, n, body, err := d.readPackedForHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return out, nil
	}
	bitUnpackU64LE(out, body, bitsPer)
	if mn != 0 {
		for i := range out {
			out[i] += mn
		}
	}
	return out, nil
}

func (d *Decoder) readPackedForInt64Slice() ([]int64, error) {
	bitsPer, _, mn, n, body, err := d.readPackedForHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return out, nil
	}
	tmp := make([]uint64, n)
	bitUnpackU64LE(tmp, body, bitsPer)
	mnU := uint64(mn)
	for i, v := range tmp {
		out[i] = int64(v + mnU)
	}
	return out, nil
}

// bitsForDelta returns the number of bits required to represent values
// in [0, d], i.e. bits.Len64(d). 0 => no bits needed (all equal).
func bitsForDelta(d uint64) int {
	return bits.Len64(d)
}
