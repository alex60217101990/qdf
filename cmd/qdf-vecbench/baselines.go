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

// ---- warm (buffer-reusing) encoders for the fair performance comparison ----
//
// These mirror the cold encoders above but reuse caller-provided scratch so the
// per-call allocation cost (not one-time setup) is what gets measured, on the
// same footing as qdf's reused Scratch.

// naiveScalarEncodeInto quantizes v into the reused q buffer.
func naiveScalarEncodeInto(v []float64, bits int, q []uint16) ([]uint16, float64, float64) {
	mn := math.Inf(1)
	mx := math.Inf(-1)
	for _, x := range v {
		mn = math.Min(mn, x)
		mx = math.Max(mx, x)
	}
	levels := float64(int(1) << bits)
	delta := (mx - mn) / (levels - 1)
	if delta == 0 {
		delta = 1
	}
	if cap(q) < len(v) {
		q = make([]uint16, len(v))
	}
	q = q[:len(v)]
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

// naiveBatchEncodeWarm encodes every vector in corpus reusing q and rowless
// state; returns the reused q and the total wire bytes. This is the timed unit.
func naiveBatchEncodeWarm(corpus [][]float64, bits int, q []uint16) ([]uint16, int) {
	total := 0
	for _, v := range corpus {
		var mn, delta float64
		q, mn, delta = naiveScalarEncodeInto(v, bits, q)
		_, _ = mn, delta
		total += naiveScalarBytes(q, bits)
	}
	return q, total
}

// tqBatchEncodeWarm rotates+quantizes every vector reusing row and q.
func tqBatchEncodeWarm(corpus [][]float64, bits int, seed uint64, row []float64, q []uint16) ([]float64, []uint16, int) {
	total := 0
	pdim := nextPow2(len(corpus[0]))
	if cap(row) < pdim {
		row = make([]float64, pdim)
	}
	row = row[:pdim]
	for _, v := range corpus {
		for i := range row {
			row[i] = 0
		}
		copy(row, v)
		hadamard.Forward(row, seed)
		var mn, delta float64
		q, mn, delta = naiveScalarEncodeInto(row, bits, q)
		_, _ = mn, delta
		total += turboQuantScalarBytes(q, bits)
	}
	return row, q, total
}

// ---- decode-only timing helpers ----
//
// The decode comparison must time decode in isolation: the encode step is run
// once up front (cold) and each vector keeps its own code slice, so the timed
// closure does pure decode work — matching qdf's bl.Decode(), which decodes
// without re-encoding. Running encode inside the timed loop (as a naive batch
// helper would) measures encode+decode and understates decode throughput.

// naiveEncoded retains one vector's quantized form for the decode-only timing.
type naiveEncoded struct {
	q     []uint16
	mn    float64
	delta float64
}

// naiveBatchEncodeAll encodes every vector into its own retained code slice
// (cold; runs once, outside the timed decode loop).
func naiveBatchEncodeAll(corpus [][]float64, bits int) []naiveEncoded {
	enc := make([]naiveEncoded, len(corpus))
	for i, v := range corpus {
		q, mn, delta := naiveScalarEncode(v, bits)
		enc[i] = naiveEncoded{q: q, mn: mn, delta: delta}
	}
	return enc
}

// naiveBatchDecodeWarm decodes every pre-encoded vector into recon; this is the
// timed decode unit (no encode work inside).
func naiveBatchDecodeWarm(enc []naiveEncoded, recon [][]float64) {
	for i := range enc {
		recon[i] = naiveScalarDecode(enc[i].q, enc[i].mn, enc[i].delta)
	}
}

// tqEncoded is the TurboQuant-scalar analogue of naiveEncoded.
type tqEncoded struct {
	q     []uint16
	mn    float64
	delta float64
	pdim  int
}

// tqBatchEncodeAll rotates+quantizes every vector into its own retained code
// slice (cold; runs once, outside the timed decode loop).
func tqBatchEncodeAll(corpus [][]float64, bits int, seed uint64) []tqEncoded {
	enc := make([]tqEncoded, len(corpus))
	for i, v := range corpus {
		q, mn, delta, pdim := turboQuantScalarEncode(v, bits, seed)
		enc[i] = tqEncoded{q: q, mn: mn, delta: delta, pdim: pdim}
	}
	return enc
}

// tqBatchDecodeWarm decodes (dequant + inverse-rotate) every pre-encoded vector
// into recon; this is the timed decode unit (no encode work inside).
func tqBatchDecodeWarm(enc []tqEncoded, origDim int, seed uint64, recon [][]float64) {
	for i := range enc {
		recon[i] = turboQuantScalarDecode(enc[i].q, enc[i].mn, enc[i].delta, enc[i].pdim, origDim, seed)
	}
}
