// embeddings demonstrates the opt-in lossy vector codec for AI embeddings.
//
// Embeddings don't need bit-exact storage — only their nearest-neighbor
// geometry has to survive. Under OptLossyVec, qdf quantizes the []float32 /
// []float64 vector columns of a []struct to a fidelity budget you set
// (min cosine, max relative error, or target SNR) and entropy-codes the result,
// with a never-larger fallback to the lossless body.
//
//	go run ./examples/embeddings
package main

import (
	"fmt"
	"math"

	"github.com/alex60217101990/qdf"
)

type Doc struct {
	ID  string    `qdf:"id"`
	Emb []float32 `qdf:"emb"`
}

func main() {
	const n, dim = 2000, 128
	docs := make([]Doc, n)
	for i := range docs {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(math.Sin(float64(i*dim+j) * 0.017))
		}
		docs[i] = Doc{ID: fmt.Sprintf("doc-%05d", i), Emb: v}
	}

	// Lossless baseline.
	lossless, err := qdf.Marshal(docs, qdf.OptBalanced)
	if err != nil {
		panic(err)
	}

	// Lossy vector codec targeting a minimum cosine similarity of 0.99.
	enc := qdf.NewEncoderWith(qdf.OptBalanced | qdf.OptLossyVec)
	enc.SetVectorBudget(qdf.MinCosine(0.99))
	if err := enc.EncodeValue(docs); err != nil {
		panic(err)
	}
	lossy := enc.Bytes()

	// Decode and measure the worst-case cosine similarity actually achieved.
	var back []Doc
	if err := qdf.Unmarshal(lossy, &back); err != nil {
		panic(err)
	}
	worst := 1.0
	for i := range docs {
		if c := cosine(docs[i].Emb, back[i].Emb); c < worst {
			worst = c
		}
	}

	fmt.Printf("vectors:        %d × %d dims\n", n, dim)
	fmt.Printf("lossless:       %7d bytes\n", len(lossless))
	fmt.Printf("lossy (cos≥.99):%7d bytes  (%.1f%% smaller)\n",
		len(lossy), 100*(1-float64(len(lossy))/float64(len(lossless))))
	fmt.Printf("worst cosine:   %.4f (target 0.99)\n", worst)
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	return dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30)
}
