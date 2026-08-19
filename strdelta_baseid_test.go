package qdf

import (
	"fmt"
	"reflect"
	"testing"
)

// strDeltaCheckBaseID is enabled for the WHOLE test binary, not inside one test.
//
// A stale cached id only reaches the wire when a later value happens to equal a
// base the id no longer belongs to, and no single fixture lands there reliably —
// this test's own does not. Running the invariant under every test in the
// package does: with it on, removing either of the two load-bearing
// invalidations in writeStringField trips the check, while the same removals go
// unnoticed by any round-trip assertion including the one below.
//
// Same idiom as strDeltaCount: an init in a _test.go file is compiled only into
// the test binary and reaches every test without a TestMain.
func init() { strDeltaCheckBaseID = true }

type baseIDRow struct {
	S string `qdf:"s"`
	T string `qdf:"t"`
}

// writeStringField caches the intern id of the value sitting in a field's base,
// so a consecutive repeat emits a state-ref without re-hashing a string it has
// just proven equal to that base.
//
// The risk is not the fast path; it is INVALIDATION. Four paths move the base
// without interning the value — the first value of a field, an ineligible one,
// and the two inline codec wins — and each must clear the cached id or a later
// repeat would reference an id belonging to a different string.
//
// This walks a field through every transition that can move the base: first
// value, repeat, delta-eligible change, repeat again, a short (sub-minIntern)
// value, and a repeat after that.
func TestBaseIDCacheSurvivesEveryBaseTransition(t *testing.T) {

	const stem = "com.acme.platform.worker.service."
	rows := make([]baseIDRow, 0, 48)
	rows = append(rows, []baseIDRow{
		{S: stem + "000001", T: "x"}, // first value of each field
		{S: stem + "000001", T: "x"}, // consecutive repeat
		{S: stem + "000002", T: "y"}, // delta-eligible change
		{S: stem + "000002", T: "y"}, // repeat after a delta
		{S: "s", T: "z"},             // short: below minIntern, inline
		{S: "s", T: "z"},             // repeat of an inline value
		{S: stem + "000001", T: "x"}, // back to an id seen long ago
		{S: stem + "000001", T: "x"}, // and repeat it
	}...)
	for i := range 40 {
		rows = append(rows, baseIDRow{
			S: stem + fmt.Sprintf("%06d", i%3),
			T: fmt.Sprintf("t%02d", i%2),
		})
	}
	for _, o := range []struct {
		name string
		opts Options
	}{
		{"balanced", OptBalanced},
		{"balanced+alpha", OptBalanced | OptStringAlphabet},
		{"compression", OptCompression},
		{"dense only", OptDense},
	} {
		b, err := Marshal(rows, o.opts)
		if err != nil {
			t.Fatalf("%s: %v", o.name, err)
		}
		var got []baseIDRow
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: decode: %v", o.name, err)
		}
		if !reflect.DeepEqual(got, rows) {
			for i := range rows {
				if got[i] != rows[i] {
					t.Fatalf("%s row %d: got %+v want %+v — a cached intern id outlived "+
						"the value it belonged to", o.name, i, got[i], rows[i])
				}
			}
		}
		// Byte stability: two encodes of the same value must agree, or the cache
		// has made the output depend on encoder history.
		again, err := Marshal(rows, o.opts)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != string(again) {
			t.Errorf("%s: two encodes of the same value differ (%d vs %d bytes)",
				o.name, len(b), len(again))
		}
	}
}
