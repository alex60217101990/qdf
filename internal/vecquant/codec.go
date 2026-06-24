package vecquant

import (
	"math"

	"github.com/alex60217101990/qdf/internal/hadamard"
	"github.com/alex60217101990/qdf/internal/lattice"
)

// hadamardSeed is the fixed rotation seed (stored on the wire so a future
// encoder may randomize it per column without breaking decode).
const hadamardSeed uint64 = 0x51ed270b9f4d8c3a

// Variant identifies the lattice used for a Block's coordinates.
const (
	VariantScalar uint8 = 0
	VariantE8     uint8 = 1
)

// Block is the self-contained encoded form of a set of equal-length vectors.
type Block struct {
	Dim     int
	Count   int
	Seed    uint64
	Delta   float64
	Coords  []byte // encodeCoords output over the flattened, row-major indices
	Variant uint8  // VariantScalar or VariantE8
	Cosets  []byte // coset bits for VariantE8; nil for VariantScalar
}

// e8Eligible reports whether the E8 variant is worth attempting for a padded
// dimension. Below 16 (one or two 8-D blocks) the coset/packing overhead
// dominates any packing gain, so we skip the extra encode. This is a perf gate,
// never a correctness gate: scalar is always computed.
func e8Eligible(pdim int) bool { return pdim >= 16 }

// rotateAll pads each vector to pdim, rotates in place, returns a flat
// [count*pdim] buffer plus the padded dim.
func rotateAll(vectors [][]float64, seed uint64) (flat []float64, pdim int) {
	dim := len(vectors[0])
	pdim = hadamard.NextPow2(dim)
	flat = make([]float64, len(vectors)*pdim)
	for i, v := range vectors {
		row := flat[i*pdim : i*pdim+pdim]
		copy(row, v) // tail stays zero (pad)
		hadamard.Forward(row, seed)
	}
	return flat, pdim
}

func rms(flat []float64) float64 {
	var s float64
	for _, v := range flat {
		s += v * v
	}
	if len(flat) == 0 {
		return 1
	}
	return math.Sqrt(s / float64(len(flat)))
}

// variantResult holds the output of one encode attempt.
type variantResult struct {
	delta  float64
	coords []byte
	cosets []byte
	ok     bool
}

// encodeScalar runs the scalar verify-loop and returns the encoded coords.
// Scalar is always emitted as the floor; ok is not checked by the caller.
func encodeScalar(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget) variantResult {
	delta := DeltaFor(b, sigma, lattice.ScalarG)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma
	}
	var q []int32
	// Scalar is always emitted as the floor: tighten delta until it meets the
	// budget, then stop. Even at the tightest tried delta it is the fallback.
	for range 4 {
		q = lattice.QuantizeScalar(flat, delta, q)
		if budgetMetScalar(orig, q, pdim, dim, delta, b) {
			break
		}
		delta *= 0.6
	}
	return variantResult{delta: delta, coords: encodeCoords(q), ok: true}
}

// encodeE8 runs the E8 verify-loop; ok=false if it never meets the budget.
func encodeE8(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget) variantResult {
	delta := DeltaFor(b, sigma, lattice.E8G)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma
	}
	var coords []int32
	var cosets []byte
	ok := false
	for range 4 {
		coords, cosets = lattice.QuantizeE8(flat, delta)
		if budgetMetE8(orig, coords, cosets, pdim, dim, delta, b) {
			ok = true
			break
		}
		delta *= 0.6
	}
	if !ok {
		return variantResult{ok: false}
	}
	return variantResult{delta: delta, coords: encodeCoords(coords), cosets: cosets, ok: true}
}

func budgetMetScalar(orig [][]float64, q []int32, pdim, dim int, delta float64, b Budget) bool {
	recon := reconstruct(q, pdim, dim, len(orig), delta)
	return budgetCheck(orig, recon, b)
}

func budgetMetE8(orig [][]float64, coords []int32, cosets []byte, pdim, dim int, delta float64, b Budget) bool {
	recon := reconstructE8(coords, cosets, pdim, dim, len(orig), delta)
	return budgetCheck(orig, recon, b)
}

func budgetCheck(orig, recon [][]float64, b Budget) bool {
	got := metric(orig, recon, b.Kind)
	if b.Kind == KindCosine {
		return got >= b.Val
	}
	return got <= relTarget(b)
}

// Encode rotates, then encodes with the scalar quantizer and (when eligible)
// the E8 quantizer, keeping the smaller encoding that meets the budget.
func Encode(vectors [][]float64, b Budget) Block {
	if len(vectors) == 0 {
		return Block{Seed: hadamardSeed, Delta: 1, Variant: VariantScalar}
	}
	dim := len(vectors[0])
	flat, pdim := rotateAll(vectors, hadamardSeed)
	sigma := rms(flat)
	if sigma == 0 {
		sigma = 1
	}

	sc := encodeScalar(vectors, flat, pdim, dim, sigma, b)
	best := Block{
		Dim: dim, Count: len(vectors), Seed: hadamardSeed,
		Delta: sc.delta, Coords: sc.coords, Variant: VariantScalar,
	}
	bestSize := len(sc.coords)

	if e8Eligible(pdim) {
		e8 := encodeE8(vectors, flat, pdim, dim, sigma, b)
		if e8.ok && len(e8.coords)+len(e8.cosets) < bestSize {
			best = Block{
				Dim: dim, Count: len(vectors), Seed: hadamardSeed,
				Delta: e8.delta, Coords: e8.coords, Variant: VariantE8, Cosets: e8.cosets,
			}
		}
	}
	return best
}

