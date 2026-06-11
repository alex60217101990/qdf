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

func TestAppendInterleavedStructure(t *testing.T) {
	src := []byte("the quick brown fox jumps over the lazy dog, again and again")
	freq, cum := buildFreqs(src)
	for _, N := range []int{2, 4} {
		blob := appendInterleaved(nil, src, &freq, &cum, N)
		// blob = N*4 states + (N-1) uvarint lengths + substream bytes; must be non-empty.
		if len(blob) < N*4 {
			t.Fatalf("N=%d: blob too short (%d)", N, len(blob))
		}
	}
}
