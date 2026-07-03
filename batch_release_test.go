package qdf

import "testing"

// TestBatchSteadyStateZeroAlloc drives the decode+Release loop repeatedly on
// a WARM slab pool: once the slab's buf and rowsBuf both have enough
// capacity from the first decode, a subsequent UnmarshalBatch+Release cycle
// must not allocate the rows backing (takeRows reuses rowsBuf), the slab
// header (sync.Pool-recycled), or the Batch[T] header (returned by value —
// no heap allocation for the generic wrapper struct itself).
//
// Budget note: isolating takeRows and the slab get/release confirms BOTH are
// 0-alloc once warm (verified with a standalone probe during development).
// The remaining allocations are the columnar decoder's per-message INLINE
// shape declaration (readColShape's idv==0 branch: make([]string,cnt) +
// make([]colKind,cnt) + one string(bytes) conversion per field name) — a
// cost every INDEPENDENT columnar decode pays, batch or not (a plain
// Unmarshal into []batSrc from the same wire measures ~12 allocs/op, not 0).
// That floor is orthogonal to rows/slab pooling and out of Task 5's scope
// (cross-message shape caching would be needed to remove it — see
// TestBatchColumnarAllocBudget, which documents the same floor at ≤13 for
// the columnar fast path as a whole). Measured ~8 here; budget set with a
// couple of slots of slack, same convention as TestBatchColumnarAllocBudget.
func TestBatchSteadyStateZeroAlloc(t *testing.T) {
	if raceEnabled {
		t.Skip("alloc budgets are not measured under -race (sync.Pool churn instrumentation)")
	}
	src := mkBatSrc(512)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	// Warm: first decode allocates slab+rows; Release recycles both.
	b, err := UnmarshalBatch[batDoc](data)
	if err != nil {
		t.Fatal(err)
	}
	b.Release()
	allocs := testing.AllocsPerRun(20, func() {
		b, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			t.Fatal(err)
		}
		b.Release()
	})
	// Measured 2 after the zero-alloc shape reader (batchReadColShape matches
	// column names against the plan as []byte views instead of materializing
	// []string + per-name copies) plus pooled rows/slab. Allow slack of 1.
	if allocs > 3 {
		t.Fatalf("steady-state allocs = %v, want <= 3 (rows/slab/shape pooling regressed?)", allocs)
	}
}
