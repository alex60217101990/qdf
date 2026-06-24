// E8 lattice quantizer. E8 is the densest lattice packing in 8 dimensions,
// the union of D8 (integer coords, even sum) and its glue coset D8+½
// (half-integer coords). Its normalized second moment E8G is below the cubic
// lattice's 1/12, so at the same step it quantizes with lower MSE. After the
// Hadamard rotation the data is ~Gaussian, the regime where this packing gain
// is realized.
package lattice

import "math"

// E8G is the normalized second moment of the E8 lattice (vs 1/12 for the cube).
const E8G = 0.071682

// nearestD8 returns the closest point of D8 (integer coordinates with even sum)
// to x. Round each coordinate to the nearest integer; if the sum is odd, move
// the single coordinate with the largest rounding error to its second-nearest
// integer (the exact D8 decoder).
func nearestD8(x *[8]float64) [8]float64 {
	var pt [8]float64
	var sum int
	worstIdx, worstErr := 0, -1.0
	for i := range 8 {
		r := math.RoundToEven(x[i])
		pt[i] = r
		sum += int(r)
		if e := math.Abs(x[i] - r); e > worstErr {
			worstErr, worstIdx = e, i
		}
	}
	if sum&1 != 0 {
		if x[worstIdx] >= pt[worstIdx] {
			pt[worstIdx]++
		} else {
			pt[worstIdx]--
		}
	}
	return pt
}

func dist2(a, b *[8]float64) float64 {
	var s float64
	for i := range 8 {
		d := a[i] - b[i]
		s += d * d
	}
	return s
}

// NearestE8 returns the exact nearest E8 lattice point to x: the closer of the
// nearest D8 point and the nearest D8+½ point.
func NearestE8(x *[8]float64) [8]float64 {
	p0 := nearestD8(x)
	var xs [8]float64
	for i := range 8 {
		xs[i] = x[i] - 0.5
	}
	p1d := nearestD8(&xs)
	var p1 [8]float64
	for i := range 8 {
		p1[i] = p1d[i] + 0.5
	}
	if dist2(x, &p1) < dist2(x, &p0) {
		return p1
	}
	return p0
}

// QuantizeE8 quantizes x (length a multiple of 8) to the E8 lattice at the
// given step. Returns the integer coordinates (floor for the half-integer
// coset) and one coset bit per 8-D block, LSB-first.
func QuantizeE8(x []float64, delta float64) (coords []int32, cosets []byte) {
	if len(x)%8 != 0 {
		panic("lattice: QuantizeE8 length must be a multiple of 8")
	}
	nb := len(x) / 8
	coords = make([]int32, 0, len(x))
	cosets = make([]byte, (nb+7)/8)
	inv := 1.0 / delta
	var blk [8]float64
	for b := range nb {
		for i := range 8 {
			blk[i] = x[b*8+i] * inv
		}
		pt := NearestE8(&blk)
		if pt[0]-math.Floor(pt[0]) == 0.5 { // half-integer coset
			cosets[b/8] |= 1 << (uint(b) & 7)
			for i := range 8 {
				coords = append(coords, int32(math.Floor(pt[i])))
			}
		} else {
			for i := range 8 {
				coords = append(coords, int32(pt[i]))
			}
		}
	}
	return coords, cosets
}

// ReconstructE8 inverts QuantizeE8. n must be a multiple of 8 and len(coords)==n.
func ReconstructE8(coords []int32, cosets []byte, delta float64, n int) []float64 {
	out := make([]float64, n)
	nb := n / 8
	for b := range nb {
		off := 0.0
		if cosets[b/8]&(1<<(uint(b)&7)) != 0 {
			off = 0.5
		}
		for i := range 8 {
			out[b*8+i] = (float64(coords[b*8+i]) + off) * delta
		}
	}
	return out
}
