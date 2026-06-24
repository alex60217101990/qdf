package qdf

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// embedHybridRow is a hybrid-columnar struct: int64 and string are handled by
// the columnar container (int64 as a column, string as a residual/dict column),
// while Emb []float64 is a non-[]byte slice — classifyColKind rejects it, so it
// becomes a RESIDUAL field encoded row-major per record via the slice codec.
// Under OptLossyVec that genuine vector field is lossy-eligible and emits
// tagColVecLossy (0xFD). Decoding it exercises decodeSliceFloat64 inside the
// columnar (hybrid) container.
type embedHybridRow struct {
	ID  int64
	Tag string
	Emb []float64
}

// TestLossyVecColumnarFloat64 regression-tests that the lossy 0xFD block
// round-trips when a genuine SLICE-typed float64 vector field rides inside the
// columnar (hybrid) container — the residual slice field reaches the lossy
// codec and decode must handle 0xFD there.
func TestLossyVecColumnarFloat64(t *testing.T) {
	// 64 rows of low-cardinality scalar columns make the columnar probe predict
	// a gain and route the struct through the hybrid container; the Emb slice
	// field then encodes as a residual lossy block (0xFD) per record.
	const nRows = 64
	const dim = 32 // >= lossyVecMinElems so the lossy codec fires

	rows := make([]embedHybridRow, nRows)
	for i := range rows {
		emb := make([]float64, dim)
		for j := range emb {
			emb[j] = math.Sin(float64(i*dim+j) * 0.05)
		}
		rows[i] = embedHybridRow{
			ID:  int64(i % 4),
			Tag: fmt.Sprintf("host-%d", i%5),
			Emb: emb,
		}
	}

	opts := OptBalanced | OptColumnIndex | OptLossyVec
	enc := NewEncoderWith(opts)
	enc.SetVectorBudget(MinCosine(0.999))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}
	data := enc.Bytes()

	// Confirm the test drives the columnar (hybrid) path with at least one lossy
	// block. The slice field is residual, so the container is tagHybridColStruct.
	if !bytes.Contains(data, []byte{tagHybridColStruct}) {
		t.Fatal("expected tagHybridColStruct in payload — hybrid columnar path not taken; test invalid")
	}
	count0xFD := bytes.Count(data, []byte{tagColVecLossy})
	if count0xFD == 0 {
		t.Fatal("expected at least one tagColVecLossy (0xFD) — lossy vec not fired on the slice field; test invalid")
	}
	t.Logf("tagColStruct present, tagColVecLossy (0xFD) count=%d — columnar+lossy slice-field path confirmed", count0xFD)

	var out []embedHybridRow
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
		if len(out[i].Emb) != dim {
			t.Fatalf("row %d: Emb len got %d want %d", i, len(out[i].Emb), dim)
		}
		// Emb is lossy; bound per-vector cosine similarity against the budget.
		var dot, na, nb float64
		for j := range orig.Emb {
			dot += orig.Emb[j] * out[i].Emb[j]
			na += orig.Emb[j] * orig.Emb[j]
			nb += out[i].Emb[j] * out[i].Emb[j]
		}
		cos := dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30)
		if cos < 0.99 {
			t.Errorf("row %d: Emb cosine %.5f < 0.99 (lossy decode degraded)", i, cos)
		}
	}
}
