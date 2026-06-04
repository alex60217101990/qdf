package qdf

import (
	"bytes"
	"errors"
	"testing"
)

type streamCycleNode struct {
	Name string           `qdf:"name"`
	Self *streamCycleNode `qdf:"self"`
}

// TestStream_BrokenAfterMidMessageError: a mid-message encode failure advances
// cross-message encoder state (interned strings / shapes) that the decoder
// never saw; emitting further frames would silently desync. The stream must
// refuse further Encode with ErrStreamBroken instead.
func TestStream_BrokenAfterMidMessageError(t *testing.T) {
	var buf bytes.Buffer
	s := NewStreamEncoder(&buf, Dense)
	if err := s.Encode(map[string]string{"a": "1"}); err != nil {
		t.Fatalf("first encode: %v", err)
	}
	cyc := &streamCycleNode{Name: "loop"}
	cyc.Self = cyc // self-cycle → encode error after some state mutates
	if err := s.Encode(cyc); err == nil {
		t.Fatal("expected an error encoding a cyclic value")
	}
	if err := s.Encode(map[string]string{"b": "2"}); !errors.Is(err, ErrStreamBroken) {
		t.Fatalf("after mid-message error, Encode = %v, want ErrStreamBroken", err)
	}
}

// TestGorilla_HugeN_NoOOM: a tiny Gorilla payload claiming a huge element
// count must error before allocating, not attempt a multi-GB make([]float64).
func TestGorilla_HugeN_NoOOM(t *testing.T) {
	for _, kind := range []byte{qpackKindFloat64, qpackKindFloat32} {
		buf := []byte{kind}
		buf = appendUvarint(buf, 1<<34) // ~17 billion elements
		d := &Decoder{buf: buf}
		var err error
		if kind == qpackKindFloat64 {
			_, err = d.readPackedGorillaFloat64Slice()
		} else {
			_, err = d.readPackedGorillaFloat32Slice()
		}
		if err == nil {
			t.Fatalf("kind %#x: expected error on huge-n Gorilla payload, got nil", kind)
		}
	}
}

// TestGorilla_RoundTrip_Smooth confirms the bound does not break valid decode.
func TestGorilla_RoundTrip_Smooth(t *testing.T) {
	in := make([]float64, 200)
	for i := range in {
		in[i] = 100.0 + float64(i)*0.5
	}
	type wrap struct {
		F []float64 `qdf:"f"`
	}
	b, err := Marshal(wrap{F: in}, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var out wrap
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.F) != len(in) {
		t.Fatalf("len %d != %d", len(out.F), len(in))
	}
	for i := range in {
		if out.F[i] != in[i] {
			t.Fatalf("idx %d: %v != %v", i, out.F[i], in[i])
		}
	}
}
