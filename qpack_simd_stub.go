//go:build !amd64 || !qdf_simd

package qdf

import "encoding/binary"

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

// unpackBitsVar fallback: scalar sliding window for an arbitrary width.
func unpackBitsVar(out []uint64, in []byte, bitsPer int) {
	bitUnpackU64LEFast(out, in, bitsPer)
}

// unpackBits10/12/14/20 fallbacks: the scalar sliding-window decoder.
func unpackBits10(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 10) }
func unpackBits12(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 12) }
func unpackBits14(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 14) }
func unpackBits20(out []uint64, in []byte) { bitUnpackU64LEFast(out, in, 20) }

// packBoolsBitsLSB fallback: plain scalar bit-by-bit pack.
func packBoolsBitsLSB(dst []byte, src []bool, n int) {
	for i := range n {
		if src[i] {
			dst[i>>3] |= 1 << uint(i&7)
		}
	}
}
