package qdf

import "testing"

// TestUnpackBitsVar_MatchesReference checks the general b<=14 VPSRLVQ
// kernel (variable in-chunk offset + table-indexed shift) against the
// byte-at-a-time scalar decoder, bit-for-bit. Covers odd widths (which
// lack a fixed byte-aligned chunk) and a range of sizes that exercise
// the SIMD body, the read-headroom guard, the byte-alignment handoff,
// and the scalar tail.
func TestUnpackBitsVar_MatchesReference(t *testing.T) {
	widths := []int{5, 6, 7, 9, 11, 13}
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 64, 100, 257, 1000}
	for _, b := range widths {
		mask := uint64(1)<<uint(b) - 1
		for _, n := range sizes {
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = uint64(i*2654435761+11) & mask
			}
			body := make([]byte, (n*b+7)/8)
			bitPackU64LE(body, vals, b)

			want := make([]uint64, n)
			bitUnpackU64LEScalar(want, body, b)
			got := make([]uint64, n)
			unpackBitsVar(got, body, b)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("b=%d n=%d i=%d: got %d want %d", b, n, i, got[i], want[i])
				}
			}
		}
	}
}
