package qdf

import "testing"

// Regression: a non-string-keyed map boxed in an any/interface value marshalled
// successfully but failed to Unmarshal into `any` (decodeAny hard-coded string
// keys → ErrTypeMismatch, silent data loss). It now round-trips to map[any]any.
// A typed destination (e.g. *map[int]string) already worked and still does.
func TestDecodeAnyNonStringKeyedMap(t *testing.T) {
	for _, opt := range []Options{OptSpeed, OptBalanced} {
		// small (fixint) keys → uint64 schemaless
		var top any = map[int]string{1: "one", 2: "two"}
		b, err := Marshal(top, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal: %v", opt, err)
		}
		var got any
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("opt=%v unmarshal int-keyed map in any: %v", opt, err)
		}
		m, ok := got.(map[any]any)
		if !ok {
			t.Fatalf("opt=%v want map[any]any got %T", opt, got)
		}
		if len(m) != 2 || m[uint64(1)] != "one" || m[uint64(2)] != "two" {
			t.Fatalf("opt=%v content wrong: %#v", opt, m)
		}

		// negative key → int64 schemaless
		var neg any = map[int]string{-5: "neg"}
		bn, _ := Marshal(neg, opt)
		var gn any
		if err := Unmarshal(bn, &gn); err != nil {
			t.Fatalf("opt=%v unmarshal negative-keyed map: %v", opt, err)
		}
		if gn.(map[any]any)[int64(-5)] != "neg" {
			t.Fatalf("opt=%v negative key lost: %#v", opt, gn)
		}

		// typed destination still works (unchanged behaviour)
		var typed map[int]string
		if err := Unmarshal(b, &typed); err != nil {
			t.Fatalf("opt=%v typed decode regressed: %v", opt, err)
		}
		if typed[1] != "one" || typed[2] != "two" {
			t.Fatalf("opt=%v typed content wrong: %#v", opt, typed)
		}
	}
}

// Regression: a map[any]any with MIXED key types (string + int) must round-trip
// regardless of Go's randomized map iteration order. The decode path used to
// peek only the FIRST key's tag and commit the whole map to one path, so a
// string key sorting first sent an int key into the string reader →
// ErrTypeMismatch ~85% of the time. decodeAny now peeks per key and migrates to
// map[any]any the moment a non-string key appears.
func TestDecodeAnyMixedKeyMap(t *testing.T) {
	for _, opt := range []Options{OptSpeed, OptBalanced} {
		for i := range 50 { // many trials: hit both iteration orders
			var top any = map[any]any{"strkey": "v1", int(42): "v2", "k3": int64(7)}
			b, err := Marshal(top, opt)
			if err != nil {
				t.Fatalf("opt=%v marshal: %v", opt, err)
			}
			var got any
			if err := Unmarshal(b, &got); err != nil {
				t.Fatalf("opt=%v trial %d unmarshal: %v", opt, i, err)
			}
			m, ok := got.(map[any]any)
			if !ok {
				t.Fatalf("opt=%v want map[any]any got %T", opt, got)
			}
			if len(m) != 3 || m["strkey"] != "v1" || m[uint64(42)] != "v2" || m["k3"] != uint64(7) {
				t.Fatalf("opt=%v trial %d content wrong: %#v", opt, i, m)
			}
		}
	}
}

// String-keyed maps in `any` must keep decoding to map[string]any (no regression
// from the new non-string-key branch).
func TestDecodeAnyStringKeyedMapUnchanged(t *testing.T) {
	var top any = map[string]int{"a": 1, "b": 2}
	b, _ := Marshal(top, OptBalanced)
	var got any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("want map[string]any got %T", got)
	}
	if m["a"] != uint64(1) || m["b"] != uint64(2) {
		t.Fatalf("content wrong: %#v", m)
	}
}
