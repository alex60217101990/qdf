package qdf

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// TestCanonicalizeFloatHelpers exercises the float-normalization helpers
// directly (the encode choke-point wiring lands in a later phase).
func TestCanonicalizeFloatHelpers(t *testing.T) {
	negZero := math.Copysign(0, -1)
	if got := canonicalizeFloat64(negZero); math.Float64bits(got) != 0 {
		t.Fatalf("canonicalizeFloat64(-0.0) bits = %#x, want 0", math.Float64bits(got))
	}
	if got := canonicalizeFloat64(math.Float64frombits(0x7FFABCDEF0000001)); math.Float64bits(got) != 0x7FF8000000000000 {
		t.Fatalf("canonicalizeFloat64(NaN) bits = %#x", math.Float64bits(got))
	}
	if got := canonicalizeFloat64(3.5); got != 3.5 {
		t.Fatalf("canonicalizeFloat64(3.5) = %v", got)
	}
	if got := canonicalizeFloat32(float32(negZero)); math.Float32bits(got) != 0 {
		t.Fatalf("canonicalizeFloat32(-0.0) bits = %#x", math.Float32bits(got))
	}
	if got := canonicalizeFloat32(math.Float32frombits(0x7FABCDEF)); math.Float32bits(got) != 0x7FC00000 {
		t.Fatalf("canonicalizeFloat32(NaN) bits = %#x", math.Float32bits(got))
	}
	if got := canonicalizeFloat32Bits(uint64(math.Float32bits(float32(negZero)))); got != 0 {
		t.Fatalf("canonicalizeFloat32Bits(-0.0) = %#x", got)
	}
	if got := canonicalizeFloat32Bits(0x7FABCDEF); got != 0x7FC00000 {
		t.Fatalf("canonicalizeFloat32Bits(NaN) = %#x", got)
	}
	if got := canonicalizeFloat32Bits(uint64(math.Float32bits(1.25))); got != uint64(math.Float32bits(1.25)) {
		t.Fatalf("canonicalizeFloat32Bits(1.25) = %#x", got)
	}
}

func TestOptCanonicalExists(t *testing.T) {
	if OptCanonical == 0 {
		t.Fatal("OptCanonical not defined")
	}
	o := OptBalanced | OptCanonical
	if !o.Has(OptCanonical) {
		t.Fatal("Has(OptCanonical) false")
	}
	// Must not collide with any existing option bit.
	for _, x := range []Options{
		OptDense, OptQPack, OptShapeIntern, OptPairPred, OptMTF,
		OptGorillaFloat, OptRANS, OptColumnIndex, OptFSST, OptMapShape,
		OptDeltaNoBaseFingerprint,
	} {
		if OptCanonical == x {
			t.Fatalf("OptCanonical collides with %v", x)
		}
	}
}

// TestCanonicalMapStableReflect covers the reflect encodeMap sorted-key emit.
// It must use a map type WITHOUT a generated fast path so it actually exercises
// the reflect path: map[string]int and the scalar value kinds are generated
// (maps_fast_generated.go, covered by the generated-path task). A []int64 value
// has no generated entry, so this routes through reflect encodeMap.
func TestCanonicalMapStableReflect(t *testing.T) {
	m := map[string][]int64{"z": {1}, "a": {2}, "m": {3}, "b": {4}, "q": {5}}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		// rebuild the map so Go's iteration order differs
		m2 := map[string][]int64{}
		for k, v := range m {
			m2[k] = v
		}
		b, err := Marshal(m2, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical map bytes unstable at iter %d", i)
		}
	}
	// round-trips
	var out map[string][]int64
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

// TestCanonicalMapStableInt64Key covers an integer-keyed map on the reflect
// path. map[int64][]int64 has no generated fast path (only int64-keyed
// string/int64/any are generated), so this exercises gatherIntKeys + SetInt.
func TestCanonicalMapStableInt64Key(t *testing.T) {
	m := map[int64][]int64{9: {1}, 1: {2}, 5: {3}, 3: {4}, 7: {5}}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		m2 := map[int64][]int64{}
		for k, v := range m {
			m2[k] = v
		}
		b, err := Marshal(m2, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical int64-key map unstable at iter %d", i)
		}
	}
	var out map[int64][]int64
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

// TestCanonicalMapStableUintKey covers an unsigned-int-keyed reflect map
// (gatherUintKeys + SetUint). map[uint32][]int64 has no generated fast path.
func TestCanonicalMapStableUintKey(t *testing.T) {
	m := map[uint32][]int64{9: {1}, 1: {2}, 5: {3}, 3: {4}, 7: {5}}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		m2 := map[uint32][]int64{}
		for k, v := range m {
			m2[k] = v
		}
		b, err := Marshal(m2, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical uint-key map unstable at iter %d", i)
		}
	}
	var out map[uint32][]int64
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

// TestCanonicalMapStableBoolKey covers the no-sort false-then-true bool emit.
func TestCanonicalMapStableBoolKey(t *testing.T) {
	m := map[bool][]int64{true: {1, 2}, false: {3, 4}}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		m2 := map[bool][]int64{}
		for k, v := range m {
			m2[k] = v
		}
		b, err := Marshal(m2, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical bool-key map unstable at iter %d", i)
		}
	}
	var out map[bool][]int64
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

