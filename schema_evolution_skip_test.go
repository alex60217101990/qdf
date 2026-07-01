package qdf

import (
	"bytes"
	"testing"
)

type skipSrc struct {
	Vals  []int64   // ≥512 regime-shifting → tagPackBlock (0xF0) at top level
	UVals []uint64  // tagPackBlock
	Vec   []float32 // ≥32 under OptLossyVec → tagColVecLossy (0xFD)
	After string    // sentinel after the skipped fields
}
type skipDst struct{ After string } // schema evolution: Vals/UVals/Vec unknown → Skip

// TestSchemaEvolutionSkipBlockLossy guards that Skip() can advance past a
// block-coded (0xF0) and a lossy-vec (0xFD) field, so decoding into a struct
// that no longer has those fields succeeds instead of failing with ErrBadTag.
func TestSchemaEvolutionSkipBlockLossy(t *testing.T) {
	src := skipSrc{After: "sentinel"}
	for i := range 600 {
		v := int64(i)
		if i%50 >= 25 {
			v = int64(i * 1000)
		}
		src.Vals = append(src.Vals, v)
		src.UVals = append(src.UVals, uint64(v))
	}
	for i := range 64 {
		src.Vec = append(src.Vec, float32(i)*0.5)
	}

	for _, opt := range []Options{OptBalanced | OptColumnIndex, OptBalanced | OptLossyVec, OptCompression | OptLossyVec} {
		b, err := Marshal(src, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal: %v", opt, err)
		}
		var dst skipDst
		if err := Unmarshal(b, &dst); err != nil {
			t.Fatalf("opt=%v skip-decode: %v", opt, err)
		}
		if dst.After != "sentinel" {
			t.Fatalf("opt=%v After=%q (cursor desync after skip)", opt, dst.After)
		}
	}

	// Sanity: the data really exercises the tags the fix targets.
	b1, _ := Marshal(src, OptBalanced|OptColumnIndex)
	b2, _ := Marshal(src, OptBalanced|OptLossyVec)
	if bytes.IndexByte(b1, tagPackBlock) < 0 {
		t.Fatal("test data did not emit tagPackBlock (0xF0)")
	}
	if bytes.IndexByte(b2, tagColVecLossy) < 0 {
		t.Fatal("test data did not emit tagColVecLossy (0xFD)")
	}
}

type vbRow struct {
	ID   int64
	Name string    // repeated → interned under OptShapeIntern (state-ref per row)
	Emb  []float32 // equal length ≥ lossyVecMinElems → batched
}
type vbBatchSrc struct {
	Rows  []vbRow // ≥ columnarMinElems rows w/ a batchable vector field → tagVecBatchStruct (0xFE)
	After string  // sentinel after the skipped field
}
type vbBatchDst struct{ After string } // schema evolution: Rows unknown → Skip

// TestSchemaEvolutionSkipVecBatchStruct guards that Skip() can advance past a
// tagVecBatchStruct (0xFE) field — a []struct with a batched vector field under
// OptLossyVec — including walking its per-row non-batched fields (ID, interned
// Name) so their intern/shape state is replayed and the trailing sentinel stays
// in sync. Before the fix Skip() fell through to ErrBadTag on 0xFE.
func TestSchemaEvolutionSkipVecBatchStruct(t *testing.T) {
	var src vbBatchSrc
	src.After = "sentinel"
	names := []string{"alpha", "beta", "gamma"}
	for i := range 24 {
		var r vbRow
		r.ID = int64(i)
		r.Name = names[i%len(names)]
		for j := range 64 {
			r.Emb = append(r.Emb, float32(i*64+j)*0.01)
		}
		src.Rows = append(src.Rows, r)
	}

	opt := OptLossyVec | OptDense | OptShapeIntern
	b, err := Marshal(src, opt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.IndexByte(b, tagVecBatchStruct) < 0 {
		t.Fatal("test data did not emit tagVecBatchStruct (0xFE)")
	}
	var dst vbBatchDst
	if err := Unmarshal(b, &dst); err != nil {
		t.Fatalf("skip-decode: %v", err)
	}
	if dst.After != "sentinel" {
		t.Fatalf("After=%q (cursor desync after skipping 0xFE)", dst.After)
	}
}
