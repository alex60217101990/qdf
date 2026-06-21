package cgsample

import (
	"bytes"
	"testing"

	"github.com/alex60217101990/qdf"
)

// A field whose named type has a hand-written codec must route through it in the
// generated code (qdf.EncodeNested/DecodeNested), not be emitted as a bare scalar
// that bypasses the codec. The reflect path always uses the codec, so a bypass
// makes the generated wire DIVERGE from the reflect wire (interop break).
func TestNamedCodecFieldRoutesThroughCodec(t *testing.T) {
	v := GenNamedCodec{Label: "hello", N: 42}

	// Interop invariant: generated MarshalQDF must produce byte-identical wire to
	// the reflect encoder (which routes GenTag through MarshalQDF). A bypass emits
	// the bare string instead of the codec frame → the wires differ.
	gb, err := any(&v).(interface{ MarshalQDF([]byte) ([]byte, error) }).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := qdf.Marshal(&v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gb, rb) {
		t.Fatalf("generated wire diverges from reflect (codec bypassed):\n gen=%q\nrefl=%q", gb, rb)
	}

	// Round-trip via the generated codec preserves the value.
	var out GenNamedCodec
	if _, err := any(&out).(interface{ UnmarshalQDF([]byte) (int, error) }).UnmarshalQDF(gb); err != nil {
		t.Fatal(err)
	}
	if out.Label != "hello" || out.N != 42 {
		t.Fatalf("gen round-trip: got %+v, want {hello 42}", out)
	}
}
