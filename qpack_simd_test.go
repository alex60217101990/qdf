package qdf

import (
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"
)

// scalarUnpackBits32 is the reference for parity tests; the build-tag
// switch may route the production call to AVX2 asm.
func scalarUnpackBits32(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(binary.LittleEndian.Uint32(in[i*4:]))
	}
}

func TestUnpackBits32_Parity(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, n := range []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31, 32, 33, 1000, 4096} {
		in := make([]byte, n*4)
		for i := range in {
			in[i] = byte(rng.Uint32())
		}
		ref := make([]uint64, n)
		scalarUnpackBits32(ref, in)
		got := make([]uint64, n)
		unpackBits32(got, in)
		if !reflect.DeepEqual(ref, got) {
			for i := range ref {
				if ref[i] != got[i] {
					t.Fatalf("n=%d at [%d]: ref=%d got=%d", n, i, ref[i], got[i])
				}
			}
		}
	}
}

// Round-trip through the FOR codec at bits=32 to confirm the asm path
// integrates cleanly with the encoder.
func TestUnpackBits32_RoundTripVsBitPack(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	const b = 32
	mask := uint64(1)<<uint(b) - 1
	for _, n := range []int{0, 1, 4, 5, 8, 17, 1000, 4096} {
		vals := make([]uint64, n)
		for i := range vals {
			vals[i] = rng.Uint64() & mask
		}
		body := make([]byte, (n*b+7)>>3)
		bitPackU64LE(body, vals, b)
		out := make([]uint64, n)
		bitUnpackU64LE(out, body, b)
		if !reflect.DeepEqual(vals, out) {
			t.Fatalf("n=%d round-trip failed", n)
		}
	}
}

func scalarUnpackBits16(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(binary.LittleEndian.Uint16(in[i*2:]))
	}
}

func scalarUnpackBits8(out []uint64, in []byte) {
	for i := range out {
		out[i] = uint64(in[i])
	}
}

func TestUnpackBits16_Parity(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	for _, n := range []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 1000, 4096} {
		in := make([]byte, n*2)
		for i := range in {
			in[i] = byte(rng.Uint32())
		}
		ref := make([]uint64, n)
		scalarUnpackBits16(ref, in)
		got := make([]uint64, n)
		unpackBits16(got, in)
		if !reflect.DeepEqual(ref, got) {
			for i := range ref {
				if ref[i] != got[i] {
					t.Fatalf("16: n=%d [%d] ref=%d got=%d", n, i, ref[i], got[i])
				}
			}
		}
	}
}

func TestUnpackBits8_Parity(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	for _, n := range []int{0, 1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 1000, 4096} {
		in := make([]byte, n)
		for i := range in {
			in[i] = byte(rng.Uint32())
		}
		ref := make([]uint64, n)
		scalarUnpackBits8(ref, in)
		got := make([]uint64, n)
		unpackBits8(got, in)
		if !reflect.DeepEqual(ref, got) {
			for i := range ref {
				if ref[i] != got[i] {
					t.Fatalf("8: n=%d [%d] ref=%d got=%d", n, i, ref[i], got[i])
				}
			}
		}
	}
}

func TestPackBoolsBitsLSB_Parity(t *testing.T) {
	scalar := func(dst []byte, src []bool, n int) {
		for i := range n {
			if src[i] {
				dst[i>>3] |= 1 << uint(i&7)
			}
		}
	}
	rng := rand.New(rand.NewSource(23))
	for _, n := range []int{0, 1, 7, 8, 9, 31, 32, 33, 63, 64, 65, 127, 128, 129, 1000, 4096} {
		src := make([]bool, n)
		for i := range src {
			src[i] = rng.Int31()&1 == 1
		}
		nBytes := (n + 7) >> 3
		got := make([]byte, nBytes)
		want := make([]byte, nBytes)
		scalar(want, src, n)
		packBoolsBitsLSB(got, src, n)
		if !reflect.DeepEqual(got, want) {
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("n=%d byte[%d] got=%02x want=%02x", n, i, got[i], want[i])
				}
			}
		}
	}
}

func TestUnpackBits816_RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	for _, b := range []int{8, 16} {
		mask := uint64(1)<<uint(b) - 1
		for _, n := range []int{0, 1, 4, 5, 8, 1000, 4096} {
			vals := make([]uint64, n)
			for i := range vals {
				vals[i] = rng.Uint64() & mask
			}
			body := make([]byte, (n*b+7)>>3)
			bitPackU64LE(body, vals, b)
			out := make([]uint64, n)
			bitUnpackU64LE(out, body, b)
			if !reflect.DeepEqual(vals, out) {
				t.Fatalf("b=%d n=%d mismatch", b, n)
			}
		}
	}
}
