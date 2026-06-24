package qdf

import (
	"math"
	"testing"
)

func TestLossyVecOptionMatrix(t *testing.T) {
	budgets := []VectorBudget{MaxRelError(0.02), MinCosine(0.999), TargetSNR(40)}
	bundles := []Options{OptBalanced | OptLossyVec, OptCompression | OptLossyVec}
	for _, opt := range bundles {
		for _, bud := range budgets {
			rows := make([]embedRow, 40)
			for i := range rows {
				v := make([]float32, 384)
				for j := range v {
					v[j] = float32(math.Cos(float64(i*384+j) * 0.007))
				}
				rows[i] = embedRow{ID: "d", Emb: v}
			}
			enc := NewEncoderWith(opt)
			enc.SetVectorBudget(bud)
			if err := enc.EncodeValue(rows); err != nil {
				t.Fatalf("opt=%v bud=%v encode: %v", opt, bud, err)
			}
			data := enc.Bytes()
			var out []embedRow
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("opt=%v unmarshal: %v", opt, err)
			}
			if len(out) != len(rows) || len(out[0].Emb) != 384 {
				t.Fatalf("opt=%v shape lost", opt)
			}
		}
	}
}

func TestLossyVecNaNInfException(t *testing.T) {
	// NaN/Inf cannot be quantized; the codec must preserve them via an
	// exception path (or refuse lossy for that vector and store it raw).
	rows := []embedRow{{ID: "x", Emb: []float32{
		1, 2, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 0,
	}}}
	// pad to >= lossyVecMinElems so the codec engages
	for len(rows[0].Emb) < 64 {
		rows[0].Emb = append(rows[0].Emb, 0.5)
	}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.02))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("encode: %v", err)
	}
	data := enc.Bytes()
	var out []embedRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !math.IsNaN(float64(out[0].Emb[2])) ||
		!math.IsInf(float64(out[0].Emb[3]), 1) ||
		!math.IsInf(float64(out[0].Emb[4]), -1) {
		t.Fatalf("NaN/Inf not preserved: %v", out[0].Emb[:5])
	}
}
