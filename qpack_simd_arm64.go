//go:build arm64 && qdf_simd

package qdf

import "encoding/binary"

// arm64 NEON codec paths. NEON (Advanced SIMD) is baseline-mandatory on arm64,
// so there is no CPUID gate — the kernels run unconditionally. Functions not
// yet ported to NEON keep a scalar body here (the shared stub is excluded on
// arm64 && qdf_simd, so this file must define the full codec surface).

//go:noescape
func unpackBits8NEON(out []uint64, in []byte, n int)

//go:noescape
func unpackBits16NEON(out []uint64, in []byte, n int)

//go:noescape
func unpackBits32NEON(out []uint64, in []byte, n int)

// unpackBits8 zero-extends a width-8 stream into out. NEON widens 8 bytes to
// 8 uint64 per iteration (USHLL chain); the n%8 tail is scalar.
func unpackBits8(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	blocks := n &^ 7
	if blocks > 0 {
		unpackBits8NEON(out[:blocks], in, blocks)
	}
	for i := blocks; i < n; i++ {
		out[i] = uint64(in[i])
	}
}

// unpackBits16 widens 8 uint16 -> 8 uint64 per NEON iteration; n%8 tail scalar.
func unpackBits16(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	blocks := n &^ 7
	if blocks > 0 {
		unpackBits16NEON(out[:blocks], in, blocks)
	}
	for i := blocks; i < n; i++ {
		out[i] = uint64(binary.LittleEndian.Uint16(in[i*2:]))
	}
}

// unpackBits32 widens 4 uint32 -> 4 uint64 per NEON iteration; n%4 tail scalar.
func unpackBits32(out []uint64, in []byte) {
	n := len(out)
	if n == 0 {
		return
	}
	blocks := n &^ 3
	if blocks > 0 {
		unpackBits32NEON(out[:blocks], in, blocks)
	}
	for i := blocks; i < n; i++ {
		out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
	}
}

// --- scalar bodies (to be replaced by NEON in later tasks) ---

//go:noescape
func unpackVar2NEON(out []uint64, in []byte, pairs int, twoB int, shifts *[16]int64, mask uint64)

// unpackVarNEON decodes an arbitrary width b in [1,28] using NEON: 2 values
// per iteration (128-bit = 2 uint64 lanes). Each pair broadcasts the 8-byte
// window (LD1R) and right-shifts each lane by its in-window offset via USHL
// with a negative shift vector, then masks. The scalar sliding window handles
// the remainder; group count is bounded by 8-byte read headroom and a
// byte-aligned handoff. Widths > 28 stay scalar.
func unpackVarNEON(out []uint64, in []byte, b int) {
	n := len(out)
	if n == 0 {
		return
	}
	pairs := 0
	if b >= 1 && b <= 28 {
		var shifts [16]int64
		for off := 0; off < 8; off++ {
			shifts[off*2+0] = -int64(off)
			shifts[off*2+1] = -int64(off + b)
		}
		mask := uint64(1)<<uint(b) - 1
		pairs = n / 2
		for pairs > 0 && ((pairs-1)*2*b)/8+8 > len(in) {
			pairs--
		}
		for pairs > 0 && (2*pairs*b)%8 != 0 {
			pairs--
		}
		if pairs > 0 {
			unpackVar2NEON(out[:2*pairs], in, pairs, 2*b, &shifts, mask)
		}
	}
	if done := 2 * pairs; done < n {
		bitUnpackU64LEFast(out[done:], in[(2*pairs*b)/8:], b)
	}
}

func unpackBitsVar(out []uint64, in []byte, bitsPer int)     { unpackVarNEON(out, in, bitsPer) }
func unpackBitsVarWide(out []uint64, in []byte, bitsPer int) { unpackVarNEON(out, in, bitsPer) }
func unpackBits10(out []uint64, in []byte)                   { unpackVarNEON(out, in, 10) }
func unpackBits12(out []uint64, in []byte)                   { unpackVarNEON(out, in, 12) }
func unpackBits14(out []uint64, in []byte)                   { unpackVarNEON(out, in, 14) }
func unpackBits20(out []uint64, in []byte)                   { unpackVarNEON(out, in, 20) }

func packBits8(out []byte, vals []uint64) {
	for i, v := range vals {
		out[i] = byte(v)
	}
}

func packBits16(out []byte, vals []uint64) {
	for i, v := range vals {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
}

func packBits32(out []byte, vals []uint64) {
	for i, v := range vals {
		binary.LittleEndian.PutUint32(out[i*4:], uint32(v))
	}
}

func packBits10(out []byte, vals []uint64) { bitPackU64LE(out, vals, 10) }
func packBits12(out []byte, vals []uint64) { bitPackU64LE(out, vals, 12) }
func packBits14(out []byte, vals []uint64) { bitPackU64LE(out, vals, 14) }
func packBits20(out []byte, vals []uint64) { bitPackU64LE(out, vals, 20) }

func packBoolsBitsLSB(dst []byte, src []bool, n int) {
	for i := range n {
		if src[i] {
			dst[i>>3] |= 1 << uint(i&7)
		}
	}
}
