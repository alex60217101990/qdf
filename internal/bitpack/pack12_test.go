package bitpack

import (
	"bytes"
	"testing"
)

func dispatchPack(out []byte, vals []uint64, bitsPer int) {
	switch bitsPer {
	case 10:
		packBits10(out, vals)
	case 12:
		packBits12(out, vals)
	case 14:
		packBits14(out, vals)
	case 20:
		packBits20(out, vals)
	default:
		Pack(out, vals, bitsPer)
	}
}

// TestPackVarWidth_MatchesReference checks each VPSLLVQ-specialized encode
// width against the scalar accumulator packer, byte-for-byte.
func TestPackVarWidth_MatchesReference(t *testing.T) {
	widths := []int{10, 12, 14, 20}
	sizes := []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 64, 100, 257, 1000}
	for _, b := range widths {
		mask := uint64(1)<<uint(b) - 1
		for _, n := range sizes {
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = uint64(i*2654435761+11) & mask
			}
			got := make([]byte, (n*b+7)/8)
			dispatchPack(got, vals, b)
			want := make([]byte, (n*b+7)/8)
			Pack(want, vals, b)
			if !bytes.Equal(got, want) {
				t.Fatalf("b=%d n=%d: packBits mismatch\n got=%v\nwant=%v", b, n, got, want)
			}
		}
	}
}

func benchPackWidth(b *testing.B, bitsPer int) {
	b.Helper()
	vals := make([]uint64, 1024)
	mask := uint64(1)<<uint(bitsPer) - 1
	for i := range vals {
		vals[i] = uint64(i*2654435761+11) & mask
	}
	out := make([]byte, (1024*bitsPer+7)/8)
	b.SetBytes(1024 * 8)
	for b.Loop() {
		dispatchPack(out, vals, bitsPer)
	}
}

func BenchmarkPackWidth10(b *testing.B) { benchPackWidth(b, 10) }
func BenchmarkPackWidth12(b *testing.B) { benchPackWidth(b, 12) }
func BenchmarkPackWidth14(b *testing.B) { benchPackWidth(b, 14) }
func BenchmarkPackWidth20(b *testing.B) { benchPackWidth(b, 20) }
