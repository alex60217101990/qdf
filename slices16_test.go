package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// Before the 16-bit fast paths, []uint16/[]int16 fell through to the generic
// reflect encoder (one uvarint per element), which EXPANDS high-entropy data
// by 1.5x. These tests pin both the round-trip and the never-larger floor.
func TestSlice16RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	randU := make([]uint16, 4096)
	randI := make([]int16, 4096)
	for i := range randU {
		randU[i] = uint16(rng.Uint32())
		randI[i] = int16(rng.Uint32())
	}
	seqU := make([]uint16, 1000)
	seqI := make([]int16, 1000)
	for i := range seqU {
		seqU[i] = uint16(i)
		seqI[i] = int16(i - 500)
	}
	constU := make([]uint16, 500)
	constI := make([]int16, 500)
	for i := range constU {
		constU[i] = 4242
		constI[i] = -4242
	}

	type rowU struct{ V []uint16 }
	type rowI struct{ V []int16 }

	uCases := [][]uint16{
		nil, {}, {0}, {math.MaxUint16}, {0, math.MaxUint16, 1, 32768},
		seqU, constU, randU,
	}
	iCases := [][]int16{
		nil, {}, {0}, {math.MinInt16}, {math.MaxInt16},
		{math.MinInt16, -1, 0, 1, math.MaxInt16},
		seqI, constI, randI,
	}
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression, OptQPack} {
		for _, c := range uCases {
			blob, err := Marshal(rowU{c}, opts)
			if err != nil {
				t.Fatalf("uint16 encode opts=%v: %v", opts, err)
			}
			var out rowU
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatalf("uint16 decode opts=%v n=%d: %v", opts, len(c), err)
			}
			if !reflect.DeepEqual(c, out.V) {
				t.Fatalf("uint16 mismatch opts=%v n=%d: %v vs %v", opts, len(c), c, out.V)
			}
		}
		for _, c := range iCases {
			blob, err := Marshal(rowI{c}, opts)
			if err != nil {
				t.Fatalf("int16 encode opts=%v: %v", opts, err)
			}
			var out rowI
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatalf("int16 decode opts=%v n=%d: %v", opts, len(c), err)
			}
			if !reflect.DeepEqual(c, out.V) {
				t.Fatalf("int16 mismatch opts=%v n=%d", opts, len(c))
			}
		}
	}
}

// TestSlice16NeverLarger pins the floor the fast paths exist for: incompressible
// 16-bit data must cost ~2 B/elem, not the 3 B/elem the varint path charged.
func TestSlice16NeverLarger(t *testing.T) {
	const n = 8192
	rng := rand.New(rand.NewSource(9))
	u := make([]uint16, n)
	s := make([]int16, n)
	for i := range u {
		u[i] = uint16(rng.Uint32())
		s[i] = int16(rng.Uint32())
	}
	type rowU struct{ V []uint16 }
	type rowI struct{ V []int16 }
	const slack = 64 // header + tag + length
	for _, opts := range []Options{OptBalanced, OptQPack} {
		bu, err := Marshal(rowU{u}, opts)
		if err != nil {
			t.Fatal(err)
		}
		bi, err := Marshal(rowI{s}, opts)
		if err != nil {
			t.Fatal(err)
		}
		if len(bu) > 2*n+slack {
			t.Fatalf("uint16 opts=%v: %d bytes for %d values (raw=%d)", opts, len(bu), n, 2*n)
		}
		if len(bi) > 2*n+slack {
			t.Fatalf("int16 opts=%v: %d bytes for %d values (raw=%d)", opts, len(bi), n, 2*n)
		}
	}
}

// TestSlice16NilVsEmpty: the field-level nil/empty distinction must survive.
func TestSlice16NilVsEmpty(t *testing.T) {
	type rowU struct{ V []uint16 }
	type rowI struct{ V []int16 }
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression} {
		b, err := Marshal(rowU{nil}, opts)
		if err != nil {
			t.Fatal(err)
		}
		var ou rowU
		if err := Unmarshal(b, &ou); err != nil {
			t.Fatal(err)
		}
		if ou.V != nil {
			t.Fatalf("uint16 nil became %v under %v", ou.V, opts)
		}
		b, err = Marshal(rowU{[]uint16{}}, opts)
		if err != nil {
			t.Fatal(err)
		}
		ou = rowU{}
		if err := Unmarshal(b, &ou); err != nil {
			t.Fatal(err)
		}
		if ou.V == nil || len(ou.V) != 0 {
			t.Fatalf("uint16 empty became %v under %v", ou.V, opts)
		}
		bi, err := Marshal(rowI{nil}, opts)
		if err != nil {
			t.Fatal(err)
		}
		var oi rowI
		if err := Unmarshal(bi, &oi); err != nil {
			t.Fatal(err)
		}
		if oi.V != nil {
			t.Fatalf("int16 nil became %v under %v", oi.V, opts)
		}
	}
}

// TestSlice16Skip: an unknown 16-bit field must Skip cleanly.
func TestSlice16Skip(t *testing.T) {
	type full struct {
		V    []uint16
		S    []int16
		Tail int64
	}
	type slim struct{ Tail int64 }
	rng := rand.New(rand.NewSource(3))
	u := make([]uint16, 2000)
	s := make([]int16, 2000)
	for i := range u {
		u[i] = uint16(rng.Uint32())
		s[i] = int16(rng.Uint32())
	}
	for _, opts := range []Options{OptBalanced, OptCompression} {
		b, err := Marshal(full{u, s, 99}, opts)
		if err != nil {
			t.Fatal(err)
		}
		var out slim
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("skip opts=%v: %v", opts, err)
		}
		if out.Tail != 99 {
			t.Fatalf("tail=%d", out.Tail)
		}
	}
}
