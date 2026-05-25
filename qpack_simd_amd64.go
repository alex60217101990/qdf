//go:build amd64 && qdf_simd

package qdf

import (
	"encoding/binary"

	"golang.org/x/sys/cpu"
)

// hasAVX2 is set once at init from CPUID. The bitUnpack dispatcher
// consults it before calling into Plan9 asm so non-AVX2 amd64 targets
// (very old hardware) still work.
var hasAVX2 = cpu.X86.HasAVX2

//go:noescape
func unpackBits32AVX2(out []uint64, in []byte, n int)

// unpackBits32 zero-extends a bitsPer=32 packed stream into out. With
// the qdf_simd build tag and AVX2 it dispatches to a Plan9-asm loop
// that lifts 4 u32 to 4 u64 per VPMOVZXDQ + VMOVDQU pair.
func unpackBits32(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	if hasAVX2 && n >= 4 {
		// Take the 4-aligned prefix through asm.
		blocks := n &^ 3
		unpackBits32AVX2(out[:blocks], in, blocks)
		// Scalar tail.
		for i := blocks; i < n; i++ {
			out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
		}
		return
	}
	for i := 0; i < n; i++ {
		out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
	}
}
