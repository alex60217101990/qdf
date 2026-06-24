package main

import (
	"math"
	"math/rand"

	"github.com/alex60217101990/qdf/internal/hadamard"
)

// naiveScalarEncode: per-vector min-max uniform scalar quantization at `bits`
// bits per coordinate. Returns quantized codes plus the range parameters
// needed for decode.
func naiveScalarEncode(v []float64, bits int) (q []uint16, mn, delta float64) {
	mn = math.Inf(1)
	mx := math.Inf(-1)
	for _, x := range v {
		mn = math.Min(mn, x)
		mx = math.Max(mx, x)
	}
	levels := float64(int(1) << bits)
	delta = (mx - mn) / (levels - 1)
	if delta == 0 {
		delta = 1
	}
	q = make([]uint16, len(v))
	for i, x := range v {
		c := math.RoundToEven((x - mn) / delta)
		if c < 0 {
			c = 0
		}
		if c > levels-1 {
			c = levels - 1
		}
		q[i] = uint16(c)
	}
	return q, mn, delta
}

func naiveScalarDecode(q []uint16, mn, delta float64) []float64 {
	out := make([]float64, len(q))
	for i, c := range q {
		out[i] = mn + float64(c)*delta
	}
	return out
}

// naiveScalarBytes returns the wire size for an encoded vector: two float64
// range parameters (mn, delta) plus the codes bit-packed at the given
// bit-width. Bit-packing is the conventional representation; storing as
// uint16 would be wasteful and not representative.
func naiveScalarBytes(q []uint16, bits int) int {
	codeBits := len(q) * bits
	codeBytes := (codeBits + 7) / 8
	return 8 + 8 + codeBytes // mn(8) + delta(8) + packed codes
}

// turboQuantScalarEncode applies a randomised Hadamard rotation then encodes
// with naiveScalarEncode. This mirrors TurboQuant's "rotate → scalar" path
// without the entropy stage.
func turboQuantScalarEncode(v []float64, bits int, seed uint64) (q []uint16, mn, delta float64, pdim int) {
	pdim = hadamard.NextPow2(len(v))
	row := make([]float64, pdim)
	copy(row, v)
	hadamard.Forward(row, seed)
	q, mn, delta = naiveScalarEncode(row, bits)
	return q, mn, delta, pdim
}

func turboQuantScalarDecode(q []uint16, mn, delta float64, pdim, origDim int, seed uint64) []float64 {
	row := naiveScalarDecode(q, mn, delta)
	hadamard.Inverse(row, seed)
	return row[:origDim]
}

// turboQuantScalarBytes returns the byte cost for the TurboQuant-scalar codec:
// seed(8) + mn(8) + delta(8) + codes bit-packed at the given bit-width.
// The TurboQuant scheme pads the vector to the next power of two before
// rotation, so the code count is pdim (which equals len(q)).
func turboQuantScalarBytes(q []uint16, bits int) int {
	codeBits := len(q) * bits
	codeBytes := (codeBits + 7) / 8
	return 8 + 8 + 8 + codeBytes // seed(8) + mn(8) + delta(8) + packed codes
}

// pqAvgRelError trains product-quantization codebooks (k-means per subspace)
// and returns the mean per-vector relative L2 error over the full dataset.
func pqAvgRelError(data [][]float64, subspaces, bits int) float64 {
	codebooks := trainPQ(data, subspaces, bits)
	var sum float64
	for _, v := range data {
		recon := pqReconstruct(v, codebooks, subspaces)
		var se, ne float64
		for j := range v {
			d := v[j] - recon[j]
			se += d * d
			ne += v[j] * v[j]
		}
		sum += math.Sqrt(se / (ne + 1e-30))
	}
	return sum / float64(len(data))
}

// pqBytesPerVector returns the wire cost for PQ given the codebook layout.
// Each subspace requires ceil(bits) bits; we round up to a whole number of
// bytes per vector. Codebook tables are amortised over the corpus and not
// included in per-vector cost (standard ANN convention).
func pqBytesPerVector(subspaces, bits int) float64 {
	bitsTotal := subspaces * bits
	return math.Ceil(float64(bitsTotal) / 8.0)
}

// pqCodebook holds the centroids for one PQ subspace.
type pqCodebook struct {
	centroids [][]float64 // K × subDim
	start     int         // coordinate offset in the full vector
	subDim    int
}

func trainPQ(data [][]float64, subspaces, bits int) []pqCodebook {
	dim := len(data[0])
	subDim := dim / subspaces
	K := 1 << bits
	r := rand.New(rand.NewSource(11))
	books := make([]pqCodebook, subspaces)
	for s := 0; s < subspaces; s++ {
		start := s * subDim
		cents := make([][]float64, K)
		for k := 0; k < K; k++ { // init from random rows
			row := data[r.Intn(len(data))]
			c := make([]float64, subDim)
			copy(c, row[start:start+subDim])
			cents[k] = c
		}
		// Lloyd iterations
		for it := 0; it < 8; it++ {
			sums := make([][]float64, K)
			cnts := make([]int, K)
			for k := range sums {
				sums[k] = make([]float64, subDim)
			}
			for _, v := range data {
				best, bd := 0, math.Inf(1)
				for k := 0; k < K; k++ {
					d := sqDist(v[start:start+subDim], cents[k])
					if d < bd {
						bd, best = d, k
					}
				}
				for j := 0; j < subDim; j++ {
					sums[best][j] += v[start+j]
				}
				cnts[best]++
			}
			for k := 0; k < K; k++ {
				if cnts[k] == 0 {
					continue
				}
				for j := 0; j < subDim; j++ {
					cents[k][j] = sums[k][j] / float64(cnts[k])
				}
			}
		}
		books[s] = pqCodebook{centroids: cents, start: start, subDim: subDim}
	}
	return books
}

func pqReconstruct(v []float64, books []pqCodebook, _ int) []float64 {
	out := make([]float64, len(v))
	for _, b := range books {
		best, bd := 0, math.Inf(1)
		for k, c := range b.centroids {
			d := sqDist(v[b.start:b.start+b.subDim], c)
			if d < bd {
				bd, best = d, k
			}
		}
		copy(out[b.start:b.start+b.subDim], b.centroids[best])
	}
	return out
}

func sqDist(a, b []float64) float64 {
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return s
}
