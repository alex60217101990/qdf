// Package bitpack implements the pure bit-packing and SIMD kernel layer
// shared by qdf's integer codecs (FOR, Delta+FOR, dict, PFOR, ALP). It
// encodes []uint64 deltas as a tight LSB-first bit-stream and decodes it
// back, with byte-aligned and variable-width fast paths dispatched to
// hand-written AVX2 (amd64) / NEON (arm64) kernels under the qdf_simd
// build tag and a portable scalar fallback otherwise.
//
// The layer is dependency-closed: it imports only the standard library
// and golang.org/x/sys/cpu, and holds no qdf wire-format or codec state.
// All bit layouts are LSB-first within each byte; Pack and Unpack are
// exact inverses for every width in [1, 56].
package bitpack

import (
	"encoding/binary"
	"math/bits"
)

// Pack writes len(vals)*bits bits into out, LSB-first within each byte.
// out must have len >= ceil(len(vals)*bits/8). bits must be in [1, 56].
func Pack(out []byte, vals []uint64, bitsPer int) {
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

// Unpack reads len(out)*bits bits from in. bits must be in [1, 56]. in
// must have len >= ceil(len(out)*bits/8). Byte-aligned widths take a
// dedicated zero-extend fast path (asm under qdf_simd, otherwise scalar
// memcpy-with-mask); other widths use the variable-width / sliding-window
// decoders.
func Unpack(out []uint64, in []byte, bitsPer int) {
	switch bitsPer {
	case 8:
		unpackBits8(out, in)
		return
	case 10:
		unpackBits10(out, in)
		return
	case 12:
		unpackBits12(out, in)
		return
	case 14:
		unpackBits14(out, in)
		return
	case 16:
		unpackBits16(out, in)
		return
	case 20:
		unpackBits20(out, in)
		return
	case 32:
		unpackBits32(out, in)
		return
	}
	// Remaining small widths (1-7, 9, 11, 13) go through the general
	// 4-value VPSRLVQ kernel; the wider non-aligned widths (15, 17-19,
	// 21-28) use the 2-value kernel. Both fall back to the scalar window
	// on non-SIMD builds or non-AVX2 CPUs. Widths >= 29 stay scalar.
	if bitsPer <= 14 {
		unpackBitsVar(out, in, bitsPer)
		return
	}
	if bitsPer <= 28 {
		unpackBitsVarWide(out, in, bitsPer)
		return
	}
	bitUnpackU64LEFast(out, in, bitsPer)
}

// UnpackScalar is the original byte-at-a-time decoder, kept as a parity
// reference (oracle) for the fast paths' tests.
func UnpackScalar(out []uint64, in []byte, bitsPer int) {
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

// PackChunk writes a chunk of values starting at element offset elemOff in
// the output bit-stream. It is Pack generalised to write into the middle
// of an existing buffer. bitsPer must be in [0, 56].
func PackChunk(out []byte, vals []uint64, bitsPer int, elemOff int) {
	if bitsPer == 0 || len(vals) == 0 {
		return
	}
	bitOff := elemOff * bitsPer
	// Byte-aligned widths land on whole-byte boundaries (bitsPer multiple
	// of 8 ⇒ bitOff multiple of 8), so each value is an independent LE
	// store with no cross-byte accumulator. These dedicated packers mirror
	// the byte-aligned unpack fast paths and get a SIMD variant under
	// qdf_simd.
	// elemOff is always a multiple of the 64-element chunk size, so bitOff
	// is byte-aligned for every width below (64*b is a multiple of 8). The
	// dedicated packers write whole-byte chunks with a SIMD variant under
	// qdf_simd; 10/12/14/20 use VPSLLVQ + lane-OR, 8/16/32 use VPSHUFB.
	switch bitsPer {
	case 8:
		packBits8(out[bitOff>>3:], vals)
		return
	case 10:
		packBits10(out[bitOff>>3:], vals)
		return
	case 12:
		packBits12(out[bitOff>>3:], vals)
		return
	case 14:
		packBits14(out[bitOff>>3:], vals)
		return
	case 16:
		packBits16(out[bitOff>>3:], vals)
		return
	case 20:
		packBits20(out[bitOff>>3:], vals)
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
	width := uint(bitsPer)

	// Word-wise while a full 8-byte store fits: fill a 64-bit container and
	// flush it whole, instead of peeling one byte at a time. The byte-at-a-time
	// inner loop ran up to seven data-dependent iterations per value and paid a
	// bounds check per store — measured 230ms of a 490ms PackChunk on an IoT
	// encode. This mirrors the word-wise bitWriter the Gorilla path already
	// uses, little-endian to match the byte order the unpackers expect.
	//
	// Bodies reach PackChunk freshly zeroed and are filled by ascending chunk,
	// so a store that runs ahead of the current bit position only rewrites
	// zeros that a later chunk will fill; the guard keeps it inside out.
	i := 0
	for ; i < len(vals) && pos+8 <= len(out); i++ {
		x := vals[i] & mask
		space := 64 - have
		if width < space {
			acc |= x << have
			have += width
			continue
		}
		// This value straddles the container boundary: its low `space` bits
		// complete the word, the rest opens the next one.
		acc |= x << have
		binary.LittleEndian.PutUint64(out[pos:], acc)
		pos += 8
		have = width - space
		if have > 0 {
			acc = x >> space
		} else {
			acc = 0
		}
	}

	// Byte-wise for whatever is left: the tail of the buffer, where a 64-bit
	// store would run past it.
	for _, v := range vals[i:] {
		acc |= (v & mask) << have
		have += width
		for have >= 8 {
			out[pos] = byte(acc)
			acc >>= 8
			have -= 8
			pos++
		}
	}
	for have >= 8 {
		out[pos] = byte(acc)
		acc >>= 8
		have -= 8
		pos++
	}
	if have > 0 {
		// merge with any existing high bits in out[pos] (zero on a freshly
		// cleared body, but safe in case the caller passed a populated buf)
		out[pos] = byte(acc) | (out[pos] &^ byte((1<<have)-1))
	}
}

// BitsForDelta returns the number of bits required to represent values in
// [0, d], i.e. bits.Len64(d). 0 => no bits needed (all equal).
func BitsForDelta(d uint64) int {
	return bits.Len64(d)
}

// decodeHex4Scalar fills dst from a 4-bit-packed nibble stream src, mapping each
// nibble through lut: dst[2i] = lut[src[i]&0x0f], dst[2i+1] = lut[src[i]>>4].
// len(dst) outputs are produced; an odd len(dst) consumes the low nibble of the
// final src byte only. It is the portable reference for DecodeHex4 and the
// scalar tail of the SIMD kernels — keep it bit-identical to them.
func decodeHex4Scalar(dst, src []byte, lut *[16]byte) {
	n := len(dst)
	full := n &^ 1
	for k := 0; k < full; k += 2 {
		b := src[k>>1]
		dst[k] = lut[b&0x0f]
		dst[k+1] = lut[b>>4]
	}
	if full < n {
		dst[full] = lut[src[full>>1]&0x0f]
	}
}
