package qdf

import (
	"math"
	"testing"
)

func TestLossyVecWireRoundTrip(t *testing.T) {
	vecs := make([][]float64, 64)
	for i := range vecs {
		v := make([]float64, 256)
		for j := range v {
			v[j] = math.Sin(float64(i*256+j) * 0.01)
		}
		vecs[i] = v
	}
	enc := appendLossyVec(nil, vecs, false, toBudget(MinCosine(0.999)))
	got, isF32, used, err := readLossyVec(enc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if used != len(enc) {
		t.Fatalf("used %d != %d", used, len(enc))
	}
	if isF32 {
		t.Fatalf("elem type flipped")
	}
	if len(got) != len(vecs) || len(got[0]) != 256 {
		t.Fatalf("shape lost")
	}
	for i := range vecs {
		var dot, na, nb float64
		for j := range vecs[i] {
			dot += vecs[i][j] * got[i][j]
			na += vecs[i][j] * vecs[i][j]
			nb += got[i][j] * got[i][j]
		}
		if dot/(math.Sqrt(na)*math.Sqrt(nb)) < 0.999*0.999 {
			t.Fatalf("i=%d below cosine target", i)
		}
	}
}

func TestLossyVecSmallerThanRaw(t *testing.T) {
	vecs := make([][]float64, 128)
	for i := range vecs {
		v := make([]float64, 256)
		for j := range v {
			v[j] = float64((i + j) % 5) // low entropy
		}
		vecs[i] = v
	}
	enc := appendLossyVec(nil, vecs, true, toBudget(MaxRelError(0.02)))
	raw := 128 * 256 * 4 // f32 raw bytes
	if len(enc) >= raw {
		t.Fatalf("lossy %d not smaller than raw %d", len(enc), raw)
	}
}
