package qdf

import (
	"math/rand"
	"testing"
)

func FuzzReadLossyVec(f *testing.F) {
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MinCosine(0.999))
	rows := []embedRow{{ID: "s", Emb: make([]float32, 128)}}
	if err := enc.EncodeValue(rows); err == nil {
		f.Add(enc.Bytes())
	}

	// Seed a genuine E8-variant block. Gaussian 256-dim vectors at a tight
	// MaxRelError select the E8 variant, so this seed exercises the variant +
	// coset wire parsing. Built directly via appendLossyVec to guarantee the
	// E8 block is present regardless of the surrounding container framing.
	{
		r := rand.New(rand.NewSource(11))
		vecs := make([][]float64, 64)
		for i := range vecs {
			v := make([]float64, 256)
			for j := range v {
				v[j] = r.NormFloat64()
			}
			vecs[i] = v
		}
		f.Add(appendLossyVec(vecs, false, toBudget(MaxRelError(0.02))))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Decode into both a float32-field and a float64-field struct so the
		// scalar and E8 reconstruction paths for both element types are
		// exercised. Neither may panic or OOM on arbitrary input.
		var out []embedRow
		_ = Unmarshal(data, &out)
		var out64 []struct {
			ID  string
			Emb []float64
		}
		_ = Unmarshal(data, &out64)
	})
}
