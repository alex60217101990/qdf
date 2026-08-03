package tans

import (
	"strconv"
	"testing"
)

// BenchmarkTansEncode benchmarks Encode across body sizes (path picked by size:
// <4096 single-stream, otherwise interleaved-4).
func BenchmarkTansEncode(b *testing.B) {
	for _, sz := range []int{1 << 12, 1 << 14, 1 << 16, 1 << 18} {
		src := mkSkewed(sz)
		b.Run("sz"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz))
			for i := 0; i < b.N; i++ {
				_ = Encode(nil, src)
			}
		})
	}
}

// BenchmarkTansDecode benchmarks Decode across body sizes.
func BenchmarkTansDecode(b *testing.B) {
	for _, sz := range []int{1 << 12, 1 << 14, 1 << 16, 1 << 18} {
		src := mkSkewed(sz)
		blob := Encode(nil, src)
		b.Run("sz"+strconv.Itoa(sz), func(b *testing.B) {
			b.SetBytes(int64(sz))
			for i := 0; i < b.N; i++ {
				if _, err := Decode(blob, sz); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
