package qdf

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// Exotic map-key kinds under OptCanonical: float keys (incl. NaN/Inf), mixed
// dynamic-type interface keys, and distinct NaN bit patterns. These exercise the
// reflect-comparator fallback + the (key,value)-pair gather that avoids a
// MapIndex re-lookup (a NaN key is unfindable by MapIndex). Regression guard for
// the two panics the final review found: b.Int()-on-float (mixed iface) and
// Set-on-zero-Value (NaN key via MapIndex).

func canonStable(t *testing.T, mk func() any) {
	t.Helper()
	const iters = 200
	base, err := Marshal(mk(), OptBalanced|OptCanonical)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for i := 0; i < iters; i++ {
		b, err := Marshal(mk(), OptBalanced|OptCanonical)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if !bytes.Equal(b, base) {
			t.Fatalf("canonical unstable at iter %d", i)
		}
	}
}

func TestCanonicalFloatKeys(t *testing.T) {
	canonStable(t, func() any {
		return map[float64]int{1.5: 1, -2.5: 2, 0.0: 3, math.Inf(1): 4, math.Inf(-1): 5, math.NaN(): 6}
	})
	// round-trip the non-NaN portion (a NaN key can't be looked up post-decode).
	m := map[float64]string{2.5: "a", -1.0: "b", 0.0: "c"}
	b, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	var out map[float64]string
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, m) {
		t.Fatalf("float-key round-trip mismatch: %v vs %v", out, m)
	}
}

func TestCanonicalMixedInterfaceKeys(t *testing.T) {
	canonStable(t, func() any {
		return map[any]int{
			float64(2): 1, int64(7): 2, "str": 3, true: 4, uint32(9): 5, float64(-1.5): 6,
		}
	})
	// round-trips (no NaN key here).
	m := map[any]int{float64(2): 1, int64(7): 2, "str": 3, true: 4}
	b, err := Marshal(m, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	var out map[any]int
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(m) {
		t.Fatalf("mixed-iface round-trip len %d != %d", len(out), len(m))
	}
}

func TestCanonicalDistinctNaNKeys(t *testing.T) {
	// Two coexisting NaN keys with distinct bit patterns (Go allows it: NaN != NaN).
	// They must sort deterministically (by raw bits) so the output is stable.
	mk := func() any {
		m := map[float64]int{}
		m[math.Float64frombits(0x7FF8000000000001)] = 1
		m[math.Float64frombits(0x7FFABCDEF0000000)] = 2
		m[3.0] = 3
		return m
	}
	canonStable(t, mk)
}

func TestCanonicalStructKeys(t *testing.T) {
	type key struct {
		A int64
		B string
	}
	canonStable(t, func() any {
		return map[key]int{{2, "x"}: 1, {1, "y"}: 2, {2, "a"}: 3, {1, "y"}: 4}
	})
}
