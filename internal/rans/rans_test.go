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
			for s := range 256 {
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
	f.Add(bytes.Repeat([]byte{1, 2, 3}, 5000))                // ≥ interleaveMinBytes → interleaved path
	f.Add(bytes.Repeat([]byte("the quick brown fox "), 1000)) // skewed, large → interleaved
	f.Add(adversarialRareSubstream(8000))                     // rare-byte substream → worst-case region
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
	f.Add(Encode(nil, bytes.Repeat([]byte{4, 5, 6}, 4000)), 12000) // interleaved blob seed
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

// adversarialRareSubstream builds a stream whose interleaved substream 0 is
// dominated by globally-rare bytes: every position is 0x00 (huge frequency)
// except positions ≡0 mod interleaveN, which cycle through 1..255 (each a low,
// often freq-1 symbol). Encoded under the SHARED freq table, substream 0 emits
// ~maxRenormBytesPerSym bytes/symbol — overflowing the old m+16 region.
func adversarialRareSubstream(nElems int) []byte {
	src := make([]byte, nElems)
	r := byte(1)
	for i := range src {
		if i%4 == 0 {
			src[i] = r
			r++
			if r == 0 {
				r = 1
			}
		}
	}
	return src
}

// TestEncode_AdversarialSubstreamNoOverflow is a regression for the interleaved
// rANS encode buffer underflow: a substream skewed against the shared frequency
// table emits up to 2 bytes/symbol, which the old len(src)+16 region under-sized
// → index-out-of-range panic. Encode must now size for the worst case and the
// stream must round-trip.
func TestEncode_AdversarialSubstreamNoOverflow(t *testing.T) {
	for _, n := range []int{2000, 5000, 9000, 40000} {
		src := adversarialRareSubstream(n)
		enc := Encode(nil, src) // must not panic
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("n=%d decode err: %v", n, err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("n=%d round-trip mismatch", n)
		}
	}
}
