package qdf

import (
	"reflect"
	"testing"
)

// nil_coverage_test.go closes the audit gaps around nil-value
// round-tripping. Earlier tests exercised single-nil cases; these
// stress combinations the existing suite missed: maps holding
// multiple nil values, nested maps where the inner map is nil,
// sparse pointer slices, omitempty on nil pointer fields, and the
// nil-vs-empty-slice distinction. Every case runs through all
// encode profiles (Speed / QPack / Dense / DenseQPack / Balanced)
// via runValidity so the shape-intern, MTF, and pair-predictor
// codecs all see the same input.

type nilCovLeaf struct {
	V int    `qdf:"v"`
	S string `qdf:"s"`
}

// TestNilCov_MapStringPtrMultipleNils exercises map[string]*T with
// several nil values interleaved with live ones. The audit flagged
// this as untested — only single-nil maps were covered.
func TestNilCov_MapStringPtrMultipleNils(t *testing.T) {
	in := map[string]*nilCovLeaf{
		"a": {V: 1, S: "one"},
		"b": nil,
		"c": {V: 3, S: "three"},
		"d": nil,
		"e": nil,
	}
	runValidity(t, in)
}

// TestNilCov_NestedMapNilInner: outer map key is present, value is a
// nil inner map. The nil/empty distinction must survive.
func TestNilCov_NestedMapNilInner(t *testing.T) {
	in := map[string]map[string]int{
		"present": {"k": 42, "z": 7},
		"absent":  nil,
		"empty":   {},
	}
	// reflect.DeepEqual treats nil map and empty map as different,
	// so this guards both branches of the decoder.
	runValidity(t, in)
}

// TestNilCov_SparsePointerSlice covers []*T where nil positions are
// interleaved with live ones. Index preservation matters for
// downstream consumers that key by position.
func TestNilCov_SparsePointerSlice(t *testing.T) {
	a, b, c := 10, 20, 30
	in := []*int{nil, &a, nil, &b, nil, &c, nil}
	runValidity(t, in)
}

// TestNilCov_OmitemptyNilPtr — struct with omitempty-tagged nil
// pointer fields. The decode side must not invent zero values; the
// fields stay nil.
func TestNilCov_OmitemptyNilPtr(t *testing.T) {
	type Outer struct {
		Present *int `qdf:"present"`
		Omit    *int `qdf:"omit,omitempty"`
	}
	in := Outer{Present: nil, Omit: nil}
	runValidity(t, in)
}

// TestNilCov_NilVsEmptySliceField documents that QDF — like JSON,
// MessagePack, and most binary codecs — collapses nil and empty
// slices into a single wire form (zero-length array header). Both
// sides decode to a length-0 slice; the nil-vs-empty distinction
// is not part of the wire contract. Pinned so a future change can't
// flip the semantics silently. Callers that need the distinction
// must encode it explicitly (e.g. an extra boolean field).
func TestNilCov_NilVsEmptySliceField(t *testing.T) {
	type S struct {
		NilSlice   []int `qdf:"nil_slice"`
		EmptySlice []int `qdf:"empty_slice"`
	}
	in := S{NilSlice: nil, EmptySlice: []int{}}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			var out S
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			// Both fields decode to a length-0 slice. Whether the
			// header is nil or non-nil is an implementation detail
			// of the reflect decode path and may differ across
			// option bundles.
			if len(out.NilSlice) != 0 {
				t.Fatalf("nil slice grew: %#v", out.NilSlice)
			}
			if len(out.EmptySlice) != 0 {
				t.Fatalf("empty slice grew: %#v", out.EmptySlice)
			}
		})
	}
}

// TestNilCov_MapInt64Ptr — non-string key with nil pointer values.
// Covers the fast-path map[int64]*T branch which the audit noted is
// not exercised in the existing matrix.
func TestNilCov_MapInt64Ptr(t *testing.T) {
	v1, v3 := nilCovLeaf{V: 1}, nilCovLeaf{V: 3}
	in := map[int64]*nilCovLeaf{
		1: &v1,
		2: nil,
		3: &v3,
		4: nil,
	}
	runValidity(t, in)
}

// TestNilCov_SliceOfMapsWithNil covers []map[string]int where some
// outer slots are nil maps. The slice-of-map decoder must not turn
// the nil entries into empty maps.
func TestNilCov_SliceOfMapsWithNil(t *testing.T) {
	in := []map[string]int{
		{"a": 1},
		nil,
		{"b": 2, "c": 3},
		nil,
	}
	runValidity(t, in)
}

// TestNilCov_TypedNilInterface documents the wire semantics of a
// typed-nil pointer stored in an interface{}. Both this and a bare
// nil interface should decode to a bare nil (or at least round-trip
// consistently); this test pins the behaviour so a future codec
// change cannot regress it silently.
func TestNilCov_TypedNilInterface(t *testing.T) {
	var typedNil *string
	in := []any{typedNil, nil, "live"}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			var out []any
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(out) != 3 {
				t.Fatalf("len mismatch: got %d want 3", len(out))
			}
			if out[0] != nil {
				t.Fatalf("typed-nil pointer in any did not round-trip to nil: %#v", out[0])
			}
			if out[1] != nil {
				t.Fatalf("bare nil did not round-trip to nil: %#v", out[1])
			}
			if !reflect.DeepEqual(out[2], "live") {
				t.Fatalf("live value lost: %#v", out[2])
			}
		})
	}
}

// TestNilCov_MapWithEmptyKey checks that "" as a map key (a valid
// but easily-mishandled case) survives the round trip alongside nil
// values. Short keys hit the inline fast path; this also stresses
// the minIntern boundary.
func TestNilCov_MapWithEmptyKey(t *testing.T) {
	in := map[string]*nilCovLeaf{
		"":    {V: 9, S: "empty-key"},
		"non": nil,
	}
	runValidity(t, in)
}

// TestNilCov_DeepPointerToNil documents the (non-)distinction between
// a nil **T and a non-nil **T pointing at a nil *T. Like JSON and
// MessagePack, QDF collapses both to a single tagNil on the wire —
// the receiver gets back a bare nil **T. The test pins this
// behaviour so a future codec change can't silently start preserving
// the distinction (which would be a wire-incompatible change).
func TestNilCov_DeepPointerToNil(t *testing.T) {
	var inner *nilCovLeaf
	type wrap struct {
		PP **nilCovLeaf `qdf:"pp"`
	}
	in := wrap{PP: &inner}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			var out wrap
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.PP != nil {
				t.Fatalf("expected collapsed nil **T; got %#v", out.PP)
			}
		})
	}
}
