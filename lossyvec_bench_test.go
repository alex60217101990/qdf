package qdf

import (
	"math/rand"
	"testing"
)

func benchEmbF32(n, dim int) []embedRow {
	r := rand.New(rand.NewSource(1))
	rows := make([]embedRow, n)
	for i := range rows {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(r.NormFloat64())
		}
		rows[i] = embedRow{ID: "doc", Emb: v}
	}
	return rows
}

// BenchmarkLossyEncodeWarm measures the realistic server hot path: repeated
// pooled Marshal calls under OptLossyVec, where the encoder pool reuses the
// vector-codec scratch across calls. This is where the allocation-reduction
// pass pays off (scratch reuse + streamed verify metric).
func BenchmarkLossyEncodeWarm(b *testing.B) {
	rows := benchEmbF32(256, 768)
	b.ReportAllocs()

	for b.Loop() {
		if _, err := Marshal(rows, OptBalanced|OptLossyVec); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLossyEncodeCold measures a one-off encode with a fresh encoder (no
// scratch reuse): the within-encode allocation reduction (streamed verify
// metric, no [][]float64 materialization).
func BenchmarkLossyEncodeCold(b *testing.B) {
	rows := benchEmbF32(256, 768)
	b.ReportAllocs()

	for b.Loop() {
		enc := NewEncoderWith(OptBalanced | OptLossyVec)
		enc.SetVectorBudget(MaxRelError(0.02))
		if err := enc.EncodeValue(rows); err != nil {
			b.Fatal(err)
		}
		_ = enc.Bytes()
	}
}

func BenchmarkLossyDecode(b *testing.B) {
	rows := benchEmbF32(256, 768)
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.02))
	_ = enc.EncodeValue(rows)
	data := append([]byte(nil), enc.Bytes()...)
	b.ReportAllocs()

	for b.Loop() {
		var out []embedRow
		if err := Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}
