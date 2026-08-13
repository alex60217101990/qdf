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

	// Interop invariant: generated MarshalQDF must produce the same BODY as the
	// reflect encoder (which routes GenTag through MarshalQDF). A bypass emits
	// the bare string instead of the codec frame → the bodies differ.
	//
	// Bodies, not whole wires. The two entry points frame differently by design:
	// MarshalQDF builds its own encoder at qdf.Fast and ignores Options, while
	// qdf.Marshal states the mode it was asked for. They agreed only while the
	// reflect path forced a Fast header onto every top-level Marshaler, which
	// also cost generated structs their rANS framing. The header is five bytes
	// and is not what this test is about.
	gb, err := any(&v).(interface{ MarshalQDF([]byte) ([]byte, error) }).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := qdf.Marshal(&v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	const hdr = 5
	if len(gb) < hdr || len(rb) < hdr {
		t.Fatalf("wire shorter than a header: gen=%d refl=%d", len(gb), len(rb))
	}
	if !bytes.Equal(gb[hdr:], rb[hdr:]) {
		t.Fatalf("generated body diverges from reflect (codec bypassed):\n gen=%q\nrefl=%q",
			gb[hdr:], rb[hdr:])
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
