package qdf

import "testing"

// TestUnpackByteAligned_MatchesReference checks the byte-aligned decode fast
// paths (8/16/32) against the scalar reference, bit-for-bit. It is the direct
// parity guard for the amd64 VPMOVZX and arm64 NEON widen kernels.
func TestUnpackByteAligned_MatchesReference(t *testing.T) {
	widths := []int{8, 16, 32}
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 64, 100, 1000}
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
			bitUnpackU64LE(got, body, b) // dispatches to the SIMD widen path
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("b=%d n=%d i=%d: got %d want %d", b, n, i, got[i], want[i])
				}
			}
		}
	}
}
