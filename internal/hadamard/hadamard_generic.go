//go:build !arm64

package hadamard

// fwht runs the in-place unnormalized Walsh–Hadamard butterfly.
// len(a) must be a power of two.
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