// EncodeForcedE8 encodes with the E8 quantizer only, ignoring the scalar
// alternative. It returns the resulting Block and ok=true when E8 is eligible
// and meets the budget; otherwise ok=false. Intended for measurement/benchmark
// callers that want the real E8 wire size (rANS-coded coords + coset stream),
// not for the production encode path, which uses the never-worse try-both Encode.
func EncodeForcedE8(vectors [][]float64, b Budget) (Block, bool) {
	if len(vectors) == 0 {
		return Block{Seed: hadamardSeed, Delta: 1, Variant: VariantE8}, false
	}
	dim := len(vectors[0])
	flat, pdim := rotateAll(vectors, hadamardSeed)
	if !e8Eligible(pdim) {
		return Block{}, false
	}
	sigma := rms(flat)
	if sigma == 0 {
		sigma = 1
	}
	e8 := encodeE8(vectors, flat, pdim, dim, sigma, b)
	if !e8.ok {
		return Block{}, false
	}
	return Block{
		Dim: dim, Count: len(vectors), Seed: hadamardSeed,
		Delta: e8.delta, Coords: e8.coords, Variant: VariantE8, Cosets: e8.cosets,
	}, true
}

// relTarget converts any budget to an equivalent max-rel-error for comparison.
func relTarget(b Budget) float64 {
	switch b.Kind {
	case KindRelError:
		return b.Val
	case KindSNR:
		return math.Pow(10, -b.Val/20)
	case KindCosine:
		return math.Sqrt(2 * (1 - b.Val))
	}
	return b.Val
}

// metric returns rel-error (RelError/SNR) or min-cosine (Cosine) over all rows.
func metric(orig, recon [][]float64, kind BudgetKind) float64 {
	if kind == KindCosine {
		worst := math.Inf(1)
		for i := range orig {
			var dot, na, nb float64
			for j := range orig[i] {
				dot += orig[i][j] * recon[i][j]
				na += orig[i][j] * orig[i][j]
				nb += recon[i][j] * recon[i][j]
			}
			cos := dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30)
			if cos < worst {
				worst = cos
			}
		}
		return worst
	}
	return achievedRelError(orig, recon)
}

func achievedRelError(orig, recon [][]float64) float64 {
	worst := 0.0
	for i := range orig {
		var se, ne float64
		for j := range orig[i] {
			d := orig[i][j] - recon[i][j]
			se += d * d
			ne += orig[i][j] * orig[i][j]
		}
		rel := math.Sqrt(se / (ne + 1e-30))
		if rel > worst {
			worst = rel
		}
	}
	return worst
}

// reconstruct dequantizes q and inverse-rotates back to original dim.
func reconstruct(q []int32, pdim, dim, count int, delta float64) [][]float64 {
	out := make([][]float64, count)
	row := make([]float64, pdim)
	for i := range count {
		seg := q[i*pdim : i*pdim+pdim]
		for j := range pdim {
			row[j] = float64(seg[j]) * delta
		}
		hadamard.Inverse(row, hadamardSeed)
		v := make([]float64, dim)
		copy(v, row[:dim])
		out[i] = v
	}
	return out
}

// reconstructE8 dequantizes E8 coords+cosets and inverse-rotates to dim.
func reconstructE8(coords []int32, cosets []byte, pdim, dim, count int, delta float64) [][]float64 {
	flat := lattice.ReconstructE8(coords, cosets, delta, count*pdim)
	out := make([][]float64, count)
	row := make([]float64, pdim)
	for i := 0; i < count; i++ {
		copy(row, flat[i*pdim:i*pdim+pdim])
		hadamard.Inverse(row, hadamardSeed)
		v := make([]float64, dim)
		copy(v, row[:dim])
		out[i] = v
	}
	return out
}

// Decode inverts the pipeline from the on-wire block.
func (bl Block) Decode() [][]float64 {
	if bl.Count == 0 {
		return nil
	}
	pdim := hadamard.NextPow2(bl.Dim)
	q, err := decodeCoords(bl.Coords, bl.Count*pdim)
	if err != nil {
		return nil
	}
	if bl.Variant == VariantE8 {
		// Defense in depth: the coset stream holds one bit per 8-D block, so
		// blocks = count*pdim/8 and the byte length is ceil(blocks/8). The wire
		// layer validates this too, but guard here so a malformed Block can
		// never index past the coset slice.
		blocks := bl.Count * pdim / 8
		if len(bl.Cosets) != (blocks+7)/8 {
			return nil
		}
		return reconstructE8(q, bl.Cosets, pdim, bl.Dim, bl.Count, bl.Delta)
	}
	return reconstruct(q, pdim, bl.Dim, bl.Count, bl.Delta)
}
