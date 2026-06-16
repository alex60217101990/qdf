package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

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

func TestCanonicalMapStableReflect(t *testing.T) {
	m := map[string]int{"z": 1, "a": 2, "m": 3, "b": 4, "q": 5}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		// rebuild the map so Go's iteration order differs
		m2 := map[string]int{}
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
	var out map[string]int
	if err := Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatal("round-trip mismatch")
	}
}

func TestCanonicalMapStableInt64Key(t *testing.T) {
	m := map[int64]string{9: "x", 1: "y", 5: "z", 3: "w", 7: "v"}
	first, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		m2 := map[int64]string{}
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
	var out map[int64]string
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
