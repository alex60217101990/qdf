package qdf

import "testing"

// Decode-side SIMD headroom probe. bitUnpackU64LE dispatches to the
// VPMOVZX asm under qdf_simd for byte-aligned widths and to the scalar
// 128-bit sliding window for the rest. Comparing the same benchmark
// across default and qdf_simd builds shows how much the simple byte-
// aligned SIMD actually buys on the expanding decode path — a cheap
// proxy for whether the much harder arbitrary-width SIMD unpack is worth
// the per-width asm.

const unpackBenchN = 1024

func benchUnpack(b *testing.B, bitsPer int) {
	vals := make([]uint64, unpackBenchN)
	mask := uint64(1)<<uint(bitsPer) - 1
	for i := range vals {
		vals[i] = uint64(i*2654435761+11) & mask
	}
	body := make([]byte, (unpackBenchN*bitsPer+7)/8)
	bitPackU64LE(body, vals, bitsPer)
	out := make([]uint64, unpackBenchN)
	b.SetBytes(int64(unpackBenchN * 8))
	for b.Loop() {
		bitUnpackU64LE(out, body, bitsPer)
	}
}

func BenchmarkUnpack8(b *testing.B)  { benchUnpack(b, 8) }
func BenchmarkUnpack16(b *testing.B) { benchUnpack(b, 16) }
func BenchmarkUnpack32(b *testing.B) { benchUnpack(b, 32) }

// Arbitrary (non-byte-aligned) widths — always the scalar sliding window
// today. Baseline a SIMD VPSRLVQ unpack would have to beat.
func BenchmarkUnpack10(b *testing.B) { benchUnpack(b, 10) }
func BenchmarkUnpack12(b *testing.B) { benchUnpack(b, 12) }
func BenchmarkUnpack14(b *testing.B) { benchUnpack(b, 14) }
func BenchmarkUnpack20(b *testing.B) { benchUnpack(b, 20) }

// BenchmarkUnpack12Direct exercises unpackBits12 (the VPSRLVQ path under
// qdf_simd, scalar otherwise) directly, isolating the width-12 codec from
// the bitUnpackU64LE dispatch.
func BenchmarkUnpack12Direct(b *testing.B) {
	vals := make([]uint64, unpackBenchN)
	for i := range vals {
		vals[i] = uint64(i*2654435761+11) & 0xFFF
	}
	body := make([]byte, (unpackBenchN*12+7)/8)
	bitPackU64LE(body, vals, 12)
	out := make([]uint64, unpackBenchN)
	b.SetBytes(int64(unpackBenchN * 8))
	for b.Loop() {
		unpackBits12(out, body)
	}
}
