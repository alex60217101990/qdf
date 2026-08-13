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

	// A bypass emits the bare string where the codec's frame belongs, and the
	// way to see that is to encode the same value under TWO DIFFERENT MODES and
	// compare the bodies.
	//
	// That is the whole mechanism, and it is easy to break by "improving" it.
	// MarshalQDF builds its own encoder at qdf.Fast; qdf.Marshal here runs
	// OptBalanced — and note it also reaches generated code, since Marshal
	// dispatches to EncodeQDF for an EncoderMarshaler, so this is not a reflect
	// encode despite appearances. A value that went through the codec is
	// mode-invariant — the codec writes its own bytes and never consults
	// Options — so the two agree. A bypassed bare string is mode-SENSITIVE:
	// Balanced interns it, Fast does not, and the bodies diverge.
	//
	// Levelling both sides to OptSpeed makes the comparison look tidier and
	// blinds it completely: a bypassed string then matches too. Measured — with
	// the bypass reintroduced and both sides at OptSpeed, this assertion passes
	// and only the round-trip below notices.
	//
	// Bodies, not whole wires: the two entry points frame differently by design,
	// and the five-byte header is not what this test is about.
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
		t.Fatalf("generated body diverges from the Balanced encode (codec bypassed):\n gen=%q\nbal=%q",
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
