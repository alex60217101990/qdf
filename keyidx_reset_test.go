package qdf

import "testing"

func TestKeyIdxClearedOnReset(t *testing.T) {
	e := NewEncoderWith(OptBalanced)
	e.keyIdx = map[string]int{"alpha": 1, "beta": 2} // simulate aliasing keys
	e.resetForReuse()
	if len(e.keyIdx) != 0 {
		t.Fatalf("encoder keyIdx not cleared on reset: %d entries (GC-pins aliasing keys)", len(e.keyIdx))
	}
}
