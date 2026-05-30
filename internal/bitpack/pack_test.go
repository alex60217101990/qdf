package bitpack

import (
	"bytes"
	"testing"
)

const benchPackN = 1024

func benchPackVals(mask uint64) []uint64 {
	vals := make([]uint64, benchPackN)
	for i := range vals {
		vals[i] = uint64(i*2654435761+11) & mask
	}
	return vals
}

func BenchmarkPack8_Accumulator(b *testing.B) {
	vals := benchPackVals(0xFF)
	out := make([]byte, 1024)
	b.SetBytes(1024)
	for b.Loop() {
		Pack(out, vals, 8)
	}
}

func BenchmarkPack8_Fast(b *testing.B) {
	vals := benchPackVals(0xFF)
	out := make([]byte, 1024)
	b.SetBytes(1024)
	for b.Loop() {
		packBits8(out, vals)
	}
}

func BenchmarkPack16_Accumulator(b *testing.B) {
	vals := benchPackVals(0xFFFF)
	out := make([]byte, 1024*2)
	b.SetBytes(1024 * 2)
	for b.Loop() {
		Pack(out, vals, 16)
	}
}

func BenchmarkPack16_Fast(b *testing.B) {
	vals := benchPackVals(0xFFFF)
	out := make([]byte, 1024*2)
	b.SetBytes(1024 * 2)
	for b.Loop() {
		packBits16(out, vals)
	}
}

func BenchmarkPack32_Accumulator(b *testing.B) {
	vals := benchPackVals(0xFFFFFFFF)
	out := make([]byte, 1024*4)
	b.SetBytes(1024 * 4)
	for b.Loop() {
		Pack(out, vals, 32)
	}
}

func BenchmarkPack32_Fast(b *testing.B) {
	vals := benchPackVals(0xFFFFFFFF)
	out := make([]byte, 1024*4)
	b.SetBytes(1024 * 4)
	for b.Loop() {
		packBits32(out, vals)
	}
}

// packReference packs vals at bitsPer using the general accumulator packer,
// the source of truth the byte-aligned fast paths must match byte-for-byte.
func packReference(vals []uint64, bitsPer int) []byte {
	out := make([]byte, (len(vals)*bitsPer+7)/8)
	Pack(out, vals, bitsPer)
	return out
}

var packSizes = []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 33, 100, 1000}

func TestPackBits8_MatchesReference(t *testing.T) {
	for _, n := range packSizes {
		vals := make([]uint64, n)
		for i := range vals {
			vals[i] = uint64(i*7+3) & 0xFF
		}
		got := make([]byte, n)
		packBits8(got, vals)
		want := packReference(vals, 8)
		if !bytes.Equal(got, want) {
			t.Fatalf("n=%d: packBits8 mismatch\n got=%v\nwant=%v", n, got, want)
		}
	}
}

func TestPackBits16_MatchesReference(t *testing.T) {
	for _, n := range packSizes {
		vals := make([]uint64, n)
		for i := range vals {
			vals[i] = uint64(i*1337+9) & 0xFFFF
		}
		got := make([]byte, n*2)
		packBits16(got, vals)
		want := packReference(vals, 16)
		if !bytes.Equal(got, want) {
			t.Fatalf("n=%d: packBits16 mismatch\n got=%v\nwant=%v", n, got, want)
		}
	}
}

func TestPackBits32_MatchesReference(t *testing.T) {
	for _, n := range packSizes {
		vals := make([]uint64, n)
		for i := range vals {
			vals[i] = uint64(i*2654435761+11) & 0xFFFFFFFF
		}
		got := make([]byte, n*4)
		packBits32(got, vals)
		want := packReference(vals, 32)
		if !bytes.Equal(got, want) {
			t.Fatalf("n=%d: packBits32 mismatch\n got=%v\nwant=%v", n, got, want)
		}
	}
}
