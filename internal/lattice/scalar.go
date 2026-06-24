// Package lattice implements vector quantizers used by the lossy vector codec.
// It currently provides the scalar (integer/cube) lattice. After the Hadamard
// rotation the data is ~Gaussian, so a uniform scalar grid with step Δ is
// already a reasonable quantizer and the entropy stage recovers the rest.
package lattice

import "math"

// ScalarG is the normalized second moment of the 1-D integer lattice (cube):
// the quantization MSE of a uniform scalar quantizer with step Δ is Δ²·ScalarG.
const ScalarG = 1.0 / 12.0

// QuantizeScalar rounds each x[i]/delta to the nearest integer, writing the
// results into dst and returning it. dst is reused when it has enough capacity
// (its existing contents are discarded), otherwise a new slice is allocated.
func QuantizeScalar(x []float64, delta float64, dst []int32) []int32 {
	if cap(dst) < len(x) {
		dst = make([]int32, 0, len(x))
	}
	dst = dst[:0]
	inv := 1.0 / delta
	for _, v := range x {
		dst = append(dst, int32(math.RoundToEven(v*inv)))
	}
	return dst
}

// ReconstructScalar maps each integer index back to float64(q[i])*delta.
func ReconstructScalar(q []int32, delta float64, dst []float64) []float64 {
	if cap(dst) < len(q) {
		dst = make([]float64, 0, len(q))
	}
	dst = dst[:0]
	for _, v := range q {
		dst = append(dst, float64(v)*delta)
	}
	return dst
}
