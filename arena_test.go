package qdf

import (
	"testing"
	"unsafe"
)

type arenaStr struct {
	A string `qdf:"a"`
	B string `qdf:"b"`
	C string `qdf:"c"`
}

// TestArenaRoundTrip: decoding with an arena yields bit-exact values.
func TestArenaRoundTrip(t *testing.T) {
	in := arenaStr{A: "alpha", B: "bravo", C: "charlie"}
	buf, err := Marshal(&in, OptSpeed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	a := NewArena()
	var out arenaStr
	if err := Unmarshal(buf, &out, WithArena(a)); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: got %+v want %+v", out, in)
	}
	// Strings must NOT alias the input buffer (they are copied into the arena).
	bufBase := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
	pa := uintptr(unsafe.Pointer(unsafe.StringData(out.A)))
	if pa >= bufBase && pa < bufBase+uintptr(len(buf)) {
		t.Fatal("out.A aliases input buffer; expected arena copy")
	}
}

// TestArenaCoalesces: all inline string fields land contiguously in one arena
// block, bump-packed in decode order (deterministic structural check).
func TestArenaCoalesces(t *testing.T) {
	in := arenaStr{A: "alpha", B: "bravo", C: "charlie"}
	buf, _ := Marshal(&in, OptSpeed)
	a := NewArena()
	var out arenaStr
	if err := Unmarshal(buf, &out, WithArena(a)); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pa := uintptr(unsafe.Pointer(unsafe.StringData(out.A)))
	pb := uintptr(unsafe.Pointer(unsafe.StringData(out.B)))
	pc := uintptr(unsafe.Pointer(unsafe.StringData(out.C)))
	if pb != pa+uintptr(len(out.A)) {
		t.Fatalf("B not bump-adjacent to A: A@%#x len %d B@%#x", pa, len(out.A), pb)
	}
	if pc != pb+uintptr(len(out.B)) {
		t.Fatalf("C not bump-adjacent to B: B@%#x len %d C@%#x", pb, len(out.B), pc)
	}
}

// TestArenaResetReuses: Reset rewinds the bump cursor and reuses the same
// backing block (deterministic pointer-identity check), so a reused arena
// allocates nothing on the next epoch.
func TestArenaResetReuses(t *testing.T) {
	in := arenaStr{A: "alpha", B: "bravo", C: "charlie"}
	buf, _ := Marshal(&in, OptSpeed)
	a := NewArena()

	var out1 arenaStr
	if err := Unmarshal(buf, &out1, WithArena(a)); err != nil {
		t.Fatalf("unmarshal 1: %v", err)
	}
	p1 := uintptr(unsafe.Pointer(unsafe.StringData(out1.A)))

	a.Reset()

	var out2 arenaStr
	if err := Unmarshal(buf, &out2, WithArena(a)); err != nil {
		t.Fatalf("unmarshal 2: %v", err)
	}
	p2 := uintptr(unsafe.Pointer(unsafe.StringData(out2.A)))

	if p1 != p2 {
		t.Fatalf("Reset did not reuse backing: first A@%#x second A@%#x", p1, p2)
	}
	if out2 != in {
		t.Fatalf("post-reset round-trip mismatch: got %+v", out2)
	}
}
