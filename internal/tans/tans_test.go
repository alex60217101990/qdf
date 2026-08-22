package tans

import (
	"bytes"
	"testing"
)

func TestRoundTrip_Skeleton(t *testing.T) {
	src := bytes.Repeat([]byte("abcabc"), 100)
	enc := Encode(nil, src)
	got, err := Decode(enc, len(src))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatal("round-trip mismatch")
	}
}

func TestEncode_NonEmpty(t *testing.T) {
	src := mkSkewed(8192)
	enc := Encode(nil, src)
	if len(enc) == 0 {
		t.Fatal("encode produced empty output")
	}
	if enc[0] != TagInter4 {
		t.Fatalf("expected TagInter4, got %d", enc[0])
	}
}

func TestEncode_Small(t *testing.T) {
	src := []byte("hello")
	enc := Encode(nil, src)
	if enc[0] != TagSingle {
		t.Fatalf("expected TagSingle, got %d", enc[0])
	}
}

// mkSkewed returns n bytes where ~70% are 'a'. Used by benchmarks.
func mkSkewed(n int) []byte {
	src := make([]byte, n)
	for i := range src {
		if i%10 < 7 {
			src[i] = 'a'
		} else {
			src[i] = byte('b' + i%26)
		}
	}
	return src
}

func TestRoundTrip_Full(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
	}{
		{"empty", nil},
		{"single_byte", []byte{0x42}},
		{"constant_1k", bytes.Repeat([]byte{0xAA}, 1024)},
		{"constant_8k", bytes.Repeat([]byte{0xBB}, 8192)},
		{"skewed_4k", mkSkewed(4096)},
		{"skewed_64k", mkSkewed(65536)},
		{"all_256_symbols", func() []byte {
			b := make([]byte, 0, 5000)
			for s := range 256 {
				b = append(b, bytes.Repeat([]byte{byte(s)}, s+1)...)
			}
			return b
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc := Encode(nil, tc.src)
			got, err := Decode(enc, len(tc.src))
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(got, tc.src) {
				t.Fatalf("round-trip mismatch: got %d bytes, want %d", len(got), len(tc.src))
			}
		})
	}
}

func TestDecode_RejectsGarbage(t *testing.T) {
	// Tag byte out of range.
	if _, err := Decode([]byte{0x99}, 10); err == nil {
		t.Fatal("expected error on bad tag")
	}
	// Truncated freq table.
	if _, err := Decode(append([]byte{TagSingle}, make([]byte, 4)...), 10); err == nil {
		t.Fatal("expected error on truncated table")
	}
	// All-zero freq table (sum != TableSize).
	if _, err := Decode(append([]byte{TagSingle}, make([]byte, 256)...), 10); err == nil {
		t.Fatal("expected error on zero-sum table")
	}
}

func FuzzRoundTrip(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add(bytes.Repeat([]byte{1, 2, 3}, 100))
	f.Add(bytes.Repeat([]byte{1, 2, 3}, 2000)) // ≥ interleaveMinBytes
	f.Fuzz(func(t *testing.T, src []byte) {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("decode error on valid encode: %v", err)
		}
		if !bytes.Equal(got, src) {
			t.Fatal("round-trip mismatch")
		}
	})
}

func FuzzDecode_NeverPanics(f *testing.F) {
	f.Add([]byte{TagSingle, 0x00}, 5)
	f.Add(Encode(nil, bytes.Repeat([]byte{4, 5, 6}, 4000)), 12000)
	f.Fuzz(func(_ *testing.T, src []byte, n int) {
		if n < 0 || n > 1<<20 {
			return
		}
		_, _ = Decode(src, n) // must not panic
	})
}
