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
