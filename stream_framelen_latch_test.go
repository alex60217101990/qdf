package qdf

import (
	"bytes"
	"errors"
	"testing"
)

// Regression: a truncated frame BODY must latch the decoder broken, exactly as a
// corrupt header or a mid-frame decode error does. Previously
// StreamDecoder.Decode returned the fill() ErrShortBuffer WITHOUT setting
// s.broken, leaving the cursor advanced past the frame-length prefix. A caller
// that ignored the error and called Decode again would re-parse the buffered
// partial body bytes as a fresh frame and silently misparse, violating the
// documented "once broken, further Decode is refused" invariant.
func TestStreamTruncatedFrameBodyLatchesBroken(t *testing.T) {
	type rec struct {
		Name string
		ID   int
	}
	var w bytes.Buffer
	se := NewStreamEncoderWith(&w, OptBalanced)
	if err := se.Encode(rec{"a-reasonably-long-name-for-body-bytes", 7}); err != nil {
		t.Fatal(err)
	}
	if err := se.Close(); err != nil {
		t.Fatal(err)
	}
	raw := w.Bytes()
	// Cut into the frame body: keep the header and the length prefix plus a few
	// body bytes, but drop the tail so fill(framelen) sees a truncated frame.
	truncated := append([]byte(nil), raw[:len(raw)-3]...)

	sd := NewStreamDecoder(bytes.NewReader(truncated))
	var got rec
	if err := sd.Decode(&got); err == nil {
		t.Fatal("expected first decode to fail on the truncated frame body")
	}
	if err := sd.Decode(&got); !errors.Is(err, ErrStreamDecoderBroken) {
		t.Fatalf("after a truncated frame the decoder must latch broken; got %v want ErrStreamDecoderBroken", err)
	}
}
