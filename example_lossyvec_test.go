package qdf_test

import (
	"fmt"
	"math"

	"github.com/alex60217101990/qdf"
)

// ExampleEncoder_lossyVector shows the opt-in lossy vector codec. With
// OptLossyVec and a fidelity budget, []float32/[]float64 embedding fields are
// rotated, quantized, and entropy-coded to a fraction of their raw size; the
// budget (here cosine similarity >= 0.999) bounds the approximation. Decoding
// needs no special flag — the 0xFD tag is recognized automatically.
func ExampleEncoder_lossyVector() {
	type Doc struct {
		ID  string
		Emb []float32
	}

	// 64 deterministic embedding vectors of dimension 256.
	docs := make([]Doc, 64)
	for i := range docs {
		v := make([]float32, 256)
		for j := range v {
			v[j] = float32(math.Sin(float64(i*256+j) * 0.01))
		}
		docs[i] = Doc{ID: fmt.Sprintf("doc-%d", i), Emb: v}
	}

	enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
	enc.SetVectorBudget(qdf.MinCosine(0.999))
	if err := enc.EncodeValue(docs); err != nil {
		panic(err)
	}
	data := enc.Bytes()

	var out []Doc
	if err := qdf.Unmarshal(data, &out); err != nil {
		panic(err)
	}

	// Cosine similarity of the first reconstructed vector vs the original.
	var dot, na, nb float64
	for j := range docs[0].Emb {
		a, b := float64(docs[0].Emb[j]), float64(out[0].Emb[j])
		dot += a * b
		na += a * a
		nb += b * b
	}
	cosine := dot / (math.Sqrt(na) * math.Sqrt(nb))

	rawF32 := len(docs) * 256 * 4
	fmt.Println("decoded count:", len(out))
	fmt.Println("smaller than raw f32:", len(data) < rawF32)
	fmt.Println("cosine >= 0.999:", cosine >= 0.999)

	// Output:
	// decoded count: 64
	// smaller than raw f32: true
	// cosine >= 0.999: true
}

// ExampleMaxRelError shows the relative-error budget. MaxRelError(eps) bounds
// the per-vector relative L2 error; a tighter eps spends more bytes for a
// closer reconstruction. NaN/Inf values are preserved bit-exactly via an
// exception list, independent of the lossy budget.
func ExampleMaxRelError() {
	type Row struct {
		V []float64
	}

	v := make([]float64, 128)
	for i := range v {
		v[i] = float64(i) * 0.5
	}
	v[3] = math.NaN()
	v[7] = math.Inf(1)
	row := []Row{{V: v}}

	enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
	enc.SetVectorBudget(qdf.MaxRelError(0.01)) // <= 1% relative L2 error
	if err := enc.EncodeValue(row); err != nil {
		panic(err)
	}

	var out []Row
	if err := qdf.Unmarshal(enc.Bytes(), &out); err != nil {
		panic(err)
	}

	fmt.Println("NaN preserved:", math.IsNaN(out[0].V[3]))
	fmt.Println("+Inf preserved:", math.IsInf(out[0].V[7], 1))

	// finite values within the 1% budget
	var se, ne float64
	for i := range v {
		if math.IsNaN(v[i]) || math.IsInf(v[i], 0) {
			continue
		}
		d := v[i] - out[0].V[i]
		se += d * d
		ne += v[i] * v[i]
	}
	fmt.Println("rel error <= 1%:", math.Sqrt(se/ne) <= 0.01)

	// Output:
	// NaN preserved: true
	// +Inf preserved: true
	// rel error <= 1%: true
}
