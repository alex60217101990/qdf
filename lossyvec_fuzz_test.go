package qdf

import "testing"

func FuzzReadLossyVec(f *testing.F) {
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MinCosine(0.999))
	rows := []embedRow{{ID: "s", Emb: make([]float32, 128)}}
	if err := enc.EncodeValue(rows); err == nil {
		f.Add(enc.Bytes())
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var out []embedRow
		_ = Unmarshal(data, &out) // must never panic / OOM
	})
}
