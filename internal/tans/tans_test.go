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
