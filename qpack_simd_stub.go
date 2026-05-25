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

// packBoolsBitsLSB fallback: plain scalar bit-by-bit pack.
func packBoolsBitsLSB(dst []byte, src []bool, n int) {
	for i := range n {
		if src[i] {
			dst[i>>3] |= 1 << uint(i&7)
		}
	}
}
