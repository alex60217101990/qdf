package qdf

import (
	"math"
	"testing"
)

type embedRowE8 struct {
	ID  string
	Emb []float64
}

func TestE8OptionMatrixRoundTrip(t *testing.T) {
	bundles := []Options{OptBalanced | OptLossyVec, OptCompression | OptLossyVec}
	budgets := []VectorBudget{MaxRelError(0.02), MinCosine(0.999), TargetSNR(40)}
	for _, opt := range bundles {
		for _, bud := range budgets {
			rows := make([]embedRowE8, 48)
			for i := range rows {
				v := make([]float64, 256) // pdim 256 -> E8 eligible
				for j := range v {
					v[j] = math.Sin(float64(i*256+j) * 0.011)
				}
				rows[i] = embedRowE8{ID: "d", Emb: v}
			}
			enc := NewEncoderWith(opt)
			enc.SetVectorBudget(bud)
			// Brief specifies enc.Marshal(rows); actual API is EncodeValue+Bytes.
			if err := enc.EncodeValue(rows); err != nil {
				t.Fatalf("opt=%v marshal: %v", opt, err)
			}
			data := enc.Bytes()
			var out []embedRowE8
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("opt=%v unmarshal: %v", opt, err)
			}
			if len(out) != len(rows) || len(out[0].Emb) != 256 {
				t.Fatalf("opt=%v shape lost", opt)
			}
		}
	}
}

func TestE8AutoNeverWorseVsLossless(t *testing.T) {
	// The lossy auto codec (which keeps the smaller of scalar/E8 per column)
	// must never produce a larger payload than the lossless encoding of the
	// same rows, and must round-trip cleanly. Uses 256-dim vectors so E8 is
	// eligible at the tight budget.
	rows := make([]embedRowE8, 64)
	for i := range rows {
		v := make([]float64, 256)
		for j := range v {
			v[j] = math.Sin(float64(i*256+j) * 0.03)
		}
		rows[i] = embedRowE8{ID: "d", Emb: v}
	}

	lossless, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("lossless marshal: %v", err)
	}

	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.02))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("lossy marshal: %v", err)
	}
	lossy := enc.Bytes()

	if len(lossy) > len(lossless) {
		t.Fatalf("never-worse violated: lossy %d > lossless %d", len(lossy), len(lossless))
	}

	var out []embedRowE8
	if err := Unmarshal(lossy, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != len(rows) {
		t.Fatalf("rows lost")
	}
}
