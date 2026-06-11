package rans

import (
	"bytes"
	"testing"
)

func TestTagRoundTripSingle(t *testing.T) {
	for _, src := range [][]byte{
		{}, {0}, {1, 1, 1, 1}, bytes.Repeat([]byte("hello world "), 100),
	} {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("round-trip mismatch: got %q want %q", got, src)
		}
	}
	enc := Encode(nil, bytes.Repeat([]byte("x"), 50))
	if enc[0] != ransTagSingle {
		t.Fatalf("expected tag 0, got %d", enc[0])
	}
}
