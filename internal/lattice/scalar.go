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
//
// The second return reports whether any coordinate saturated the int32 range:
// a magnitude past 2^31 (a too-tight budget on a spiky vector shrinking delta)
// cannot be represented, and a raw cast would wrap (sign-flip) and silently
// corrupt the vector. Such values are clamped (saturated) and overflow=true is
// reported so the caller can abort to a lossless encoding instead.
func QuantizeScalar(x []float64, delta float64, dst []int32) (out []int32, overflow bool) {
	if cap(dst) < len(x) {
		dst = make([]int32, 0, len(x))
	}
	dst = dst[:0]
	inv := 1.0 / delta
	for _, v := range x {
		r := math.RoundToEven(v * inv)
		if r > math.MaxInt32 {
			r, overflow = math.MaxInt32, true
		} else if r < math.MinInt32 {
			r, overflow = math.MinInt32, true
		}
		dst = append(dst, int32(r))
	}
	return dst, overflow
}

// clampI32 saturates a rounded lattice coordinate to the int32 range, reporting
// whether it was out of range (see QuantizeScalar).
func clampI32(f float64) (int32, bool) {
	if f > math.MaxInt32 {
		return math.MaxInt32, true
	}
	if f < math.MinInt32 {
		return math.MinInt32, true
	}
	return int32(f), false
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
