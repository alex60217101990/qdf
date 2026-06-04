package qdf

import "testing"

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
