package qdf

import (
	"bytes"
	"testing"
)

// TestDirect_EqualsSpeed verifies that MarshalDirect produces the same wire
// bytes as Marshal(v, OptSpeed), as documented.
func TestDirect_EqualsSpeed(t *testing.T) {
	in := directSample{ID: 42, Name: "hello"}

	wantBytes, err := Marshal(&in, OptSpeed)
	if err != nil {
		t.Fatalf("Marshal(OptSpeed): %v", err)
	}
	gotBytes, err := MarshalDirect(&in)
	if err != nil {
		t.Fatalf("MarshalDirect: %v", err)
	}
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatalf("wire mismatch:\n OptSpeed=%x\n Direct  =%x", wantBytes, gotBytes)
	}
}

// TestDirect_FallbackOnDense verifies that UnmarshalDirect correctly falls
// back to the reflect path when decoding a Dense-encoded payload (FlagDense
// set), so that generated/hand-rolled UnmarshalQDF code does not need to
// understand the intern table.
func TestDirect_FallbackOnDense(t *testing.T) {
	in := directSample{ID: 99, Name: "dense-fallback"}

	// Encode with OptBalanced which sets FlagDense.
	dense, err := Marshal(&in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal(OptBalanced): %v", err)
	}

	// UnmarshalDirect must still decode correctly via the reflect fallback.
	var out directSample
	if err := UnmarshalDirect(dense, &out); err != nil {
		t.Fatalf("UnmarshalDirect on dense payload: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", out, in)
	}
}
