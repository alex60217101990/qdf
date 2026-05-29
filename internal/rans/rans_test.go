package rans

import (
	"bytes"
	"testing"
)

func TestRoundTrip_Basic(t *testing.T) {
	inputs := [][]byte{
		[]byte("hello hello hello world world"),
		bytes.Repeat([]byte{0x00}, 1000),
		append(bytes.Repeat([]byte("ab"), 500), bytes.Repeat([]byte("c"), 200)...),
	}
	for i, src := range inputs {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("case %d: decode err: %v", i, err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("case %d: round-trip mismatch\n got=%v\nwant=%v", i, got, src)
		}
	}
}
