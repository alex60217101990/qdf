//go:build (!amd64 && !arm64) || !qdf_simd

package bitpack

import "encoding/binary"

// DecodeHex4 fills dst from a 4-bit nibble stream src via the 16-entry LUT
// (dst[2i]=lut[src[i]&0xf], dst[2i+1]=lut[src[i]>>4]). Scalar fallback for
// non-SIMD builds.
func DecodeHex4(dst, src []byte, lut *[16]byte) {
	decodeHex4Scalar(dst, src, lut)
}

// unpackBits32 fallback for non-amd64 builds and amd64 builds without
// the qdf_simd tag.
func unpackBits32(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
	}
}

// unpackBits16 fallback.
func unpackBits16(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(binary.LittleEndian.Uint16(in[i*2:]))
	}
}

// unpackBits8 fallback.
func unpackBits8(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(in[i])
	}
}

// packBits8 fallback: low byte of each value, contiguous.
func packBits8(out []byte, vals []uint64) {
	for i, v := range vals {
		out[i] = byte(v)
	}
}

// packBits16 fallback: low 2 bytes LE per value.
func packBits16(out []byte, vals []uint64) {
	for i, v := range vals {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
}

// packBits32 fallback: low 4 bytes LE per value.
func packBits32(out []byte, vals []uint64) {
	for i, v := range vals {
		binary.LittleEndian.PutUint32(out[i*4:], uint32(v))
	}
}

// packBits10/12/14/20 fallbacks: scalar accumulator packer.
func packBits10(out []byte, vals []uint64) { Pack(out, vals, 10) }
func packBits12(out []byte, vals []uint64) { Pack(out, vals, 12) }
func packBits14(out []byte, vals []uint64) { Pack(out, vals, 14) }
func packBits20(out []byte, vals []uint64) { Pack(out, vals, 20) }

// unpackBitsVar / unpackBitsVarWide fallbacks: scalar sliding window.
func unpackBitsVar(out []uint64, in []byte, bitsPer int) {
	bitUnpackU64LEFast(out, in, bitsPer)
}

func unpackBitsVarWide(out []uint64, in []byte, bitsPer int) {
	bitUnpackU64LEFast(out, in, bitsPer)
}

// unpackBits10/12/14/20 fallbacks: the scalar sliding-window decoder.
func unpackBits10(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 10) }
func unpackBits12(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 12) }
func unpackBits14(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 14) }
func unpackBits20(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 20) }

// PackBoolsLSB fallback: plain scalar bit-by-bit pack.
func PackBoolsLSB(dst []byte, src []bool, n int) {
	for i := range n {
		if src[i] {
			dst[i>>3] |= 1 << uint(i&7)
		}
	}
}
