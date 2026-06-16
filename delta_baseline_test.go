package qdf

import "testing"

func TestBaselineRegistryEmpty(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	if r == nil {
		t.Fatal("NewBaselineRegistry returned nil")
	}
	if got := r.Len(); got != 0 {
		t.Fatalf("empty registry Len() = %d, want 0", got)
	}
}
