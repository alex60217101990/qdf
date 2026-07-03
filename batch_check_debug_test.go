//go:build qdfdebug || race

package qdf

import "testing"

// TestBatchStaleHandlePanics: resolving a handle after Release must panic in
// debug/race builds instead of returning silent garbage (Release nils the
// Batch's slab pointer and bumps batchSlab.epoch — checkEpoch catches both).
func TestBatchStaleHandlePanics(t *testing.T) {
	src := mkBatSrc(4)
	data, _ := Marshal(src, OptBalanced)
	b, _ := UnmarshalBatch[batDoc](data)
	h := b.Rows[0].Name
	b.Release()
	defer func() {
		if recover() == nil {
			t.Fatal("stale handle resolve must panic in debug builds")
		}
	}()
	_ = b.Str(h)
}

// TestBatchStaleHandlePanicsBytesOf mirrors the Str case for BytesOf, using
// the qdf.Bytes column so the checkEpoch guard on that resolve path is
// exercised too.
func TestBatchStaleHandlePanicsBytesOf(t *testing.T) {
	src := []batBytesSrc{{ID: 1, Blob: []byte("payload"), Name: "n"}}
	data, err := Marshal(src, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batBytesDoc](data)
	if err != nil {
		t.Fatal(err)
	}
	h := b.Rows[0].Blob
	b.Release()
	defer func() {
		if recover() == nil {
			t.Fatal("stale handle resolve must panic in debug builds")
		}
	}()
	_ = b.BytesOf(h)
}

// TestBatchStaleHandlePanicsAfterSlabReuse: after Release, the slab is
// returned to the pool and its epoch bumped. A fresh decode may reuse the
// same *batchSlab (same pointer) at a new epoch — the stale Batch's captured
// epoch must no longer match, so resolving its handles still panics rather
// than silently reading whatever the new decode wrote into the same buffer.
func TestBatchStaleHandlePanicsAfterSlabReuse(t *testing.T) {
	src := mkBatSrc(4)
	data, _ := Marshal(src, OptBalanced)
	b, _ := UnmarshalBatch[batDoc](data)
	h := b.Rows[0].Name
	b.Release()

	// Force a second decode through the same pooled slab.
	b2, err := UnmarshalBatch[batDoc](data)
	if err != nil {
		t.Fatal(err)
	}
	defer b2.Release()

	defer func() {
		if recover() == nil {
			t.Fatal("stale handle resolve after slab reuse must panic in debug builds")
		}
	}()
	_ = b.Str(h)
}
