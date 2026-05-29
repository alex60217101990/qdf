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

//go:noescape
func packBits8AVX2(out []byte, vals []uint64, n int)

//go:noescape
func packBits16AVX2(out []byte, vals []uint64, n int)

//go:noescape
func packBits32AVX2(out []byte, vals []uint64, n int)

//go:noescape
func unpackBits12AVX2(out []uint64, in []byte, groups int)

// unpackBits12 decodes a width-12 FOR stream: 4 values per byte-aligned
// 6-byte chunk via VPBROADCASTQ + VPSRLVQ; the remainder (and the last
// chunk, which lacks 8-byte read headroom) falls back to the scalar
// sliding window. Byte-aligned handoff: 4*groups values consume exactly
// 6*groups bytes.
func unpackBits12(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	groups := 0
	if hasAVX2 {
		groups = n / 4
		// The last chunk's VPBROADCASTQ reads in[6*(groups-1) : +8]; keep
		// that read inside the buffer.
		for groups > 0 && 6*(groups-1)+8 > len(in) {
			groups--
		}
		if groups > 0 {
			unpackBits12AVX2(out[:4*groups], in, groups)
		}
	}
	if done := 4 * groups; done < n {
		bitUnpackU64LEFast(out[done:], in[6*groups:], 12)
	}
}

//go:noescape
func unpackBits10AVX2(out []uint64, in []byte, groups int)

//go:noescape
func unpackBits14AVX2(out []uint64, in []byte, groups int)

//go:noescape
func unpackBits20AVX2(out []uint64, in []byte, pairs int)

//go:noescape
func unpackBitsVarAVX2(out []uint64, in []byte, groups int, fourB int, shifts *[32]uint64, mask uint64)

// unpackBitsVar decodes an arbitrary width b in [1,14] (including the odd
// widths that lack a fixed byte-aligned chunk). It processes 4 values per
// VPSRLVQ iteration with a per-group in-byte offset; the table `shifts`
// holds the 8 possible shift vectors (one per offset 0..7). groups is
// bounded by read headroom (each group loads 8 bytes) and by a
// byte-aligned handoff so the scalar tail can resume on a byte boundary.
func unpackBitsVar(out []uint64, in []byte, b int) {
	n := len(out)
	if n == 0 {
		return
	}
	groups := 0
	if hasAVX2 && b >= 1 && b <= 14 {
		var shifts [32]uint64
		for off := 0; off < 8; off++ {
			for k := 0; k < 4; k++ {
				shifts[off*4+k] = uint64(off + k*b)
			}
		}
		mask := uint64(1)<<uint(b) - 1
		groups = n / 4
		// Last group's VPBROADCASTQ reads 8 bytes at ((groups-1)*4b)>>3.
		for groups > 0 && ((groups-1)*4*b)/8+8 > len(in) {
			groups--
		}
		// The scalar tail resumes at bit 4*groups*b, which must be a whole
		// number of bytes (always true for even b; trims to even groups
		// for odd b).
		for groups > 0 && (4*groups*b)%8 != 0 {
			groups--
		}
		if groups > 0 {
			unpackBitsVarAVX2(out[:4*groups], in, groups, 4*b, &shifts, mask)
		}
	}
	if done := 4 * groups; done < n {
		bitUnpackU64LEFast(out[done:], in[(4*groups*b)/8:], b)
	}
}

//go:noescape
func unpackBitsVarWide2AVX2(out []uint64, in []byte, pairs int, twoB int, shifts *[16]uint64, mask uint64)

