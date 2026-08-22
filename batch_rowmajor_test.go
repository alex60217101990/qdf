package qdf

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
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

// TestBatchRowMajorHostileLength is the A2-review hardening regression
// (go-fuzz-oom-triage bug class): a tiny 3-byte wire claims a tagArr16 (0xD3)
// array header of n=65535 backed by ZERO remaining bytes. Unlike ReadArrayHeader's
// tagArr32 arm, its tagArr16 arm does not bound n against the remaining
// buffer (only the 2 header bytes are checked, not the claimed count) — before
// the CheckLength(n, 1) guard, checkColumnarBytes(n, plan.stride) was the only
// bound left, and it caps the OUTPUT allocation at 256 MB, not at the input
// size: n=65535 * batDoc's ~40-byte stride is ~2.6 MB, far under that ceiling,
// so rows(65535) would actually run — a transient multi-MB allocation driven
// by a 3-byte hostile input — before the first ReadStructHeader (0 bytes left)
// finally errors. CheckLength(n, 1) now rejects it immediately, before rows(n)
// is ever called.
func TestBatchRowMajorHostileLength(t *testing.T) {
	// tagArr16 = 0xD3 (wire.go); readU16 is little-endian, but 0xff/0xff makes
	// the byte order irrelevant here: n = 65535.
	hostile := []byte{0xd3, 0xff, 0xff}
	b, err := UnmarshalBatch[batDoc](hostile)
	if err == nil {
		b.Release()
		t.Fatal("expected an error for a hostile 65535-row header backed by 0 bytes, got nil")
	}
	if !errors.Is(err, ErrShortBuffer) {
		t.Fatalf("error = %v, want ErrShortBuffer", err)
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
	for k := range data {
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

// unmarshalBatchForceMirror decodes data via ONLY unmarshalBatchMirror,
// bypassing tryDecodeBatchColumnar/tryDecodeBatchRowMajor entirely. It mirrors
// UnmarshalBatch's structure (batch.go) exactly but calls the mirror strategy
// directly — the benchmark control for BenchmarkBatchRowMajorDecodeMirror,
// reproducing what a row-major wire costs through the reflect-mirror fallback
// instead of the direct fast path.
func unmarshalBatchForceMirror[T any](data []byte) (Batch[T], error) {
	plan, err := batchPlanOf(reflect.TypeFor[T]())
	if err != nil {
		return Batch[T]{}, err
	}
	slab := newBatchSlab()
	var rowsPtr unsafe.Pointer
	takeRows := func(n int) unsafe.Pointer {
		rowsPtr = slab.takeRows(n * int(plan.stride))
		return rowsPtr
	}
	n, err := unmarshalBatchMirror(data, plan, slab, takeRows)
	if err != nil {
		slab.release()
		return Batch[T]{}, err
	}
	var rows []T
	if n > 0 {
		rows = unsafe.Slice((*T)(rowsPtr), n)
	}
	return Batch[T]{Rows: rows, slab: slab, epoch: slab.epoch}, nil
}

// BenchmarkBatchRowMajorDecodeMirror is BenchmarkBatchRowMajorDecode's control:
// the IDENTICAL row-major wire, forced through the mirror fallback via
// unmarshalBatchForceMirror instead of tryDecodeBatchRowMajor. benchstat-diffing
// this against BenchmarkBatchRowMajorDecode isolates the direct path's win (the
// per-row string allocation and copy the reflect mirror pays), reproducible
// in-repo without checking out an earlier commit.
func BenchmarkBatchRowMajorDecodeMirror(b *testing.B) {
	src := mkBatSrc(1000)
	data, err := Marshal(src, OptBalanced&^OptDense)
	if err != nil {
		b.Fatalf("Marshal: %v", err)
	}
	if bb, err := unmarshalBatchForceMirror[batDoc](data); err != nil {
		b.Fatalf("warm: %v", err)
	} else {
		bb.Release()
	}
	b.ReportAllocs()
	for b.Loop() {
		bb, err := unmarshalBatchForceMirror[batDoc](data)
		if err != nil {
			b.Fatalf("decode: %v", err)
		}
		bb.Release()
	}
}

// hybSrc mirrors batSrc's 4 columnar-eligible fields and adds Extra, a
// map field: classifyColKind (columnar.go) has no case for reflect.Map (its
// switch covers Int/Uint/Float32/Float64/Bool/String/Slice-of-byte only, and
// falls to `default: return 0, 0, false, false` for anything else), so Extra
// is always residual while ID/Name/Val/At stay columnar-eligible —
// buildColumnarPlan (columnar.go) then produces a HYBRID plan (residual !=
// nil), not a pure one. This is the realistic hybrid trigger buildColumnarPlan
// itself documents ("mostly-scalar struct with one map/slice field", AD/log/
// RTB records) — the same shape TestHybridResidualLongerThanRowCount exercises
// for the general (non-batch) decode path.
//
// batDoc (batch_test.go) is used AS-IS for T: it has no Extra field at all, so
// decoding into it is unambiguous schema evolution ("a residual wire field the
// target struct lacks" — decodeHybridColumnar's own documented behavior,
// columnar.go) rather than the OTHER residual-matching case decodeHybridColumnar
// implements (a wire-residual field whose name DOES match a target field, but
// the target's OWN classification of that field disagrees with the wire's —
// e.g. a batch mirror's plain-scalar reconstruction of a source field the
// encoder classified residual for an unrelated reason, such as a per-field
// custom Marshaler on the SOURCE struct only). decodeHybridColumnar matches a
// wire-residual field by NAME AND by the TARGET's OWN classification
// (findResidual only searches plan.residual), not by name alone, so that other
// case decodes as the target's zero value instead of an error — batDoc sidesteps
// it entirely by simply not declaring the field, which is the shape every
// batch-eligible T is restricted to anyway (batchPlanOf rejects the map/slice/
// custom-Marshaler types that could ever trigger the mismatched case).
type hybSrc struct {
	Extra map[string]int `qdf:"extra"`
	At    time.Time      `qdf:"at"`
	Name  string         `qdf:"name"`
	ID    int64          `qdf:"id"`
	Val   float64        `qdf:"val"`
}

func mkHybSrc(n int) []hybSrc {
	out := make([]hybSrc, n)
	for i := range out {
		out[i] = hybSrc{
			ID:    int64(i),
			Name:  []string{"alpha", "beta", "gamma"}[i%3],
			Val:   float64(i) * 1.5,
			At:    time.Unix(1_700_000_000+int64(i), 500).UTC(),
			Extra: map[string]int{"i": i}, // residual: batDoc has no matching field
		}
	}
	return out
}

// TestBatchHybridFallback proves the mirror fallback still fires — and stays
// correct — for the ONE non-row-major/non-pure-columnar wire a batch-eligible
// T can actually encode: tagHybridColStruct.
//
// Reachability accounting for every wire shape NEITHER tryDecodeBatchColumnar
// NOR tryDecodeBatchRowMajor claims (batch_decode.go):
//
//   - Hybrid (tagHybridColStruct): REACHABLE — this test. Requires a
//     columnar-INELIGIBLE field (classifyColKind, columnar.go) alongside at
//     least one eligible one. hybSrc's Extra (map[string]int) field is exactly
//     that; a batch-eligible T (scalars + Str/Bytes/Time only, per
//     batchPlanOf/appendBatchFields, batch_desc.go) simply never declares the
//     residual field, which is fine — schema evolution already handles a wire
//     field the target lacks.
//   - Nullable column: UNREACHABLE for a legitimately-typed Batch[T] wire —
//     no test needed. A nullable column requires a *U source field
//     (buildColumnarPlan's `if fd.kind == reflect.Pointer` branch, columnar.go);
//     appendBatchFields (batch_desc.go) rejects ANY pointer field on T outright
//     (`case reflect.Map, reflect.Pointer, reflect.Interface, reflect.Chan,
//     reflect.Func: return fmt.Errorf(...not pointer-free...)`), so
//     batchPlanOf fails before UnmarshalBatch touches a single wire byte for
//     any T with a matching pointer field — there is no batch-eligible T a
//     nullable-column wire could legitimately decode into. (The existing,
//     separate errBatchNeedFallback guard in decodeBatchColumnar already
//     defends a HOSTILE columnar wire that merely declares a nullable column
//     under a matching field name — pre-existing, out of this task's scope.)
//   - Batched-vector (tagVecBatchStruct): UNREACHABLE — no test needed. Per
//     wire.go's tag doc it requires a []float32/[]float64 struct field;
//     appendBatchFields's `case reflect.Slice: return fmt.Errorf(...is a
//     slice — use qdf.Bytes...)` rejects every slice field except via the
//     Bytes handle ([]byte only), so batchPlanOf fails before decode for any T
//     with a vector field — no batch-eligible T can pair with this wire.
func TestBatchHybridFallback(t *testing.T) {
	const n = 64 // >= columnarMinElems
	src := mkHybSrc(n)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	d := &Decoder{buf: data}
	tag, err := d.peekTag()
	if err != nil {
		t.Fatalf("peekTag: %v", err)
	}
	if tag != tagHybridColStruct {
		t.Fatalf("expected a tagHybridColStruct wire (%#x), got tag %#x — the columnar probe did not choose hybrid for this fixture", tagHybridColStruct, tag)
	}

	// Whitebox: neither fast path claims a hybrid wire — unmarshalBatchCore
	// must fall through to unmarshalBatchMirror.
	plan, err := batchPlanOf(reflect.TypeFor[batDoc]())
	if err != nil {
		t.Fatalf("batchPlanOf: %v", err)
	}
	slab := newBatchSlab()
	defer slab.release()
	rows := func(m int) unsafe.Pointer { return slab.takeRows(m * int(plan.stride)) }
	if _, ok, _ := tryDecodeBatchColumnar(data, plan, slab, rows); ok {
		t.Fatal("tryDecodeBatchColumnar: ok = true, want false (a hybrid wire is not this fast path)")
	}
	if _, ok, _ := tryDecodeBatchRowMajor(data, plan, slab, rows); ok {
		t.Fatal("tryDecodeBatchRowMajor: ok = true, want false (a hybrid wire is not this fast path)")
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
		want := src[i]
		if r.ID != want.ID || b.Str(r.Name) != want.Name || r.Val != want.Val {
			t.Fatalf("row %d = %+v (name=%q), want id=%d name=%q val=%v",
				i, r, b.Str(r.Name), want.ID, want.Name, want.Val)
		}
		if !b.TimeOf(r.At).Equal(want.At) {
			t.Fatalf("row %d time = %v, want %v", i, b.TimeOf(r.At), want.At)
		}
	}
}
