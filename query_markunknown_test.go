package qdf

import "testing"

// TestMarkUnknownNullableProp guards markUnknown against a tempting but WRONG
// "modernization" to slices.Backward: the loop mutates flat[i].unk through the
// index on a value slice ([]cnode), so ranging by value binds a copy and the
// writes are lost. A nullable leaf's unk must be set and must propagate to its
// parent.
func TestMarkUnknownNullableProp(t *testing.T) {
	flat := []cnode{
		{op: condNot, kids: []int{1}},
		{op: condLeaf, cv: &colVals{present: []uint64{0x1}}},
	}
	markUnknown(flat)
	if !flat[1].unk {
		t.Fatal("nullable leaf unk not set")
	}
	if !flat[0].unk {
		t.Fatal("parent unk not propagated")
	}
}
