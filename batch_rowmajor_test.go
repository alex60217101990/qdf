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

// TestTryDecodeBatchRowMajorDecodes is a whitebox test asserting this task's
// contract directly: on a row-major struct-slice wire, tryDecodeBatchRowMajor
// returns ok=true with n rows decoded straight into T + slab (no mirror), and
// every row matches the source.
func TestTryDecodeBatchRowMajorDecodes(t *testing.T) {
	for _, n := range []int{0, 1, 64, 1000} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			src := mkBatSrc(n)
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
			var base unsafe.Pointer
			rows := func(m int) unsafe.Pointer {
				base = slab.takeRows(m * int(plan.stride))
				return base
			}

			gotN, ok, err := tryDecodeBatchRowMajor(data, plan, slab, rows)
			if err != nil {
				t.Fatalf("tryDecodeBatchRowMajor: unexpected err %v", err)
			}
			if !ok {
				t.Fatalf("tryDecodeBatchRowMajor: ok = false, want true (direct path must handle a row-major wire)")
			}
			if gotN != n {
				t.Fatalf("tryDecodeBatchRowMajor: n = %d, want %d", gotN, n)
			}
			if n == 0 {
				return
			}
			b := Batch[batDoc]{Rows: unsafe.Slice((*batDoc)(base), n), slab: slab, epoch: slab.epoch}
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

// TestBatchRowMajorTruncated asserts every prefix of a marshaled row-major wire
// returns an error (never panics) — the unsafe per-field scatter must guard
// every offset so a truncated/hostile payload cannot read past the buffer.
func TestBatchRowMajorTruncated(t *testing.T) {
	src := mkBatSrc(64)
	data, err := Marshal(src, OptBalanced&^OptDense)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for k := range len(data) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("prefix len %d panicked: %v", k, r)
				}
			}()
			b, err := UnmarshalBatch[batDoc](data[:k])
			if err == nil {
				// A truncated struct-slice may occasionally decode to a valid
				// SHORTER batch (a whole-row prefix boundary); that is fine as
				// long as it did not panic and the rows it did produce resolve.
				for _, r := range b.Rows {
					_ = b.Str(r.Name)
				}
				b.Release()
			}
		}()
	}
}

// TestBatchRowMajorKindMismatch corrupts a field's value tag on the wire and
// asserts the direct path rejects it (a decode error) rather than scattering a
// reinterpreted value — the outer array-header tag does not prove the element
// field types.
func TestBatchRowMajorKindMismatch(t *testing.T) {
	src := mkBatSrc(4)
	data, err := Marshal(src, OptBalanced&^OptDense)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Find the first "val" (float64) value byte on the wire and overwrite its
	// float64 tag with a bool tag: ReadFloat64 must then return ErrTypeMismatch.
	d := &Decoder{buf: data}
	if _, err := d.peekTag(); err != nil {
		t.Fatalf("peekTag: %v", err)
	}
	if _, err := d.ReadArrayHeader(); err != nil {
		t.Fatalf("ReadArrayHeader: %v", err)
	}
	// Walk the first row to the "val" field's value tag.
	names, plainN, shaped, err := d.ReadStructHeader()
	if err != nil {
		t.Fatalf("ReadStructHeader: %v", err)
	}
	pos := -1
	read1 := func(name string) {
		switch name {
		case "id":
			_, _ = d.ReadInt()
		case "val":
			pos = d.i // tag byte of the float64 value
			_, _ = d.ReadFloat64()
		case "name":
			_, _ = d.ReadString()
		case "at":
			_, _, _ = d.ReadTimestamp()
		default:
			_ = d.Skip()
		}
	}
	if shaped {
		for _, name := range names {
			read1(name)
		}
	} else {
		for range plainN {
			kb, _ := d.ReadStringBytes()
			read1(string(kb))
		}
	}
	if pos < 0 {
		t.Fatalf("could not locate the val field on the wire")
	}
	corrupt := make([]byte, len(data))
	copy(corrupt, data)
	corrupt[pos] = byte(tagFalse) // was tagFloat64
	if _, err := UnmarshalBatch[batDoc](corrupt); err == nil {
		t.Fatalf("expected a decode error for a bool where a float64 is required, got nil")
	}
}

// BenchmarkBatchRowMajorDecode measures the direct path: decode a 1000-row
// row-major wire + Release, steady-state (pools warmed). Compare its allocs/ns
// against the mirror fallback that decoded the same wire before this change.
func BenchmarkBatchRowMajorDecode(b *testing.B) {
	src := mkBatSrc(1000)
	data, err := Marshal(src, OptBalanced&^OptDense)
	if err != nil {
		b.Fatalf("Marshal: %v", err)
	}
	// Warm the decoder/slab/mirror pools.
	if bb, err := UnmarshalBatch[batDoc](data); err != nil {
		b.Fatalf("warm: %v", err)
	} else {
		bb.Release()
	}
	b.ReportAllocs()
	for b.Loop() {
		bb, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			b.Fatalf("decode: %v", err)
		}
		bb.Release()
	}
}
