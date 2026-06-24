package lattice

import (
	"math"
	"testing"
)

func TestScalarRoundTripErrorBound(t *testing.T) {
	const delta = 0.25
	x := []float64{0, 0.1, -0.1, 1.0, -2.37, 5.5, -100.2}
	q := QuantizeScalar(x, delta, nil)
	got := ReconstructScalar(q, delta, nil)
	for i := range x {
		if math.Abs(got[i]-x[i]) > delta/2+1e-12 {
			t.Fatalf("i=%d |%v-%v|=%v exceeds delta/2", i, got[i], x[i], math.Abs(got[i]-x[i]))
		}
	}
}

func TestScalarMSEMatchesModel(t *testing.T) {
	// Uniform input over many steps: empirical MSE ≈ delta^2 * ScalarG.
	const delta = 1.0
	n := 100000
	x := make([]float64, n)
	for i := range x {
		x[i] = (float64(i%1000)/1000.0 - 0.5) * 50 // spans many cells
	}
	q := QuantizeScalar(x, delta, nil)
	got := ReconstructScalar(q, delta, nil)
	var se float64
	for i := range x {
		d := got[i] - x[i]
		se += d * d
	}
	mse := se / float64(n)
	want := delta * delta * ScalarG
	if math.Abs(mse-want)/want > 0.1 {
		t.Fatalf("mse=%v want≈%v (delta^2/12)", mse, want)
	}
}
