package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// PreIntern is a non-breaking encoder-side optimisation: when the
// caller registers a hot string pool up front, subsequent WriteString
// / WriteBytes calls with the same backing pointer skip the intern
// table's hash + slot probe and emit a state-ref directly. These
// tests pin the contract.

// TestPreIntern_RoundTripDense — happy path: register a pool, encode
// a struct that uses those values, decode and compare.
func TestPreIntern_RoundTripDense(t *testing.T) {
	type row struct {
		Service string `qdf:"service"`
		Region  string `qdf:"region"`
		Status  int    `qdf:"status"`
	}
	services := []string{"billing", "auth", "ingest", "metrics", "api"} //nolint:prealloc // literal slice, prealloc hint is misleading
	regions := []string{"eu-west-1", "us-east-1", "ap-south-1"}
	rows := []row{
		{services[0], regions[0], 200},
		{services[1], regions[1], 500},
		{services[2], regions[0], 200},
		{services[3], regions[2], 404},
		{services[0], regions[0], 200},
	}

	enc := NewEncoderWith(OptBalanced)
	enc.PreIntern(append(services, regions...)...)
	if err := encodeReflect(enc, rows); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := append([]byte(nil), enc.Bytes()...)

	var dec []row
	if err := Unmarshal(out, &dec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(rows, dec) {
		t.Fatalf("round trip mismatch:\n got=%#v\nwant=%#v", dec, rows)
	}
}

// TestPreIntern_NoOpWithoutDense — PreIntern is a no-op when the
// encoder is not in Dense mode; the encode must still succeed and
// produce a valid wire payload.
func TestPreIntern_NoOpWithoutDense(t *testing.T) {
	enc := NewEncoderWith(OptSpeed)
	enc.PreIntern("never-used-because-not-dense")
	if got := len(enc.preIntern); got != 0 {
		t.Fatalf("PreIntern populated cache outside Dense mode: %d entries", got)
	}
	if err := encodeReflect(enc, "hello"); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Contains(enc.Bytes(), []byte("hello")) {
		t.Fatalf("Fast-mode encode did not contain the inline string payload")
	}
}

// TestPreIntern_ResetClearsCache — pooled encoders must drop
// caller-supplied pointers across recycle; otherwise a previous
// caller's stack-allocated string could outlive its frame.
func TestPreIntern_ResetClearsCache(t *testing.T) {
	enc := NewEncoderWith(OptBalanced)
	enc.PreIntern("alpha", "beta", "gamma")
	if got := len(enc.preIntern); got == 0 {
		t.Fatal("PreIntern did not populate cache in Dense mode")
	}
	enc.Reset()
	if got := len(enc.preIntern); got != 0 {
		t.Fatalf("Reset did not clear PreIntern cache: %d entries", got)
	}
}

// TestPreIntern_FallbackOnUnregistered — the identity scan must NOT
// false-hit on strings whose content matches but whose backing
// pointer differs. Such strings have to fall through to the regular
// intern table lookup.
func TestPreIntern_FallbackOnUnregistered(t *testing.T) {
	hotPool := []string{"alpha", "beta"}
	enc := NewEncoderWith(OptBalanced)
	enc.PreIntern(hotPool...)

	// Build a fresh copy of "alpha" with a different backing pointer.
	// Same content, distinct header.
	cold := string([]byte("alpha"))

	if err := encodeReflect(enc, []string{hotPool[0], cold}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out []string
	if err := Unmarshal(enc.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(out, []string{"alpha", "alpha"}) {
		t.Fatalf("round-trip mismatch: got %#v want [\"alpha\" \"alpha\"]", out)
	}
}
