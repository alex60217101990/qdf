package qdf

import (
	"bytes"
	"strings"
	"testing"
)

// TestRANS_LowEntropyRoundTrips pins that OptCompression never emits a rANS
// frame its own decoder rejects. A multi-MiB low-entropy body compresses well
// past the decoder's origLen bound (len(buf)*64 + 1MiB, which caps the decode
// allocation); maybeApplyRANS now declines rANS in that case and keeps the plain
// body, which still round-trips. Regression for the encode/decode bound
// asymmetry (mirror of the constant-codec OOM cap fix).
func TestRANS_LowEntropyRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates multi-MiB payloads; skipped under -short")
	}
	for _, sz := range []int{1 << 20, 4 << 20, 8 << 20} {
		in := strings.Repeat("A", sz) // maximal compression: degenerate freq
		buf, err := Marshal(in, OptCompression)
		if err != nil {
			t.Fatalf("sz=%d Marshal: %v", sz, err)
		}
		var out string
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("sz=%d round-trip broken: %d-byte input -> %d-byte wire rejected: %v",
				sz, len(in), len(buf), err)
		}
		if out != in {
			t.Fatalf("sz=%d value mismatch", sz)
		}
	}
}

// TestRANS_StillCompressesMediumEntropy is the no-regression counterpart: rANS
// must STILL fire (and round-trip) for realistic medium-entropy data that stays
// within the decoder's origLen bound.
func TestRANS_StillCompressesMediumEntropy(t *testing.T) {
	blob := bytes.Repeat([]byte("the quick brown fox 0123456789 ABCDEF "), 4000)
	buf, err := Marshal(blob, OptCompression)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(buf) >= len(blob) {
		t.Fatalf("rANS did not compress medium-entropy data: %d -> %d bytes", len(blob), len(buf))
	}
	var out []byte
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !bytes.Equal(out, blob) {
		t.Fatal("medium-entropy round-trip mismatch")
	}
}
