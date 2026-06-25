package bitpack

import "testing"

// TestUnpackBitsVarWide_MatchesReference checks the general 15<=b<=28
// VPSRLVQ kernel (2 values per 64-bit window) against the byte-at-a-time
// scalar decoder, bit-for-bit, across sizes that hit the SIMD body, the
// read-headroom guard, the byte-alignment trim, and the scalar tail.
func TestUnpackBitsVarWide_MatchesReference(t *testing.T) {
	widths := []int{15, 17, 18, 19, 21, 22, 23, 24, 25, 26, 27, 28}
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 64, 100, 257, 1000}
	for _, b := range widths {
		mask := uint64(1)<<uint(b) - 1
		for _, n := range sizes {
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = uint64(i*2654435761+11) & mask
			}
			body := make([]byte, (n*b+7)/8)
			Pack(body, vals, b)

			want := make([]uint64, n)
			UnpackScalar(want, body, b)
			got := make([]uint64, n)
			unpackBitsVarWide(got, body, b)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("b=%d n=%d i=%d: got %d want %d", b, n, i, got[i], want[i])
				}
			}
		}
	}
}
