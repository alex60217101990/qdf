package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// wantProfileWireDigest is the encoding of the whole profile corpus. It is
// pinned, not logged, so the check survives the work item that motivated it:
// an encoder change that alters what goes on the wire fails here rather than
// waiting for someone to eyeball two runs.
//
// Changing it is a deliberate act. A diff means the wire format moved, which
// breaks every already-encoded payload — so update this constant only together
// with a compatibility note saying why the break is intended, never to make a
// red test go green.
const wantProfileWireDigest = "f7b0284fbff425d9e229e3c3f874a4b8032848ef78bc5b9607f9c74f412d28a2"

// TestProfileWireHash pins the invariant work item A rested on: finding an
// intern id through the string's address rather than by hashing its bytes
// changes HOW the id is found, never WHICH id the string gets, so the encoded
// bytes must be identical. A itself was measured and reverted, but the check
// outlives it — any future change to the encoder's string or intern path is
// subject to the same requirement.
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
		} {
			blob, err := Marshal(v, o)
			if err != nil {
				t.Fatalf("%v: %v", o, err)
			}
			h.Write(blob)
		}
		// The API profile carries a map[string]string field; its encoding is
		// only key-order-stable under OptCanonical. Canonical fixes order
		// without changing whether keys are interned, so the intern path
		// this work item touches stays covered while the digest stays
		// deterministic across runs.
		blob, err := Marshal(mkAPIProfile(512), o|OptCanonical)
		if err != nil {
			t.Fatalf("%v|OptCanonical: %v", o, err)
		}
		h.Write(blob)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	t.Logf("profile wire digest: %s", sum)
	if sum != wantProfileWireDigest {
		t.Errorf("wire digest changed:\n got %s\nwant %s\nthe encoded bytes moved; see the note on wantProfileWireDigest", sum, wantProfileWireDigest)
	}

	// Vacuity control: the same corpus with one byte flipped must differ.
	h2 := sha256.New()
	first := true
	for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
		for _, v := range []any{
			mkLogProfile(1024),
			mkTelemetryProfile(1024),
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
		blob, err := Marshal(mkAPIProfile(512), o|OptCanonical)
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
	if hex.EncodeToString(h2.Sum(nil)) == sum {
		t.Fatal("vacuity control failed: a flipped byte did not change the digest")
	}
}
