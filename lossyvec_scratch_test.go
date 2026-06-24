package qdf

import (
	"math/rand"
	"testing"
)

func TestToF64Into(t *testing.T) {
	s := []float32{1, 2, 3, -4.5, 0}
	buf := make([]float64, 0, 8)
	out := toF64Into(s, buf)
	if len(out) != len(s) {
		t.Fatalf("len %d != %d", len(out), len(s))
	}
	for i := range s {
		if out[i] != float64(s[i]) {
			t.Fatalf("i=%d %v != %v", i, out[i], float64(s[i]))
		}
	}
	// reuse: a shorter slice must not leak stale tail.
	out2 := toF64Into([]float32{9, 8}, out)
	if len(out2) != 2 || out2[0] != 9 || out2[1] != 8 {
		t.Fatalf("reuse leaked: %v", out2)
	}
}

func TestLossyVecF64NoCallerMutation(t *testing.T) {
	orig := make([]float64, 64)
	for i := range orig {
		orig[i] = float64(i) + 0.25
	}
	cp := append([]float64(nil), orig...)
	rows := []embedRowE8{{ID: "x", Emb: orig}}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.02))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatal(err)
	}
	for i := range orig {
		if orig[i] != cp[i] {
			t.Fatalf("caller slice mutated at %d: %v != %v", i, orig[i], cp[i])
		}
	}
}

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
