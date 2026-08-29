package qdf

import (
	"bytes"
	"maps"
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
	for i := range 200 {
		// rebuild the map so Go's iteration order differs
		m2 := map[string][]int64{}
		maps.Copy(m2, m)
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
	for i := range 200 {
		m2 := map[int64][]int64{}
		maps.Copy(m2, m)
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
	for i := range 200 {
		m2 := map[uint32][]int64{}
		maps.Copy(m2, m)
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
	for i := range 100 {
		m2 := map[bool][]int64{}
		maps.Copy(m2, m)
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
	for i := range 200 {
		m2 := map[string]V{}
		maps.Copy(m2, m)
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
		for range 100 {
			b, err := Marshal(mk(), OptBalanced|OptCanonical|OptDense)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, base) {
				t.Fatal("generated canonical unstable")
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

// TestCanonicalDiffStable verifies a map-bearing struct's Diff(..., OptCanonical)
// is byte-stable across map rebuilds (Go randomizes iteration), and still applies
// correctly. The map-patch update/add and deletion-tombstone emit passes must be
// sorted under canonical.
func TestCanonicalDiffStable(t *testing.T) {
	type S struct{ M map[string]int }
	old := S{M: map[string]int{"a": 1, "b": 2, "c": 3}}
	// new value: "b" changes, "d" added, "c" deleted, "e"/"f" added — both
	// emit passes (updates/adds and deletions) carry several keys to sort.
	mk := func() S {
		m := map[string]int{}
		maps.Copy(m, map[string]int{"a": 1, "b": 9, "d": 4, "e": 5, "f": 6})
		return S{M: m}
	}
	base, err := Diff(old, mk(), OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		p, err := Diff(old, mk(), OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(p, base) {
			t.Fatalf("canonical diff unstable at iter %d", i)
		}
	}
	got := old
	got.M = map[string]int{"a": 1, "b": 2, "c": 3}
	if err := Apply(&got, base); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, mk()) {
		t.Fatalf("apply mismatch: got %v want %v", got, mk())
	}
}

// TestCanonicalDiffFloat32Columnar exercises the delta columnar float32 gather:
// a -0.0 cell and a +0.0 cell must diff to the same patch under canonical (the
// raw-bits uint64 codec never re-floats, so the gather must normalize).
func TestCanonicalDiffFloat32Columnar(t *testing.T) {
	type Row struct {
		A int64
		G float32
	}
	// n=64 with only 4 changed float32 cells → sparse column mode (the gather at
	// delta_columnar.go that emits raw float32 bits, which must normalize).
	const n = 64
	old := make([]Row, n)
	for i := range old {
		old[i] = Row{A: int64(i), G: 5.0}
	}
	// newA flips 4 cells to -0.0; newB flips the SAME 4 cells to +0.0 (both differ
	// from old's 5.0 → same changed-row set). Logically equal under canonical →
	// identical patches.
	newA := make([]Row, n)
	newB := make([]Row, n)
	copy(newA, old)
	copy(newB, old)
	nz := float32(math.Copysign(0, -1))
	for _, i := range []int{3, 17, 40, 58} {
		newA[i].G = nz
		newB[i].G = 0.0
	}
	pa, err := Diff(old, newA, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := Diff(old, newB, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pa, pb) {
		t.Fatal("float32 columnar diff: -0.0 patch != +0.0 patch under canonical")
	}
}

// TestCanonicalNestedStable encodes a deeply nested value (maps within maps,
// float slices with -0.0) many times and asserts byte-stability — Go randomizes
// map iteration per range, so 300 iterations exercises the perturbation.
func TestCanonicalNestedStable(t *testing.T) {
	type Inner struct {
		Tags map[string]int64
		Vals []float64
	}
	type Outer struct {
		ID       string
		Sections map[string]Inner
		Nums     map[int64]string
	}
	mk := func() Outer {
		return Outer{
			ID: "x",
			Sections: map[string]Inner{
				"s2": {Tags: map[string]int64{"b": 2, "a": 1}, Vals: []float64{math.Copysign(0, -1), 1.5}},
				"s1": {Tags: map[string]int64{"z": 9, "k": 8}},
			},
			Nums: map[int64]string{3: "c", 1: "a", 2: "b"},
		}
	}
	base, err := Marshal(mk(), OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 300 {
		b, err := Marshal(mk(), OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, base) {
			t.Fatalf("nested canonical unstable at %d", i)
		}
	}
}

// TestCanonicalIdempotent verifies Marshal(Unmarshal(Marshal(v,C)),C) ==
// Marshal(v,C): a value round-tripped through decode and re-encoded canonically
// yields the same bytes (the normalization is a fixed point).
func TestCanonicalIdempotent(t *testing.T) {
	type S struct{ M map[string]float64 }
	v := S{M: map[string]float64{"a": math.Copysign(0, -1), "b": 1.5, "c": 2.5}}
	b1, err := Marshal(v, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	var d S
	if err := Unmarshal(b1, &d); err != nil {
		t.Fatal(err)
	}
	b2, err := Marshal(d, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("canonical encoding not idempotent")
	}
}

// TestCanonicalMapAllocZeroOverhead checks the sorted-key map emit adds no
// steady-state allocations over the default map emit: the key-sort scratch is
// pooled on the encoder state, so once warmed it allocates nothing extra. Skipped
// under -race (sync.Pool churn instrumentation inflates counts — see assertAllocs).
func TestCanonicalMapAllocZeroOverhead(t *testing.T) {
	if raceEnabled {
		t.Skip("alloc budgets are not measured under -race (sync.Pool churn instrumentation)")
	}
	m := map[string]int64{"delta": 4, "alpha": 1, "gamma": 3, "beta": 2, "epsilon": 5}
	dst := make([]byte, 0, 256)
	plainFn := func() {
		var err error
		dst, err = AppendMarshal(dst[:0], m, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
	}
	canonFn := func() {
		var err error
		dst, err = AppendMarshal(dst[:0], m, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Warm both paths so the pooled scratch is sized before measuring.
	for range 50 {
		plainFn()
		canonFn()
	}
	plain := testing.AllocsPerRun(200, plainFn)
	canon := testing.AllocsPerRun(200, canonFn)
	t.Logf("map encode allocs: default=%.1f canonical=%.1f", plain, canon)
	if canon > plain {
		t.Fatalf("canonical map encode adds allocs: default=%.1f canonical=%.1f", plain, canon)
	}
}

// FuzzCanonicalStable marshals a randomly-generated deep-nested value twice under
// OptCanonical and asserts the two encodings are byte-identical and never panic.
// Any non-determinism it finds is a real canonical bug (a map-range or other
// order-dependent emit that escaped normalization), not a test weakness.
func FuzzCanonicalStable(f *testing.F) {
	for _, s := range []int64{1, 2, 7, 42, 1000, -5} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		v := gen[dnTop](seed)
		b1, err := Marshal(v, OptBalanced|OptCanonical)
		if err != nil {
			return
		}
		b2, err := Marshal(v, OptBalanced|OptCanonical)
		if err != nil {
			t.Fatalf("second canonical marshal errored for seed %d: %v", seed, err)
		}
		if !bytes.Equal(b1, b2) {
			t.Fatalf("canonical unstable for seed %d", seed)
		}
	})
}