func TestCanonicalMapStableStructValue(t *testing.T) {
	type V struct {
		A int64
		B string
	}
	m := map[string]V{
		"z": {A: 1, B: "one"},
		"a": {A: 2, B: "two"},
		"m": {A: 3, B: "three"},
		"b": {A: 4, B: "four"},
	}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		m2 := map[string]V{}
		for k, v := range m {
			m2[k] = v
		}
		b, err := Marshal(m2, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical struct-value map unstable at iter %d", i)
		}
	}
	var out map[string]V
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

// TestCanonicalMapFreeIdentical guards the zero-overhead contract: a map-free,
// normal-float value must serialize byte-identically with and without
// OptCanonical, across every tier (canonical is a separate branch).
// TestCanonicalMapStableGenerated covers the generated typed-map fast paths
// (map[string]string, map[int64]string, map[string]int64, map[string]int). These
// hit installMapFastPath, not the reflect encodeMap, so the canonical sort must
// be wired into the generator template, not the reflect path.
func TestCanonicalMapStableGenerated(t *testing.T) {
	for _, mk := range []func() any{
		func() any { return map[string]string{"z": "1", "a": "2", "k": "3"} },
		func() any { return map[int64]string{9: "x", 1: "y", 5: "z"} },
		func() any { return map[string]int64{"b": 2, "a": 1, "c": 3} },
		func() any { return map[string]int{"z": 1, "a": 2, "m": 3, "b": 4, "q": 5} },
		func() any { return map[uint64]string{9: "x", 1: "y", 5: "z"} },
	} {
		base, err := Marshal(mk(), OptBalanced|OptCanonical|OptDense)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 100; i++ {
			b, err := Marshal(mk(), OptBalanced|OptCanonical|OptDense)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, base) {
				t.Fatalf("generated canonical unstable")
			}
		}
		// round-trip a representative one
		_ = base
	}
	// explicit round-trip on the int-valued string map
	first, err := Marshal(map[string]int{"z": 1, "a": 2, "m": 3}, OptBalanced|OptCanonical|OptDense)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]int
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, map[string]int{"z": 1, "a": 2, "m": 3}) {
		t.Fatal("round-trip mismatch")
	}
}

// TestCanonicalFloatNormalize verifies -0.0 == +0.0 and distinct NaNs serialize
// identically under OptCanonical, across scalar, slice, and columnar-struct-batch
// shapes for both float64 and float32.
func TestCanonicalFloatNormalize(t *testing.T) {
	negZero := math.Copysign(0, -1)
	nan1 := math.Float64frombits(0x7FF8000000000001)
	nan2 := math.Float64frombits(0x7FFABCDEF0000000)
	enc := func(v any) []byte {
		b, err := Marshal(v, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	// scalar f64
	if !bytes.Equal(enc(negZero), enc(0.0)) {
		t.Error("scalar -0.0 != +0.0 under canonical")
	}
	if !bytes.Equal(enc(nan1), enc(nan2)) {
		t.Error("scalar distinct NaNs differ under canonical")
	}
	// scalar f32
	nz32 := float32(math.Copysign(0, -1))
	n1f := math.Float32frombits(0x7FC00001)
	n2f := math.Float32frombits(0x7FABCDEF)
	if !bytes.Equal(enc(nz32), enc(float32(0.0))) {
		t.Error("scalar f32 -0.0 != +0.0 under canonical")
	}
	if !bytes.Equal(enc(n1f), enc(n2f)) {
		t.Error("scalar f32 distinct NaNs differ under canonical")
	}
	// slice f64
	if !bytes.Equal(enc([]float64{negZero, nan1}), enc([]float64{0.0, nan2})) {
		t.Error("[]float64 not normalized")
	}
	// slice f32
	if !bytes.Equal(enc([]float32{nz32, n1f}), enc([]float32{0.0, n2f})) {
		t.Error("[]float32 not normalized")
	}
	// columnar struct batch
	type row struct {
		F float64
		G float32
	}
	a := make([]row, 20)
	b := make([]row, 20)
	for i := range a {
		a[i] = row{F: negZero, G: nz32}
		b[i] = row{F: 0.0, G: 0.0}
	}
	if !bytes.Equal(enc(a), enc(b)) {
		t.Error("columnar float column not normalized")
	}
	// columnar NaN normalization
	a2 := make([]row, 20)
	b2 := make([]row, 20)
	for i := range a2 {
		a2[i] = row{F: nan1, G: n1f}
		b2[i] = row{F: nan2, G: n2f}
	}
	if !bytes.Equal(enc(a2), enc(b2)) {
		t.Error("columnar float NaN column not normalized")
	}
}

func TestCanonicalMapFreeIdentical(t *testing.T) {
	type S struct {
		ID   int64
		Name string
		Vals []int64
		F    float64
	}
	v := S{ID: 7, Name: "hi", Vals: []int64{1, 2, 3}, F: 3.14}
	for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
		plain, err := Marshal(v, opt)
		if err != nil {
			t.Fatal(err)
		}
		canon, err := Marshal(v, opt|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(plain, canon) {
			t.Fatalf("opt %v: canonical changed map-free bytes", opt)
		}
	}
}
