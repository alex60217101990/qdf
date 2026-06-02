package qdf

import "testing"

// TestPForInto_BitsPerZero_OverwritesScratch is a deterministic regression test
// for the stale-scratch under-fill bug in the PFor Into-variant decoders.
//
// When a PForUint64/Int64 column has bitsPer==0 and min==0 with no exceptions
// (the canonical all-zero column), the Into-variant must still fully overwrite
// dst[:n]. The bug skipped both bitpack.Unpack (b==0) and the min add-loop
// (min==0), leaving the reused scratch buffer's stale contents in place. The
// old make()-based decoder masked this because a fresh slice is zero-init.
func TestPForInto_BitsPerZero_OverwritesScratch(t *testing.T) {
	// Wire (starting at the qpackKind byte, as the Into-variant expects):
	//   kind, uvarint n, byte bitsPer=0, uvarint min=0, uvarint excN=0
	const n = 5

	t.Run("uint64", func(t *testing.T) {
		buf := []byte{qpackKindUint64, n, 0x00, 0x00, 0x00}
		d := &Decoder{buf: buf}
		dst := []uint64{9, 9, 9, 9, 9} // pre-dirtied scratch
		if err := d.readPackedPForUint64SliceInto(&dst); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for i, v := range dst {
			if v != 0 {
				t.Fatalf("uint64 idx %d: got %d, want 0 (stale scratch leaked)", i, v)
			}
		}
	})

	t.Run("int64", func(t *testing.T) {
		buf := []byte{qpackKindInt64, n, 0x00, 0x00, 0x00}
		d := &Decoder{buf: buf}
		dst := []int64{9, 9, 9, 9, 9} // pre-dirtied scratch
		if err := d.readPackedPForInt64SliceInto(&dst); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for i, v := range dst {
			if v != 0 {
				t.Fatalf("int64 idx %d: got %d, want 0 (stale scratch leaked)", i, v)
			}
		}
	})
}
