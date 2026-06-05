package qdf

import (
	"bytes"
	"errors"
	"testing"
)

// TestStream_DecodeErrorPoisons pins the fix for the mid-frame desync: when a
// Decode fails partway through a frame (here, decoding a struct frame into an
// incompatible target), the read cursor and shared dense state are left
// inconsistent. The decoder must refuse further Decodes with
// ErrStreamDecoderBroken instead of misparsing the rest of the frame as new
// frames and returning silently wrong values.
func TestStream_DecodeErrorPoisons(t *testing.T) {
	type Evt struct {
		Country string `qdf:"country"`
		City    string `qdf:"city"`
		ID      int    `qdf:"id"`
	}
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for i := 0; i < 3; i++ {
		if err := enc.Encode(Evt{Country: "Lithuania", City: "Vilnius", ID: i}); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	// Decode the first frame into an incompatible target to force a mid-frame
	// error (a struct frame cannot decode into *int).
	var bad int
	if err := dec.Decode(&bad); err == nil {
		t.Skip("decoding a struct frame into *int did not error on this build; trigger no longer valid")
	}
	// The stream is now poisoned — the next Decode must fail cleanly, not return
	// a garbage value or a spurious framing error.
	var ev Evt
	err := dec.Decode(&ev)
	if !errors.Is(err, ErrStreamDecoderBroken) {
		t.Fatalf("after a mid-frame decode error, next Decode = %v, want ErrStreamDecoderBroken (got ev=%+v)", err, ev)
	}
}

// TestStream_CleanDecodeNotPoisoned guards the other side: a normal successful
// decode sequence is never marked broken.
func TestStream_CleanDecodeNotPoisoned(t *testing.T) {
	type Evt struct {
		Name string `qdf:"name"`
		N    int    `qdf:"n"`
	}
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for i := 0; i < 5; i++ {
		if err := enc.Encode(Evt{Name: "svc", N: i}); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	for i := 0; i < 5; i++ {
		var ev Evt
		if err := dec.Decode(&ev); err != nil {
			t.Fatalf("clean decode %d errored: %v", i, err)
		}
		if ev.N != i || ev.Name != "svc" {
			t.Fatalf("frame %d = %+v", i, ev)
		}
	}
}
