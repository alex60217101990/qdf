package qdf

import "testing"

func FuzzReadLossyVec(f *testing.F) {
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MinCosine(0.999))
	rows := []embedRow{{ID: "s", Emb: make([]float32, 128)}}
	if err := enc.EncodeValue(rows); err == nil {
		f.Add(enc.Bytes())
	}

	// Seed an E8-variant payload (256-dim ensures E8 is eligible/selected).
	{
		enc2 := NewEncoderWith(OptBalanced | OptLossyVec)
		enc2.SetVectorBudget(MinCosine(0.999))
		rows2 := []struct {
			ID  string
			Emb []float64
		}{{ID: "s", Emb: make([]float64, 256)}}
		for j := range rows2[0].Emb {
			rows2[0].Emb[j] = float64(j%7) * 0.5
		}
		if err := enc2.EncodeValue(rows2); err == nil {
			f.Add(enc2.Bytes())
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var out []embedRow
		_ = Unmarshal(data, &out) // must never panic / OOM
	})
}