// unpackBitsVarWide decodes a width b in [15,28] — too wide for four values
// in a 64-bit window, but two values fit (7 + 2*28 < 64). It processes 2
// values per VPSRLVQ iteration with a per-pair in-byte offset; `shifts`
// holds the 8 possible 2-lane shift vectors (one per offset 0..7). pairs is
// bounded by read headroom (each pair loads 8 bytes) and by a byte-aligned
// handoff so the scalar tail resumes on a byte boundary.
func unpackBitsVarWide(out []uint64, in []byte, b int) {
	n := len(out)
	if n == 0 {
		return
	}
	pairs := 0
	if hasAVX2 && b >= 15 && b <= 28 {
		var shifts [16]uint64
		for off := 0; off < 8; off++ {
			shifts[off*2+0] = uint64(off)
			shifts[off*2+1] = uint64(off + b)
		}
		mask := uint64(1)<<uint(b) - 1
		pairs = n / 2
		// Last pair's VPBROADCASTQ reads 8 bytes at ((pairs-1)*2b)>>3.
		for pairs > 0 && ((pairs-1)*2*b)/8+8 > len(in) {
			pairs--
		}
		// The scalar tail resumes at bit 2*pairs*b, which must be a whole
		// number of bytes.
		for pairs > 0 && (2*pairs*b)%8 != 0 {
			pairs--
		}
		if pairs > 0 {
			unpackBitsVarWide2AVX2(out[:2*pairs], in, pairs, 2*b, &shifts, mask)
		}
	}
	if done := 2 * pairs; done < n {
		bitUnpackU64LEFast(out[done:], in[(2*pairs*b)/8:], b)
	}
}

// unpackBits10 decodes a width-10 FOR stream: 4 values per byte-aligned
// 5-byte chunk via VPSRLVQ, scalar tail for the remainder.
func unpackBits10(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	groups := 0
	if hasAVX2 {
		groups = n / 4
		for groups > 0 && 5*(groups-1)+8 > len(in) {
			groups--
		}
		if groups > 0 {
			unpackBits10AVX2(out[:4*groups], in, groups)
		}
	}
	if done := 4 * groups; done < n {
		bitUnpackU64LEFast(out[done:], in[5*groups:], 10)
	}
}

// unpackBits14 decodes a width-14 FOR stream: 4 values per byte-aligned
// 7-byte chunk.
func unpackBits14(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	groups := 0
	if hasAVX2 {
		groups = n / 4
		for groups > 0 && 7*(groups-1)+8 > len(in) {
			groups--
		}
		if groups > 0 {
			unpackBits14AVX2(out[:4*groups], in, groups)
		}
	}
	if done := 4 * groups; done < n {
		bitUnpackU64LEFast(out[done:], in[7*groups:], 14)
	}
}

// unpackBits20 decodes a width-20 FOR stream: 2 values per byte-aligned
// 5-byte chunk.
func unpackBits20(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	pairs := 0
	if hasAVX2 {
		pairs = n / 2
		for pairs > 0 && 5*(pairs-1)+8 > len(in) {
			pairs--
		}
		if pairs > 0 {
			unpackBits20AVX2(out[:2*pairs], in, pairs)
		}
	}
	if done := 2 * pairs; done < n {
		bitUnpackU64LEFast(out[done:], in[5*pairs:], 20)
	}
}

// packBits8 writes the low byte of each value contiguously. With AVX2 it
// dispatches to a Plan9-asm VPSHUFB gather (4 values -> 4 bytes per iter);
// the tail (n%4) uses a scalar loop. Encode inverse of unpackBits8.
func packBits8(out []byte, vals []uint64) {
	n := len(vals)
	off := 0
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		packBits8AVX2(out[:blocks], vals[:blocks], blocks)
		off = blocks
	}
	for i := off; i < n; i++ {
		out[i] = byte(vals[i])
	}
}

// packBits16 writes the low 2 bytes LE per value. AVX2 VPSHUFB gather
// (4 values -> 8 bytes per iter) with a scalar n%4 tail.
func packBits16(out []byte, vals []uint64) {
	n := len(vals)
	off := 0
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		packBits16AVX2(out[:blocks*2], vals[:blocks], blocks)
		off = blocks
	}
	for i := off; i < n; i++ {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(vals[i]))
	}
}

// packBits32 writes the low 4 bytes LE per value. AVX2 VPSHUFB gather
// (4 values -> 16 bytes per iter) with a scalar n%4 tail.
func packBits32(out []byte, vals []uint64) {
	n := len(vals)
	off := 0
	if hasAVX2 && n >= 4 {
		blocks := n &^ 3
		packBits32AVX2(out[:blocks*4], vals[:blocks], blocks)
		off = blocks
	}
	for i := off; i < n; i++ {
		binary.LittleEndian.PutUint32(out[i*4:], uint32(vals[i]))
	}
}

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
