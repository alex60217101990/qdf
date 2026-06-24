package main

import (
	"math"
	"math/rand"
	"testing"
)

func TestNaiveScalarRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	v := make([]float64, 256)
	for i := range v {
		v[i] = r.NormFloat64()
	}
	q, mn, delta := naiveScalarEncode(v, 8) // 8-bit
	got := naiveScalarDecode(q, mn, delta)
	var se, ne float64
	for i := range v {
		d := v[i] - got[i]
		se += d * d
		ne += v[i] * v[i]
	}
	if math.Sqrt(se/ne) > 0.05 {
		t.Fatalf("8-bit naive rel error too high: %v", math.Sqrt(se/ne))
	}
}

func TestPQRoundTripImprovesWithBits(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	data := make([][]float64, 500)
	for i := range data {
		data[i] = make([]float64, 64)
		for j := range data[i] {
			data[i][j] = r.NormFloat64()
		}
	}
	e4 := pqAvgRelError(data, 8, 4) // 8 subspaces, 4-bit codebooks
	e6 := pqAvgRelError(data, 8, 6) // richer codebooks
	if !(e6 < e4) {
		t.Fatalf("more PQ bits should reduce error: e6=%v e4=%v", e6, e4)
	}
}
