package qdf

import (
	"math/rand"
	"testing"
)

// TestEncoderScratchReuseByteIdentical proves reusing one Encoder (scratch
// retained) gives the same bytes as a fresh Encoder for the same input.
func TestEncoderScratchReuseByteIdentical(t *testing.T) {
	mk := func(seed int64) []embedRowE8 {
		r := rand.New(rand.NewSource(seed))
		rows := make([]embedRowE8, 40)
		for i := range rows {
			v := make([]float64, 256)
			for j := range v {
				v[j] = r.NormFloat64()
			}
			rows[i] = embedRowE8{ID: "d", Emb: v}
		}
		return rows
	}
	target := mk(1)

	encFresh := NewEncoderWith(OptBalanced | OptLossyVec)
	encFresh.SetVectorBudget(MaxRelError(0.02))
	if err := encFresh.EncodeValue(target); err != nil {
		t.Fatal(err)
	}
	want := append([]byte(nil), encFresh.Bytes()...)

	encReuse := NewEncoderWith(OptBalanced | OptLossyVec)
	encReuse.SetVectorBudget(MaxRelError(0.02))
	// First encode different data to dirty the scratch, then reset and encode target.
	// resetForReuse (not Reset) keeps the configured opts/mode/flags while clearing
	// the output buffer and calling vecScratch.Reset() — exactly what we want here.
	_ = encReuse.EncodeValue(mk(7))
	encReuse.resetForReuse()
	encReuse.SetVectorBudget(MaxRelError(0.02))
	if err := encReuse.EncodeValue(target); err != nil {
		t.Fatal(err)
	}
	got := encReuse.Bytes()

	if string(got) != string(want) {
		t.Fatalf("scratch reuse changed bytes: got %d want %d", len(got), len(want))
	}
}
