//go:build arm64

package hadamard

// fwhtNEON is implemented in hadamard_arm64.s.
// It runs the in-place unnormalized Walsh–Hadamard butterfly on len(a)
// elements. len(a) must be a power of two and ≥ 2.
//
//go:noescape
func fwhtNEON(a []float64)

// fwht dispatches to the NEON kernel on arm64.
func fwht(a []float64) {
	fwhtNEON(a)
}
