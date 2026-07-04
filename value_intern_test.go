package qdf

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// TestValueInternRoundTrip exercises the decode-side boxed-any value cache
// (decState.boxValues): a map/slice whose string values REPEAT heavily decodes
// via state references, so the box cache fires. The decoded structure must
// equal the source exactly — sharing one immutable box across many slots must
// never corrupt a value.
func TestValueInternRoundTrip(t *testing.T) {
	// Heavy repetition: a handful of distinct values across many keys, plus a
	// nested slice reusing the same values, so most occurrences are references.
	cats := []string{"info", "warning", "error", "info", "info", "error"}
	src := map[string]any{}
	for i := range 200 {
		src[fmt.Sprintf("k%03d", i)] = cats[i%len(cats)]
	}
	src["nested"] = []any{"info", "error", "info", "warning", "info"}
	src["mixed"] = map[string]any{"level": "error", "again": "error", "n": float64(42)}

	for _, opt := range []Options{OptSpeed, OptBalanced, OptBalanced | OptShapeIntern} {
		t.Run(fmt.Sprintf("opt=%d", opt), func(t *testing.T) {
			data, err := Marshal(src, opt)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, src) {
				t.Fatalf("round-trip mismatch:\n got=%#v\nwant=%#v", got, src)
			}
			// Independence: mutating one slot's value must not affect another —
			// interface values are immutable, so replacing a slot is safe even if
			// the box was shared. Reassign and re-check a sibling.
			got["k000"] = "changed"
			if got["k006"] != "info" {
				t.Fatalf("shared box corrupted a sibling: k006=%v", got["k006"])
			}
		})
	}
}

// TestValueInternRepeatedDecodes verifies the box cache resets correctly
// between decodes on a pooled decoder: decoding two different payloads back to
// back must not leak a box from the first into the second.
func TestValueInternRepeatedDecodes(t *testing.T) {
	a := map[string]any{"x": "alpha", "y": "alpha", "z": "alpha"}
	b := map[string]any{"x": "beta", "y": "beta", "z": "beta"}
	da, _ := Marshal(a, OptBalanced)
	db, _ := Marshal(b, OptBalanced)
	for range 50 {
		var ga, gb map[string]any
		if err := Unmarshal(da, &ga); err != nil {
			t.Fatal(err)
		}
		if err := Unmarshal(db, &gb); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ga, a) || !reflect.DeepEqual(gb, b) {
			t.Fatalf("cross-decode leak: ga=%v gb=%v", ga, gb)
		}
	}
}

// TestValueInternArena exercises the boxed-any value cache under WithArena: the
// cached box holds an arena-materialized string shared across many map/slice
// slots, so a repeated value must round-trip exactly. Also guards the arena is
// threaded correctly through getBoxStr (readStringRefAny passes d.arena).
func TestValueInternArena(t *testing.T) {
	cats := []string{"aaaa", "bbbb", "aaaa", "cccc", "aaaa", "bbbb"}
	src := map[string]any{}
	for i := range 300 {
		src[fmt.Sprintf("k%03d", i)] = cats[i%len(cats)]
	}
	src["nested"] = []any{"aaaa", "cccc", "aaaa", "aaaa"}
	data, err := Marshal(src, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		a := NewArena()
		var got map[string]any
		if err := Unmarshal(data, &got, WithArena(a)); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, src) {
			t.Fatalf("arena round-trip mismatch:\n got=%#v\nwant=%#v", got, src)
		}
	}
}

// TestZeroTimeShared verifies the shared zero-time box: a map/slice with many
// zero time.Time values (unset fields) round-trips exactly while boxing the
// zero once. Non-zero times are unaffected.
func TestZeroTimeShared(t *testing.T) {
	src := map[string]any{
		"created": time.Unix(1_700_000_000, 0).UTC(),
		"deleted": time.Time{},
		"expires": time.Time{},
		"seen":    time.Time{},
		"updated": time.Unix(1_700_000_500, 7).UTC(),
	}
	src["events"] = []any{time.Time{}, time.Unix(1_700_000_001, 0).UTC(), time.Time{}, time.Time{}}
	for _, opt := range []Options{OptSpeed, OptBalanced} {
		data, err := Marshal(src, opt)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, src) {
			t.Fatalf("opt=%d zero-time round-trip mismatch:\n got=%#v\nwant=%#v", opt, got, src)
		}
	}
}
