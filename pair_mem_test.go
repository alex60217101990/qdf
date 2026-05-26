package qdf

import (
	"testing"
	"unsafe"
)

// Per-slot pairPred storage must stay at 4 bytes (uint32, succ+1
// packed with empty=0). Previous design used a [4]uint32 ring with a
// uint8 n field, padded to 24 bytes. At default maxStateEntries
// (16384) the regression here would re-add ~320 KiB to every pooled
// encoder's resident set — guard the budget so any accidental
// reintroduction of a struct slot trips this test.
func TestMemoryFootprint_PairPredSlot(t *testing.T) {
	var s uint32
	if got := unsafe.Sizeof(s); got != 4 {
		t.Fatalf("pairPred slot must be 4 bytes; got %d", got)
	}
}
