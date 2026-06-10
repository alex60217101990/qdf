package qdf

import (
	"bytes"
	"errors"
	"testing"
)

// Regression: a corrupt stream HEADER must latch the decoder broken, exactly as
// a corrupt frame body does. Previously StreamDecoder.Decode returned the
// readHeader error WITHOUT setting s.broken, so the next Decode silently resumed
// at the next byte instead of refusing per the documented "once broken, further
// Decode is refused" invariant.
func TestStreamHeaderErrorLatchesBroken(t *testing.T) {
	type rec struct {
		Name string
		ID   int
	}
	var w bytes.Buffer
	se := NewStreamEncoderWith(&w, OptBalanced)
	for _, m := range []rec{{"alpha", 1}, {"beta", 2}} {
		if err := se.Encode(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := se.Close(); err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), w.Bytes()...)
	raw[4] ^= 0x04 // flip FlagRANS in the header flag byte → readHeader fails

	sd := NewStreamDecoder(bytes.NewReader(raw))
	var got rec
	if err := sd.Decode(&got); err == nil {
		t.Fatal("expected first decode to fail on the corrupt header")
	}
	if err := sd.Decode(&got); !errors.Is(err, ErrStreamDecoderBroken) {
		t.Fatalf("after a header error the decoder must latch broken; got %v want ErrStreamDecoderBroken", err)
	}
}
