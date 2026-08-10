package qdf

import (
	"reflect"
	"testing"
)

type sdBaseA struct {
	X string `qdf:"x"`
}

type sdBaseB struct {
	Y string `qdf:"y"`
}

// Two struct types must not share one base slice. If they did, a nested struct
// would delta-code against its parent's previous field value — which decodes
// without error, because the types still line up.
func TestStrDeltaBasesArePerType(t *testing.T) {
	tdA, err := descOf(reflect.TypeFor[sdBaseA]())
	if err != nil {
		t.Fatal(err)
	}
	tdB, err := descOf(reflect.TypeFor[sdBaseB]())
	if err != nil {
		t.Fatal(err)
	}
	st := newEncState()
	bA, _ := st.strDeltaBases(tdA, 1)
	bA[0] = "from-a"
	bB, _ := st.strDeltaBases(tdB, 1)
	if got := bB[0]; got == "from-a" {
		t.Fatal("two types share one base slice")
	}
	bA2, _ := st.strDeltaBases(tdA, 1)
	if got := bA2[0]; got != "from-a" {
		t.Fatalf("type a's base did not survive a lookup of type b: %q", got)
	}
}

// The state is pooled. A base surviving reset would delta-code the first row of
// the next payload against a string from the previous one — and the decoder,
// starting with an empty base, would reconstruct something else.
func TestStrDeltaBasesResetWithState(t *testing.T) {
	td, err := descOf(reflect.TypeFor[sdBaseA]())
	if err != nil {
		t.Fatal(err)
	}
	st := newEncState()
	b0, _ := st.strDeltaBases(td, 1)
	b0[0] = "stale"
	st.reset()
	b1, _ := st.strDeltaBases(td, 1)
	if got := b1[0]; got != "" {
		t.Fatalf("base survived reset: %q", got)
	}
}

func TestStrDeltaDecBasesResetWithState(t *testing.T) {
	d := newDecState()
	d.strDeltaBases(1, 2)[0] = "stale"
	d.reset()
	if got := d.strDeltaBases(1, 2)[0]; got != "" {
		t.Fatalf("decoder base survived reset: %q", got)
	}
}

// A wider shape must grow its slice rather than index out of range.
func TestStrDeltaDecBasesGrow(t *testing.T) {
	d := newDecState()
	d.strDeltaBases(3, 2)[1] = "two"
	b := d.strDeltaBases(3, 5)
	if len(b) < 5 {
		t.Fatalf("slice did not grow: len=%d", len(b))
	}
	if b[1] != "two" {
		t.Fatalf("growing dropped an existing base: %q", b[1])
	}
}
