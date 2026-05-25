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

//go:noescape
func unpackBits16AVX2(out []uint64, in []byte, n int)

//go:noescape
func unpackBits8AVX2(out []uint64, in []byte, n int)

//go:noescape
func packBoolsAVX2Block32(out []byte, in []bool, blocks int)

// packBoolsBitsLSB writes n booleans from src as ceil(n/8) bytes into
// dst, LSB-first (bool i -> bit (i%8) of dst[i/8]). dst must have
// length ceil(n/8) and be cleared. With qdf_simd and AVX2, blocks of
// 32 go through a Plan9-asm path that uses VPSLLW + VPMOVMSKB to pack
// 32 bools per iteration; the tail (n%32) uses a scalar loop.
func packBoolsBitsLSB(dst []byte, src []bool, n int) {
	off := 0
	if hasAVX2 && n >= 32 {
		blocks := n >> 5
		packBoolsAVX2Block32(dst, src, blocks)
		off = blocks << 5
	}
	for i := off; i < n; i++ {
		if src[i] {
			dst[i>>3] |= 1 << uint(i&7)
		}
	}
}

// unpackBits32 zero-extends a bitsPer=32 packed stream into out. With
// the qdf_simd build tag and AVX2 it dispatches to a Plan9-asm loop
// that lifts 4 u32 to 4 u64 per VPMOVZXDQ + VMOVDQU pair.
func unpackBits32(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		unpackBits32AVX2(out[:blocks], in, blocks)
		for i := blocks; i < n; i++ {
			out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
		}
		return
	}
	for i := 0; i < n; i++ {
		out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
	}
}

// unpackBits16 zero-extends a bitsPer=16 packed stream into out using
// AVX2 VPMOVZXWQ (4 u16 -> 4 u64) when available.
func unpackBits16(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		unpackBits16AVX2(out[:blocks], in, blocks)
		for i := blocks; i < n; i++ {
			out[i] = uint64(binary.LittleEndian.Uint16(in[i*2:]))
		}
		return
	}
	for i := 0; i < n; i++ {
		out[i] = uint64(binary.LittleEndian.Uint16(in[i*2:]))
	}
}

// unpackBits8 zero-extends a bitsPer=8 packed stream into out using
// AVX2 VPMOVZXBQ (4 u8 -> 4 u64) when available.
func unpackBits8(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		unpackBits8AVX2(out[:blocks], in, blocks)
		for i := blocks; i < n; i++ {
			out[i] = uint64(in[i])
		}
		return
	}
	for i := 0; i < n; i++ {
		out[i] = uint64(in[i])
	}
}
