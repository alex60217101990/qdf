package main

import (
	"fmt"
	"testing"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

func TestZZBatchVsPerVector(t *testing.T) {
	const n, dim = 2000, 256
	corpus := loadSynthetic(n, dim, 42)
	normalizeRows(corpus)
	b := vecquant.Budget{Kind: vecquant.KindRelError, Val: 0.05}

	// (a) batched: whole corpus = one count=N block (what the bench reports).
	batched := lossyVecWireBytes(vecquant.Encode(corpus, b))
	// (b) per-vector: each vector its own count=1 block (what qdf does in prod).
	perVec := 0
	for i := range corpus {
		perVec += lossyVecWireBytes(vecquant.Encode(corpus[i:i+1], b))
	}
	fmt.Printf("\nbatched (count=%d):  %.2f B/vec\n", n, float64(batched)/n)
	fmt.Printf("per-vector (count=1): %.2f B/vec\n", float64(perVec)/n)
	fmt.Printf("per-vector overhead:  +%.2f B/vec (+%.1f%%)\n",
		float64(perVec-batched)/n, 100*float64(perVec-batched)/float64(batched))
}
