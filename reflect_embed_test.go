package qdf

import (
	"testing"
	"time"
)

// An embedded type that carries its own value codec (time.Time, or a Marshaler)
// must round-trip as a single value field, NOT be structurally flattened into
// zero fields and dropped. Regression for the appendStructFields flatten guard.

type embeddedTime struct {
	time.Time
	Name string
}

func TestEmbeddedTimeRoundTrip(t *testing.T) {
	in := embeddedTime{Time: time.Unix(1700000000, 0).UTC(), Name: "host-1"}
	for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
		b, err := Marshal(in, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal: %v", opt, err)
		}
		var out embeddedTime
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("opt=%v unmarshal: %v", opt, err)
		}
		if !out.Equal(in.Time) {
			t.Fatalf("opt=%v embedded time lost: got %v want %v", opt, out.Time, in.Time)
		}
		if out.Name != in.Name {
			t.Fatalf("opt=%v name: got %q want %q", opt, out.Name, in.Name)
		}
	}
}

// A plain embedded struct (no own codec) must STILL flatten its exported fields
// into the parent — the documented encoding/json idiom. Guards against the time/
// Marshaler exclusion over-firing.
type plainInner struct {
	A int
	B string
}

type embeddedPlain struct {
	plainInner
	C bool
}

func TestEmbeddedPlainStillFlattens(t *testing.T) {
	in := embeddedPlain{plainInner: plainInner{A: 7, B: "ok"}, C: true}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out embeddedPlain
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 7 || out.B != "ok" || !out.C {
		t.Fatalf("flattened embedded struct round-trip: got %+v want %+v", out, in)
	}
}
