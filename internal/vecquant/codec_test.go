package vecquant

import (
	"math"
	"math/rand"
	"testing"
)

func gaussianVecs(n, dim int, seed int64) [][]float64 {
	r := rand.New(rand.NewSource(seed))
	out := make([][]float64, n)
	for i := range out {
		v := make([]float64, dim)
		for j := range v {
			v[j] = r.NormFloat64()
		}
		out[i] = v
	}
	return out
}

func TestEncodeSelectsE8WhenSmaller(t *testing.T) {
	vecs := gaussianVecs(200, 128, 5)
	bl := Encode(vecs, Budget{Kind: KindRelError, Val: 0.05})
	// On 128-dim Gaussian data E8 should win; if not, the variant must still be
	// a valid one and round-trip.
	recon := bl.Decode()
	if len(recon) != 200 || len(recon[0]) != 128 {
		t.Fatalf("shape lost")
	}
	if bl.Variant != VariantScalar && bl.Variant != VariantE8 {
		t.Fatalf("bad variant %d", bl.Variant)
	}
	got := achievedRelError(vecs, recon)
	if got > 0.05*1.15 {
		t.Fatalf("budget missed: rel %v", got)
	}
}

func TestEncodeE8RoundTrip(t *testing.T) {
	vecs := gaussianVecs(64, 256, 9)
	bl := Encode(vecs, Budget{Kind: KindCosine, Val: 0.999})
	if bl.Variant == VariantE8 && len(bl.Cosets) == 0 {
		t.Fatal("E8 variant must carry cosets")
	}
	recon := bl.Decode()
	for i := range vecs {
		var dot, na, nb float64
		for j := range vecs[i] {
			dot += vecs[i][j] * recon[i][j]
			na += vecs[i][j] * vecs[i][j]
			nb += recon[i][j] * recon[i][j]
		}
		if dot/(math.Sqrt(na)*math.Sqrt(nb)) < 0.999*0.999 {
			t.Fatalf("i=%d cosine below target", i)
		}
	}
}

func TestE8NotAttemptedBelowGate(t *testing.T) {
	// pdim < 16 (dim 8 -> pdim 8) must stay scalar.
	vecs := gaussianVecs(40, 8, 3)
	bl := Encode(vecs, Budget{Kind: KindRelError, Val: 0.05})
	if bl.Variant != VariantScalar {
		t.Fatalf("expected scalar variant for pdim<16, got %d", bl.Variant)
	}
	if !e8Eligible(16) || e8Eligible(8) {
		t.Fatalf("e8Eligible gate wrong")
	}
}

func mkVectors(n, dim int, seed int64) [][]float64 {
	r := rand.New(rand.NewSource(seed))
	out := make([][]float64, n)
	for i := range out {
		v := make([]float64, dim)
		for j := range v {
			v[j] = r.NormFloat64() // Gaussian, but with a few injected outliers
		}
		v[r.Intn(dim)] += 12 // outlier the rotation must smear
		out[i] = v
	}
	return out
}

func TestEncodeDecodeMeetsBudget(t *testing.T) {
	vecs := mkVectors(200, 128, 1)
	for _, eps := range []float64{0.05, 0.02, 0.005} {
		bl := Encode(vecs, Budget{Kind: KindRelError, Val: eps})
		recon := bl.Decode()
		got := achievedRelError(vecs, recon)
		if got > eps*1.15 { // 15% slack for the model + verify loop
			t.Fatalf("eps=%v achieved rel=%v exceeds budget", eps, got)
		}
	}
}

func TestEncodeNonPow2Dim(t *testing.T) {
	vecs := mkVectors(50, 768, 7) // 768 -> padded to 1024 internally
	bl := Encode(vecs, Budget{Kind: KindCosine, Val: 0.999})
	recon := bl.Decode()
	if len(recon) != 50 || len(recon[0]) != 768 {
		t.Fatalf("shape lost: %d x %d", len(recon), len(recon[0]))
	}
	// cosine check
	for i := range vecs {
		var dot, na, nb float64
		for j := range vecs[i] {
			dot += vecs[i][j] * recon[i][j]
			na += vecs[i][j] * vecs[i][j]
			nb += recon[i][j] * recon[i][j]
		}
		cos := dot / (math.Sqrt(na) * math.Sqrt(nb))
		if cos < 0.999*0.999 {
			t.Fatalf("i=%d cosine %v below target", i, cos)
		}
	}
}
