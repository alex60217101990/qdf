package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestProfileWireHash pins the invariant work item A rests on: finding an
// intern id through the string's address rather than by hashing its bytes
// changes HOW the id is found, never WHICH id the string gets, so the encoded
// bytes must be identical. Run it before and after the change and compare the
// printed digest.
//
// The digest covers every profile under every option set. A vacuity control is
// included: mutating one byte of one encoding must change the digest, so a
// passing comparison cannot be passing for free.
func TestProfileWireHash(t *testing.T) {
	h := sha256.New()
	for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
		for _, v := range []any{
			mkLogProfile(1024),
			mkTelemetryProfile(1024),
			mkAPIProfile(512),
		} {
			blob, err := Marshal(v, o)
			if err != nil {
				t.Fatalf("%v: %v", o, err)
			}
			h.Write(blob)
		}
	}
	sum := hex.EncodeToString(h.Sum(nil))
	t.Logf("profile wire digest: %s", sum)

	// Vacuity control: the same corpus with one byte flipped must differ.
	h2 := sha256.New()
	first := true
	for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
		for _, v := range []any{
			mkLogProfile(1024),
			mkTelemetryProfile(1024),
			mkAPIProfile(512),
		} {
			blob, err := Marshal(v, o)
			if err != nil {
				t.Fatal(err)
			}
			if first && len(blob) > 0 {
				blob = append([]byte(nil), blob...)
				blob[len(blob)-1] ^= 1
				first = false
			}
			h2.Write(blob)
		}
	}
	if hex.EncodeToString(h2.Sum(nil)) == sum {
		t.Fatal("vacuity control failed: a flipped byte did not change the digest")
	}
}
