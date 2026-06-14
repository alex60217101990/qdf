package qdf

import (
	"bytes"
	"errors"
	"testing"
)

// TestStreamRejectsRANSHeader: a stream whose header carries FlagRANS (a
// whole-payload post-pass that does not belong in a frame stream) must be
// rejected and latch the decoder broken — not silently swap the shared window
// buffer for the rANS-decoded body and desync framing. The body here is a
// genuinely valid rANS frame (produced by Marshal with OptCompression), so the
// rANS decode would succeed and the desync would be silent without the guard.
func TestStreamRejectsRANSHeader(t *testing.T) {
	type rec struct {
		A, B, C, D, E, F, G, H int64
		Name                   string `qdf:"name"`
	}
	in := make([]rec, 4000)
	for i := range in {
		in[i].Name = "a low-entropy repeated value that rANS compresses well"
	}
	buf, err := Marshal(&in, OptCompression)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if buf[4]&FlagRANS == 0 {
		t.Skip("payload did not trigger rANS; cannot construct the hostile stream")
	}

	sd := NewStreamDecoder(bytes.NewReader(buf))
	var out []rec
	if err := sd.Decode(&out); !errors.Is(err, ErrStreamBadFlags) {
		t.Fatalf("Decode: got %v, want ErrStreamBadFlags", err)
	}
	// Latched broken: a second Decode must refuse cleanly.
	if err := sd.Decode(&out); !errors.Is(err, ErrStreamDecoderBroken) {
		t.Fatalf("second Decode: got %v, want ErrStreamDecoderBroken", err)
	}
}
