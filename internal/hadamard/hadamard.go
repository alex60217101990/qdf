// Package hadamard implements a randomized fast Walsh–Hadamard rotation
// R = (1/√n)·H·D, where D is a seed-driven random ±1 diagonal and H is the
// Walsh–Hadamard matrix. R is orthonormal, so its inverse is Rᵀ = D·(1/√n)·H.
// The rotation spreads per-coordinate outliers evenly, concentrating the data
// toward a Gaussian shape that low-bit scalar/lattice quantization handles well.
// Only a uint64 seed is stored on the wire — the matrix is never materialized.
package hadamard

import "math"

// NextPow2 returns the smallest power of two ≥ n (minimum 1).
func NextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// signs derives a deterministic ±1 per index from seed via splitmix64.
func sign(seed uint64, i int) float64 {
	z := seed + uint64(i+1)*0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	if z&1 == 0 {
		return 1
	}
	return -1
}

// fwht runs the in-place unnormalized Walsh–Hadamard butterfly. len(a) must be
// a power of two.
func fwht(a []float64) {
	n := len(a)
	for h := 1; h < n; h <<= 1 {
		for i := 0; i < n; i += h << 1 {
			for j := i; j < i+h; j++ {
				x, y := a[j], a[j+h]
				a[j], a[j+h] = x+y, x-y
			}
		}
	}
}

func checkPow2(n int) {
	if n == 0 || n&(n-1) != 0 {
		panic("hadamard: length must be a power of two")
	}
}

// Forward applies R·x = (1/√n)·H·(D·x) in place.
func Forward(x []float64, seed uint64) {
	n := len(x)
	checkPow2(n)
	for i := range x {
		x[i] *= sign(seed, i)
	}
	fwht(x)
	s := 1.0 / math.Sqrt(float64(n))
	for i := range x {
		x[i] *= s
	}
}

// Inverse applies R⁻¹·y = D·(1/√n)·H·y in place.
func Inverse(y []float64, seed uint64) {
	n := len(y)
	checkPow2(n)
	fwht(y)
	s := 1.0 / math.Sqrt(float64(n))
	for i := range y {
		y[i] *= s * sign(seed, i)
	}
}
