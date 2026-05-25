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
