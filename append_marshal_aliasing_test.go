package qdf

import (
	"bytes"
	"testing"
)

// unencodableType has no qdf representation (chan field) so encode fails,
// driving the AppendMarshal* error path.
type unencodableType struct {
	C chan int
}

// TestAppendMarshalT_ErrorDoesNotPinCallerBuf verifies that when
// AppendMarshalT fails it does NOT leave the caller's dst aliased inside a
// pooled encoder. If it does, a subsequent encode reuses that encoder and
// overwrites the caller's backing array with the next message's wire bytes.
func TestAppendMarshalT_ErrorDoesNotPinCallerBuf(t *testing.T) {
	dst := bytes.Repeat([]byte{0xAB}, 64)
	want := append([]byte(nil), dst...)

	if _, err := AppendMarshalT(dst, unencodableType{C: make(chan int)}, OptBalanced); err == nil {
		t.Fatal("expected encode error for chan field, got nil")
	}

	// Force the pool to hand back the (possibly poisoned) encoder and write a
	// fresh message through it.
	if _, err := MarshalT([]int64{1, 2, 3, 4, 5, 6}, OptBalanced); err != nil {
		t.Fatalf("MarshalT: %v", err)
	}

	if !bytes.Equal(dst, want) {
		t.Fatalf("caller dst was corrupted by a later encode:\n got %x\nwant %x", dst, want)
	}
}

// TestAppendMarshalDict_ErrorDoesNotPinCallerBuf is the same check for the
// reflect-backed public AppendMarshal path (appendMarshalDict).
func TestAppendMarshalDict_ErrorDoesNotPinCallerBuf(t *testing.T) {
	dst := bytes.Repeat([]byte{0xCD}, 64)
	want := append([]byte(nil), dst...)

	if _, err := AppendMarshal(dst, unencodableType{C: make(chan int)}, OptBalanced); err == nil {
		t.Fatal("expected encode error for chan field, got nil")
	}

	if _, err := Marshal([]int64{7, 8, 9, 10, 11, 12}, OptBalanced); err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if !bytes.Equal(dst, want) {
		t.Fatalf("caller dst was corrupted by a later encode:\n got %x\nwant %x", dst, want)
	}
}
