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

// TestDirect_ReflectUnmarshaler is a regression test for a real bug: a top-
// level Unmarshal into a type implementing Unmarshaler used to fail with
// ErrTypeMismatch because decodeUnmarshaler handed the user's UnmarshalQDF the
// full buffer (magic+flags) instead of the body. The reflect path now consumes
// the 5-byte header first (mirroring UnmarshalDirect's data[5:]).
func TestDirect_ReflectUnmarshaler(t *testing.T) {
	in := directSample{ID: 1234, Name: "reflect-path"}
	// Fast-encoded, decoded via the reflect Unmarshal entry point (NOT
	// UnmarshalDirect) — exercises decodeUnmarshaler at top level.
	data, err := Marshal(&in, OptSpeed)
	if err != nil {
		t.Fatalf("Marshal(OptSpeed): %v", err)
	}
	var out directSample
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("reflect Unmarshal of Unmarshaler type: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", out, in)
	}
}

// TestDirect_FallbackOnDense verifies that UnmarshalDirect falls back to the
// reflect path on a Dense payload (FlagDense set). NOTE: this succeeds here
// because directSample is a small struct whose Dense body carries no intern
// state-refs, so its Fast-mode UnmarshalQDF can still parse it. A type that
// relies on dense interning would NOT be decodable this way — the fallback
// routes back through the receiver's own UnmarshalQDF, which is Fast-only.
// (Tracked as a known limitation, not a guarantee.)
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
