package qdf

import (
	"math/rand"
	"testing"
)

// matchedIndicesNaive is the pre-optimization per-bit reference.
func matchedIndicesNaive(b []uint64, n int, dst []int) []int {
	for i := range n {
		if getBit(b, i) {
			dst = append(dst, i)
		}
	}
	return dst
}

func TestMatchedIndicesEquivalence(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	// Cover edges: 0, sub-word, exact word multiples, +1, large; densities 0..1.
	ns := []int{0, 1, 7, 63, 64, 65, 127, 128, 129, 1000, 4096, 4097}
	densities := []float64{0, 0.001, 0.01, 0.1, 0.5, 0.9, 1.0}
	for _, n := range ns {
		words := (n + 63) >> 6
		for _, d := range densities {
			b := make([]uint64, words)
			for i := range n {
				if r.Float64() < d {
					setBit(b, i)
				}
			}
			// Set some out-of-range bits in the tail word to prove they're ignored.
			if words > 0 && n&63 != 0 {
				b[words-1] |= ^uint64(0) << uint(n&63)
			}
			want := matchedIndicesNaive(b, n, nil)
			got := matchedIndices(b, n, nil)
			if len(want) != len(got) {
				t.Fatalf("n=%d d=%g: len %d != %d", n, d, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("n=%d d=%g: idx %d = %d, want %d", n, d, i, got[i], want[i])
				}
			}
		}
	}
}

func benchMask(n int, density float64) []uint64 {
	r := rand.New(rand.NewSource(42))
	b := make([]uint64, (n+63)>>6)
	for i := range n {
		if r.Float64() < density {
			setBit(b, i)
		}
	}
	return b
}

func BenchmarkMatchedIndices(b *testing.B) {
	const n = 100_000
	for _, d := range []float64{0.01, 0.1, 0.5, 0.9} {
		mask := benchMask(n, d)
		dst := make([]int, 0, n)
		b.Run(densityName(d), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = matchedIndices(mask, n, dst[:0])
			}
		})
	}
}

func BenchmarkMatchedIndicesNaive(b *testing.B) {
	const n = 100_000
	for _, d := range []float64{0.01, 0.1, 0.5, 0.9} {
		mask := benchMask(n, d)
		dst := make([]int, 0, n)
		b.Run(densityName(d), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = matchedIndicesNaive(mask, n, dst[:0])
			}
		})
	}
}

func densityName(d float64) string {
	switch d {
	case 0.01:
		return "d1pct"
	case 0.1:
		return "d10pct"
	case 0.5:
		return "d50pct"
	default:
		return "d90pct"
	}
}
