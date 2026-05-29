package rans

import (
	"bytes"
	"testing"
)

func TestRoundTrip_Edges(t *testing.T) {
	cases := [][]byte{
		{},
		{0x42},
		bytes.Repeat([]byte{0xFF}, 1),
		bytes.Repeat([]byte{0xAB}, 4096),
		func() []byte { // all 256 symbols, skewed
			b := make([]byte, 0, 5000)
			for s := 0; s < 256; s++ {
				b = append(b, bytes.Repeat([]byte{byte(s)}, s+1)...)
			}
			return b
		}(),
	}
	for i, src := range cases {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("case %d mismatch", i)
		}
	}
}

func TestDecode_RejectsGarbage(t *testing.T) {
	// 256 zero freqs -> sum 0 != scale: must be rejected, not panic.
	garbage := bytes.Repeat([]byte{0x00}, 8)
	if _, err := Decode(garbage, 10); err == nil {
		t.Fatal("expected error on zero-sum table")
	}
}

func FuzzRoundTrip(f *testing.F) {
	f.Add([]byte("quantum density format"))
	f.Add(bytes.Repeat([]byte{1, 2, 3}, 100))
	f.Fuzz(func(t *testing.T, src []byte) {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("decode err on valid encode: %v", err)
		}
		if !bytes.Equal(got, src) {
			t.Fatal("round-trip mismatch")
		}
	})
}

func FuzzDecode_NeverPanics(f *testing.F) {
	f.Add([]byte{0x00, 0x10, 0x00, 0x00}, 5)
	f.Fuzz(func(t *testing.T, src []byte, n int) {
		if n < 0 || n > 1<<20 {
			return
		}
		_, _ = Decode(src, n) // must not panic; error is fine
	})
}

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
