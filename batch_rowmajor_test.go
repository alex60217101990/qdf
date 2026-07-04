package qdf

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

// TestBatchRowMajorDirectParity locks Batch decode parity for a row-major
// wire BEFORE tryDecodeBatchRowMajor grows a real decode body, so a later
// change to it cannot silently regress correctness.
//
// OptBalanced includes OptDense, which transposes a []struct slice at
// len >= columnarMinElems (16) into the columnar tagColStruct wire.
// Stripping OptDense forces a row-major wire at every n, including n >= 16 —
// unlike TestUnmarshalBatchRowMajor (n=4, naturally below the columnar
// threshold), this exercises the row-major path across the full size range
// the batch fast path cares about.
func TestBatchRowMajorDirectParity(t *testing.T) {
	for _, n := range []int{0, 1, 64, 1000} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			src := mkBatSrc(n)
			data, err := Marshal(src, OptBalanced&^OptDense)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			// Whitebox check that the wire really is row-major (not columnar
			// or hybrid) — trusting the flag alone would let this test pass
			// even if a future encoder change re-triggered columnar transpose
			// for some n despite OptDense being stripped.
			d := &Decoder{buf: data}
			tag, err := d.peekTag()
			if err != nil {
				t.Fatalf("peekTag: %v", err)
			}
			if tag == tagColStruct || tag == tagHybridColStruct {
				t.Fatalf("n=%d: expected row-major wire, got columnar/hybrid tag %#x", n, tag)
			}

			b, err := UnmarshalBatch[batDoc](data)
			if err != nil {
				t.Fatalf("UnmarshalBatch: %v", err)
			}
			defer b.Release()
			if len(b.Rows) != n {
				t.Fatalf("rows = %d, want %d", len(b.Rows), n)
			}
			for i, r := range b.Rows {
				if r.ID != int64(i) || b.Str(r.Name) != src[i].Name || r.Val != src[i].Val {
					t.Fatalf("row %d = %+v (name=%q), want id=%d name=%q val=%v",
						i, r, b.Str(r.Name), src[i].ID, src[i].Name, src[i].Val)
				}
				if !b.TimeOf(r.At).Equal(src[i].At) {
					t.Fatalf("row %d time = %v, want %v", i, b.TimeOf(r.At), src[i].At)
				}
			}
		})
	}
}

// TestTryDecodeBatchRowMajorSkeletonAlwaysDefers is a whitebox test asserting
// this task's contract directly: even for a wire tryDecodeBatchRowMajor
// detects as row-major, the skeleton returns ok=false (delegates to the
// mirror fallback) rather than ok=true. A later change that implements the
// real decode body is expected to flip this to ok=true on success — at that
// point this test should be updated/replaced, not left asserting the
// skeleton's inert behavior.
func TestTryDecodeBatchRowMajorSkeletonAlwaysDefers(t *testing.T) {
	src := mkBatSrc(64)
	data, err := Marshal(src, OptBalanced&^OptDense)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	plan, err := batchPlanOf(reflect.TypeFor[batDoc]())
	if err != nil {
		t.Fatalf("batchPlanOf: %v", err)
	}
	slab := newBatchSlab()
	defer slab.release()
	rows := func(n int) unsafe.Pointer { return slab.takeRows(n * int(plan.stride)) }

	n, ok, err := tryDecodeBatchRowMajor(data, plan, slab, rows)
	if err != nil {
		t.Fatalf("tryDecodeBatchRowMajor: unexpected err %v", err)
	}
	if ok {
		t.Fatalf("tryDecodeBatchRowMajor: ok = true, want false (skeleton must still delegate to mirror)")
	}
	if n != 0 {
		t.Fatalf("tryDecodeBatchRowMajor: n = %d, want 0 when ok=false", n)
	}
}
