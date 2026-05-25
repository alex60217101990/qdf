package qdf

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func TestBitUnpackFast_Parity(t *testing.T) {
	for _, b := range []int{1, 2, 3, 5, 7, 8, 11, 12, 13, 16, 17, 23, 24, 32, 33, 47, 56} {
		mask := uint64(1)<<uint(b) - 1
		for _, n := range []int{0, 1, 7, 8, 9, 63, 64, 65, 127, 128, 1000, 4096} {
			rng := rand.New(rand.NewSource(int64(b*100 + n)))
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = rng.Uint64() & mask
			}
			body := make([]byte, (n*b+7)>>3)
			bitPackU64LE(body, vals, b)

			ref := make([]uint64, n)
			bitUnpackU64LEScalar(ref, body, b)

			got := make([]uint64, n)
			bitUnpackU64LEFast(got, body, b)

			if !reflect.DeepEqual(ref, got) {
				for i := range ref {
					if ref[i] != got[i] {
						t.Fatalf("bits=%d n=%d at [%d]: ref=%d got=%d", b, n, i, ref[i], got[i])
					}
				}
				t.Fatalf("bits=%d n=%d mismatch (no index diff)", b, n)
			}
		}
	}
}

func BenchmarkBitUnpackFast(b *testing.B) {
	for _, cfg := range []struct {
		n int
		b int
	}{
		{1024, 4},
		{1024, 8},
		{1024, 12},
		{1024, 16},
		{1024, 32},
		{1024, 56},
		{16384, 12},
		{16384, 32},
	} {
		bits := cfg.b
		mask := uint64(1)<<uint(bits) - 1
		rng := rand.New(rand.NewSource(1))
		vals := make([]uint64, cfg.n)
		for i := range vals {
			vals[i] = rng.Uint64() & mask
		}
		body := make([]byte, (cfg.n*bits+7)>>3)
		bitPackU64LE(body, vals, bits)
		out := make([]uint64, cfg.n)
		tag := strconv.Itoa(cfg.n) + "/" + strconv.Itoa(bits) + "b"
		b.Run("scalar/"+tag, func(b *testing.B) {
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				bitUnpackU64LEScalar(out, body, bits)
			}
		})
		b.Run("fast/"+tag, func(b *testing.B) {
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				bitUnpackU64LEFast(out, body, bits)
			}
		})
	}
}
