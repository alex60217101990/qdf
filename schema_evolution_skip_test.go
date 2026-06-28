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
