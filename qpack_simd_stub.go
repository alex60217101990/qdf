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
