package qdf

import (
	"math"
	"testing"
)

type embedRow struct {
	ID  string
	Emb []float32
}

func TestLossyVecMarshalField(t *testing.T) {
	rows := make([]embedRow, 32)
	for i := range rows {
		v := make([]float32, 256)
		for j := range v {
			v[j] = float32(math.Sin(float64(i*256+j) * 0.013))
		}
		rows[i] = embedRow{ID: "doc", Emb: v}
	}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MinCosine(0.999))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("encode: %v", err)
	}
	data := enc.Bytes()
	// must be smaller than raw f32 of the embeddings alone
	if len(data) >= 32*256*4 {
		t.Fatalf("not compressed: %d", len(data))
	}
	var out []embedRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != len(rows) {
		t.Fatalf("len %d", len(out))
	}
	for i := range rows {
		var dot, na, nb float64
		for j := range rows[i].Emb {
			a, b := float64(rows[i].Emb[j]), float64(out[i].Emb[j])
			dot += a * b
			na += a * a
			nb += b * b
		}
		if dot/(math.Sqrt(na)*math.Sqrt(nb)) < 0.999*0.999 {
			t.Fatalf("i=%d cosine below target", i)
		}
	}
}

func TestLossyVecOffByDefaultExact(t *testing.T) {
	// Without OptLossyVec the float slice must round-trip bit-exact.
	rows := []embedRow{{ID: "x", Emb: []float32{1.5, -2.25, 0, float32(math.Inf(1))}}}
	data, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out []embedRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for j := range rows[0].Emb {
		if math.Float32bits(rows[0].Emb[j]) != math.Float32bits(out[0].Emb[j]) {
			t.Fatalf("j=%d not bit-exact: %v vs %v", j, rows[0].Emb[j], out[0].Emb[j])
		}
	}
}
