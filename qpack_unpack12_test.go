package qdf

import (
	"testing"
)

// dispatchUnpack routes to the width-specialized decoder under test.
func dispatchUnpack(out []uint64, in []byte, bitsPer int) {
	switch bitsPer {
	case 10:
		unpackBits10(out, in)
	case 12:
		unpackBits12(out, in)
	case 14:
		unpackBits14(out, in)
	case 20:
		unpackBits20(out, in)
	default:
		bitUnpackU64LEFast(out, in, bitsPer)
	}
}

// TestUnpackVarWidth_MatchesReference checks each VPSRLVQ-specialized
// width against the byte-at-a-time scalar decoder, bit-for-bit, across
// sizes that exercise the SIMD body, the read-headroom guard, and the
// scalar tail.
func TestUnpackVarWidth_MatchesReference(t *testing.T) {
	widths := []int{10, 12, 14, 20}
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 64, 100, 1000}
	for _, bitsPer := range widths {
		mask := uint64(1)<<uint(bitsPer) - 1
		for _, n := range sizes {
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = uint64(i*2654435761+11) & mask
			}
			body := make([]byte, (n*bitsPer+7)/8)
			bitPackU64LE(body, vals, bitsPer)

			want := make([]uint64, n)
			bitUnpackU64LEScalar(want, body, bitsPer)
			got := make([]uint64, n)
			dispatchUnpack(got, body, bitsPer)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("bits=%d n=%d i=%d: got %d want %d", bitsPer, n, i, got[i], want[i])
				}
			}
		}
	}
}
