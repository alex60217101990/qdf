package lattice

import (
	"math"
	"testing"
)

// isE8Point reports whether p is a valid E8 lattice point: all-integer with even
// sum, or all-(integer+0.5) with even sum of the integer parts.
func isE8Point(p [8]float64) bool {
	allInt := true
	allHalf := true
	var isum, hsum int
	for _, v := range p {
		fl := math.Floor(v)
		frac := v - fl
		if frac != 0 {
			allInt = false
		}
		if frac != 0.5 {
			allHalf = false
		}
		isum += int(fl)
		hsum += int(fl)
	}
	if allInt {
		return isum%2 == 0
	}
	if allHalf {
		return hsum%2 == 0
	}
	return false
}

func TestNearestE8ReturnsValidPoint(t *testing.T) {
	xs := [][8]float64{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0.4, -0.6, 1.2, 2.7, -3.1, 0.1, 0.9, -0.2},
		{1.5, 1.5, 1.5, 1.5, 0, 0, 0, 0},
		{10.3, -7.8, 4.4, 4.4, -1.1, 2.2, 3.3, 5.6},
	}
	for _, x := range xs {
		p := NearestE8(&x)
		if !isE8Point(p) {
			t.Fatalf("NearestE8(%v) = %v is not a valid E8 point", x, p)
		}
	}
}

func TestNearestE8FixesLatticePoint(t *testing.T) {
	// An exact lattice point maps to itself.
	pts := [][8]float64{
		{1, 1, 0, 0, 0, 0, 0, 0},                  // integer, even sum
		{2, -2, 0, 0, 0, 0, 0, 0},                 // integer, even sum
		{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},  // half, even int-sum (0)
		{1.5, 0.5, -0.5, 0.5, 0.5, 0.5, 0.5, 0.5}, // half coset
	}
	for _, p := range pts {
		if !isE8Point(p) {
			t.Fatalf("test vector %v is not E8 — fix the test", p)
		}
		got := NearestE8(&p)
		for i := range p {
			if got[i] != p[i] {
				t.Fatalf("NearestE8(lattice point %v) moved it to %v", p, got)
			}
		}
	}
}

func TestQuantizeReconstructE8RoundTripErrorBound(t *testing.T) {
	const delta = 0.5
	x := make([]float64, 64)
	for i := range x {
		x[i] = math.Sin(float64(i)*0.37) * 4
	}
	coords, cosets := QuantizeE8(x, delta, nil)
	got := ReconstructE8(coords, cosets, delta, len(x))
	// E8 covering radius is sqrt(2); per-coordinate error is bounded loosely by
	// delta*sqrt(2). Assert a comfortable bound that still catches gross errors.
	for i := range x {
		if math.Abs(got[i]-x[i]) > delta*1.5 {
			t.Fatalf("i=%d |%v-%v| exceeds bound", i, got[i], x[i])
		}
	}
}

func TestE8MSEBeatsScalar(t *testing.T) {
	// On Gaussian-ish data at the same step, E8 MSE < scalar (cube) MSE because
	// G(E8) < G(Z). Use a fixed step and compare empirical MSE.
	const delta = 1.0
	n := 8000
	x := make([]float64, n)
	for i := range x {
		x[i] = math.Sin(float64(i)*0.911) + math.Cos(float64(i)*0.123)
	}
	// scalar
	qs := QuantizeScalar(x, delta, nil)
	rs := ReconstructScalar(qs, delta, nil)
	var seScalar float64
	for i := range x {
		d := rs[i] - x[i]
		seScalar += d * d
	}
	// e8
	ce, cs := QuantizeE8(x, delta, nil)
	re := ReconstructE8(ce, cs, delta, n)
	var seE8 float64
	for i := range x {
		d := re[i] - x[i]
		seE8 += d * d
	}
	if seE8 >= seScalar {
		t.Fatalf("E8 MSE %v not < scalar MSE %v", seE8/float64(n), seScalar/float64(n))
	}
}
