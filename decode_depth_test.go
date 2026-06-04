package qdf

import (
	"errors"
	"testing"
)

// TestDecode_DepthGuard_NoStackOverflow builds a hostile deeply-nested payload
// (arrays nested past DefaultMaxDepth) by driving the low-level Encoder, which
// bypasses the encoder's own recursion guard. Before the decode depth guard
// this crashed the process with an unrecoverable `fatal error: stack overflow`.
// It must now return an error, not crash.
func TestDecode_DepthGuard_NoStackOverflow(t *testing.T) {
	enc := NewEncoder(Fast)
	enc.EnsureHeader()
	depth := DefaultMaxDepth + 100
	for range depth {
		enc.WriteArrayHeader(1)
	}
	enc.WriteNil()
	buf := append([]byte(nil), enc.Bytes()...)

	var v any
	err := Unmarshal(buf, &v)
	if err == nil {
		t.Fatal("expected an error decoding an over-deep payload, got nil")
	}
	if !errors.Is(err, ErrCycleDetected) {
		t.Logf("got error %v (acceptable as long as it is not a crash)", err)
	}
}

// TestDecode_DepthGuard_NormalDepthOK confirms the guard does not reject
// legitimately nested payloads well within DefaultMaxDepth.
func TestDecode_DepthGuard_NormalDepthOK(t *testing.T) {
	type node struct {
		V    int     `qdf:"v"`
		Next *node   `qdf:"next"`
		Kids []*node `qdf:"kids"`
	}
	// Build a chain ~50 deep — far below the limit, must round-trip.
	var head *node
	for i := range 50 {
		head = &node{V: i, Next: head}
	}
	b, err := Marshal(head, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out *node
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("normal-depth decode failed: %v", err)
	}
	n := 0
	for p := out; p != nil; p = p.Next {
		n++
	}
	if n != 50 {
		t.Fatalf("chain length %d, want 50", n)
	}
}
