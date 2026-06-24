package qdf

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// hybridRow is a hybrid-columnar struct: int64 and float64 are columnar-eligible
// while string is not. Under OptBalanced the intern-aware probe fires for the
// string column, pulling the struct into the hybrid columnar container
// (tagColStruct). With n=32 the float64 column has exactly lossyVecMinElems
// elements, so encodeSliceFloat64 emits tagColVecLossy under OptLossyVec.
// Decoding that column calls decodeSliceFloat64Into, which (before the fix)
// did not handle tagColVecLossy and returned ErrTypeMismatch.
type hybridRow struct {
	ID    int64
	Tag   string
	Score float64
}

// TestLossyVecColumnarFloat64 is the regression test for the encode/decode
// asymmetry in decodeSliceFloat64Into: it must handle tagColVecLossy (0xFD).
func TestLossyVecColumnarFloat64(t *testing.T) {
	const nRows = 32

	rows := make([]hybridRow, nRows)
	for i := range rows {
		rows[i] = hybridRow{
			ID:    int64(i),
			Tag:   fmt.Sprintf("host-%d", i%5),
			Score: math.Sin(float64(i) * 0.3),
		}
	}

	opts := OptBalanced | OptColumnIndex | OptLossyVec
	enc := NewEncoderWith(opts)
	enc.SetVectorBudget(MinCosine(0.999))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}
	data := enc.Bytes()

	// Confirm the test actually drives the columnar path with a lossy block.
	if !bytes.Contains(data, []byte{tagColStruct}) {
		t.Fatal("expected tagColStruct (0xEF) in payload — columnar path not taken; test invalid")
	}
	count0xFD := 0
	for _, b := range data {
		if b == tagColVecLossy {
			count0xFD++
		}
	}
	if count0xFD == 0 {
		t.Fatal("expected at least one tagColVecLossy (0xFD) in payload — lossy vec not fired; test invalid")
	}
	t.Logf("tagColStruct present, tagColVecLossy (0xFD) count=%d — columnar+lossy path confirmed", count0xFD)

	var out []hybridRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(out) != nRows {
		t.Fatalf("row count: got %d, want %d", len(out), nRows)
	}

	for i, orig := range rows {
		if out[i].ID != orig.ID {
			t.Errorf("row %d: ID got %d want %d", i, out[i].ID, orig.ID)
		}
		if out[i].Tag != orig.Tag {
			t.Errorf("row %d: Tag got %q want %q", i, out[i].Tag, orig.Tag)
		}
		// Score is a scalar float64 re-assembled from a lossy column.
		// The lossy codec operates on the full 32-element column as a single
		// vector, so per-element fidelity is bounded by the Hadamard rotation +
		// quantization step; a 30% relative error bound is generous enough to
		// survive that while still catching a complete decode failure (zeros,
		// garbage, or the wrong column).
		orig64 := orig.Score
		got64 := out[i].Score
		// Use absolute error for small values (|orig| < 0.1) to avoid divide-
		// by-near-zero blowing up relative error on a lossy decode.
		if math.Abs(orig64) < 0.1 {
			absErr := math.Abs(got64 - orig64)
			if absErr > 0.15 {
				t.Errorf("row %d: Score abs error %.4f > 0.15 (orig=%v got=%v)", i, absErr, orig64, got64)
			}
			continue
		}
		rel := math.Abs((got64 - orig64) / orig64)
		if rel > 0.30 {
			t.Errorf("row %d: Score rel error %.4f > 0.30 (orig=%v got=%v)", i, rel, orig64, got64)
		}
	}
}
