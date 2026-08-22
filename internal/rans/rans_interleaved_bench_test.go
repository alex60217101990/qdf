package rans

import (
	"strconv"
	"testing"
)

// BenchmarkRans benchmarks the Encode path (single-stream and interleaved-4)
// across body sizes. Used as the bench-gate target for the bufpool scratch-buffer fix.
func BenchmarkRans(b *testing.B) {
	for _, sz := range []int{1 << 12, 1 << 14, 1 << 16, 1 << 18} {
		src := mkSkewed(sz)
		for _, force := range []int{-1, interleaveN} {
			forceTagForTest = force
			name := "encode/sz" + strconv.Itoa(sz) + "/single"
			if force > 0 {
				name = "encode/sz" + strconv.Itoa(sz) + "/inter" + strconv.Itoa(force)
			}
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(sz))
				b.ResetTimer()
				for range b.N {
					_ = Encode(nil, src)
				}
			})
			forceTagForTest = 0
		}
	}
}

// BenchmarkRANSDecode compares single-stream vs the shipped interleaved-4 decode
// across body sizes. Interleaving overlaps the per-symbol dependency-chain
// latency, so it decodes ~2-2.5x the bytes per second (latency-bound loop).
func BenchmarkRANSDecode(b *testing.B) {
	for _, sz := range []int{1 << 12, 1 << 14, 1 << 16, 1 << 18} {
		src := mkSkewed(sz)
		for _, force := range []int{-1, interleaveN} { // -1 = single, interleaveN = shipped interleaved
			forceTagForTest = force
			blob := Encode(nil, src)
			forceTagForTest = 0
			name := "sz" + strconv.Itoa(sz) + "/single"
			if force > 0 {
				name = "sz" + strconv.Itoa(sz) + "/inter" + strconv.Itoa(force)
			}
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(sz))
				for range b.N {
					if _, err := Decode(blob, sz); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
