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

func unpackBitsVar(out []uint64, in []byte, bitsPer int)     { bitUnpackU64LEFast(out, in, bitsPer) }
func unpackBitsVarWide(out []uint64, in []byte, bitsPer int) { bitUnpackU64LEFast(out, in, bitsPer) }
func unpackBits10(out []uint64, in []byte)                   { bitUnpackU64LEFast(out, in, 10) }
func unpackBits12(out []uint64, in []byte)                   { bitUnpackU64LEFast(out, in, 12) }
func unpackBits14(out []uint64, in []byte)                   { bitUnpackU64LEFast(out, in, 14) }
func unpackBits20(out []uint64, in []byte)                   { bitUnpackU64LEFast(out, in, 20) }

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
