package qdf

import "testing"

func TestOptFSSTBitsUnchanged(t *testing.T) {
	if OptDense != 1 {
		t.Fatalf("OptDense=%d want 1 (renumbering bug)", OptDense)
	}
	if OptFSST == 0 || OptFSST&(OptFSST-1) != 0 {
		t.Fatalf("OptFSST not a single bit: %d", OptFSST)
	}
	if OptCompression&OptFSST == 0 {
		t.Fatal("OptCompression must include OptFSST")
	}
	if OptBalanced&OptFSST != 0 {
		t.Fatal("OptBalanced must NOT include OptFSST")
	}
	if OptSpeed != 0 {
		t.Fatalf("OptSpeed=%d want 0", OptSpeed)
	}
}
