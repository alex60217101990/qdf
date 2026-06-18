package qdf

import (
	"reflect"
	"testing"
)

// nestedAnyTree builds a map[string]any with many nested map[string]any and
// []any containers — the shape produced by json.Unmarshal into map[string]any.
// Each nested container is the unit that the old encodeReflect allocated a
// reflect.New for, so the encode-allocation count scales with the container
// count here.
func nestedAnyTree(maps int) map[string]any {
	root := make(map[string]any, maps)
	for i := range maps {
		row := map[string]any{
			"name":    "service-name-that-is-fairly-long",
			"enabled": true,
			"count":   float64(i),
			"tags":    []any{"a", "b", "c", "d"},
			"meta":    map[string]any{"k1": "v1", "k2": float64(i), "k3": false},
		}
		root[string(rune('A'+i%26))+string(rune('0'+i%10))+itoaSmall(i)] = row
	}
	return root
}

// TestEncodeReflectAnyFastAllocs guards the encodeReflect fast paths for
// map[string]any and []any: encoding a deeply nested any-tree must NOT allocate
// per nested container (the old reflect.New-per-value path did). The bound is
// well below the container count — a regression that reintroduces the per-value
// allocation would blow straight through it.
func TestEncodeReflectAnyFastAllocs(t *testing.T) {
	const maps = 100 // 100 rows × (row map + meta map + tags slice) ≈ 300 containers
	v := nestedAnyTree(maps)

	for _, tc := range []struct {
		name string
		opts Options
	}{
		{"Speed", OptSpeed},
		{"Balanced", OptBalanced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Warm the pools so AllocsPerRun sees steady state.
			if _, err := Marshal(v, tc.opts); err != nil {
				t.Fatal(err)
			}
			allocs := testing.AllocsPerRun(20, func() {
				b, err := Marshal(v, tc.opts)
				if err != nil {
					t.Fatal(err)
				}
				_ = b
			})
			// Marshal returns a fresh buffer (≥1 alloc) plus a small fixed amount
			// of bookkeeping; the per-container reflect.New (≈300 here) must be
			// gone. 64 leaves generous headroom while still failing hard on the
			// old O(containers) behavior.
			if allocs > 64 {
				t.Fatalf("%s: Marshal of %d-container any-tree did %.0f allocs/op; "+
					"expected the per-value reflect.New to be eliminated (want ≤64)", tc.name, maps, allocs)
			}
		})
	}
}

// TestEncodeReflectAnyFastRoundTrip is the correctness gate: the fast paths must
// produce bytes that decode back to an equal value, across the option matrix.
func TestEncodeReflectAnyFastRoundTrip(t *testing.T) {
	v := nestedAnyTree(40)
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression, OptBalanced | OptCanonical} {
		b, err := Marshal(v, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		var got map[string]any
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("opts=%d unmarshal: %v", opts, err)
		}
		if !reflect.DeepEqual(v, got) {
			t.Fatalf("opts=%d round-trip mismatch", opts)
		}
	}
}

// TestEncodeSliceAnyProbeGrow exercises encodeSliceAny's probe-and-grow branch
// (n > 32): a large []any of mixed dynamic values must round-trip exactly, and
// the buffer pre-sizing must not corrupt the bytes.
func TestEncodeSliceAnyProbeGrow(t *testing.T) {
	const n = 2000
	big := make([]any, n)
	for i := range big {
		switch i % 4 {
		case 0:
			big[i] = "string-value-" + itoaSmall(i)
		case 1:
			big[i] = float64(i)
		case 2:
			big[i] = map[string]any{"i": float64(i), "ok": true}
		case 3:
			big[i] = []any{"x", float64(i), false}
		}
	}
	v := map[string]any{"items": big}
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression} {
		b, err := Marshal(v, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		var got map[string]any
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("opts=%d unmarshal: %v", opts, err)
		}
		if !reflect.DeepEqual(v, got) {
			t.Fatalf("opts=%d large []any round-trip mismatch", opts)
		}
	}
}
