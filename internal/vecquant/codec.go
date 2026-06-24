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

// rotateInto pads each vector to pdim and rotates into dst (grown as needed),
// returning the flat buffer and the padded dim.
func rotateInto(vectors [][]float64, seed uint64, dst []float64) (flat []float64, pdim int) {
	dim := len(vectors[0])
	pdim = hadamard.NextPow2(dim)
	flat = growF64(dst, len(vectors)*pdim)
	for i, v := range vectors {
		row := flat[i*pdim : i*pdim+pdim]
		copy(row, v)
		for k := len(v); k < pdim; k++ {
			row[k] = 0 // clear pad (buffer may be reused/dirty)
		}
		hadamard.Forward(row, seed)
	}
	return flat, pdim
}

// rotateAll pads each vector to pdim, rotates in place, returns a flat
// [count*pdim] buffer plus the padded dim.
func rotateAll(vectors [][]float64, seed uint64) (flat []float64, pdim int) {
	return rotateInto(vectors, seed, nil)
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

// encodeScalar runs the scalar verify-loop and returns the encoded coords plus
// the (possibly grown) coord slice so the caller can reuse it next call.
func encodeScalar(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget, row []float64, q []int32) (variantResult, []int32) {
	delta := DeltaFor(b, sigma, lattice.ScalarG)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma
	}
	// Scalar is always emitted as the floor: tighten delta until it meets the
	// budget, then stop. Even at the tightest tried delta it is the fallback.
	for range 4 {
		q = lattice.QuantizeScalar(flat, delta, q)
		if budgetMetScalar(orig, q, pdim, dim, delta, b, row) {
			break
		}
		delta *= 0.6
	}
	return variantResult{delta: delta, coords: encodeCoords(q), ok: true}, q
}

// encodeE8 runs the E8 verify-loop; ok=false if it never meets the budget.
// row and q are reused buffers; q is passed through and returned unchanged
// (QuantizeE8 allocates coords internally; E8 coord reuse is a follow-up).
func encodeE8(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget, row []float64, q []int32) (variantResult, []int32) {
	delta := DeltaFor(b, sigma, lattice.E8G)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma
	}
	var coords []int32
	var cosets []byte
	ok := false
	for range 4 {
		coords, cosets = lattice.QuantizeE8(flat, delta)
		if budgetMetE8(orig, coords, cosets, pdim, dim, delta, b, row) {
			ok = true
			break
		}
		delta *= 0.6
	}
	if !ok {
		return variantResult{ok: false}, q
	}
	return variantResult{delta: delta, coords: encodeCoords(coords), cosets: cosets, ok: true}, q
}

// budgetMetStream reconstructs one vector at a time into row (len pdim), inverse-
// rotates it, and accumulates the budget metric against orig — without building
// a [][]float64. fill(i) writes the dequantized rotated coords of vector i into
// row[:pdim].
func budgetMetStream(orig [][]float64, pdim, dim int, b Budget, row []float64, fill func(i int)) bool {
	cosine := b.Kind == KindCosine
	worst := 0.0
	if cosine {
		worst = math.Inf(1)
	}
	for i := range orig {
		fill(i)
		hadamard.Inverse(row, hadamardSeed)
		o := orig[i]
		if cosine {
			var dot, na, nb float64
			for j := 0; j < dim; j++ {
				dot += o[j] * row[j]
				na += o[j] * o[j]
				nb += row[j] * row[j]
			}
			if cos := dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30); cos < worst {
				worst = cos
			}
		} else {
			var se, ne float64
			for j := 0; j < dim; j++ {
				d := o[j] - row[j]
				se += d * d
				ne += o[j] * o[j]
			}
			if rel := math.Sqrt(se / (ne + 1e-30)); rel > worst {
				worst = rel
			}
		}
	}
	if cosine {
		return worst >= b.Val
	}
	return worst <= relTarget(b)
}

func budgetMetScalar(orig [][]float64, q []int32, pdim, dim int, delta float64, b Budget, row []float64) bool {
	return budgetMetStream(orig, pdim, dim, b, row, func(i int) {
		seg := q[i*pdim : i*pdim+pdim]
		for j := 0; j < pdim; j++ {
			row[j] = float64(seg[j]) * delta
		}
	})
}

func budgetMetE8(orig [][]float64, coords []int32, cosets []byte, pdim, dim int, delta float64, b Budget, row []float64) bool {
	nbPerVec := pdim / 8
	return budgetMetStream(orig, pdim, dim, b, row, func(i int) {
		for blk := 0; blk < nbPerVec; blk++ {
			bb := i*nbPerVec + blk
			off := 0.0
			if cosets[bb/8]&(1<<(uint(bb)&7)) != 0 {
				off = 0.5
			}
			base := i*pdim + blk*8
			for k := 0; k < 8; k++ {
				row[blk*8+k] = (float64(coords[base+k]) + off) * delta
			}
		}
	})
}

// Encode is the convenience entry that allocates a one-shot Scratch.
func Encode(vectors [][]float64, b Budget) Block {
	var sc Scratch
	return EncodeWith(vectors, b, &sc)
}

// EncodeWith encodes reusing sc's buffers across the verify-loop and variants.
func EncodeWith(vectors [][]float64, b Budget, sc *Scratch) Block {
	if len(vectors) == 0 {
		return Block{Seed: hadamardSeed, Delta: 1, Variant: VariantScalar}
	}
	dim := len(vectors[0])
	flat, pdim := rotateInto(vectors, hadamardSeed, sc.flat)
	sc.flat = flat
	sc.row = growF64(sc.row, pdim)
	sigma := rms(flat)
	if sigma == 0 {
		sigma = 1
	}

	scRes, qs := encodeScalar(vectors, flat, pdim, dim, sigma, b, sc.row, sc.qScalar)
	sc.qScalar = qs
	best := Block{
		Dim: dim, Count: len(vectors), Seed: hadamardSeed,
		Delta: scRes.delta, Coords: scRes.coords, Variant: VariantScalar,
	}
	bestSize := len(scRes.coords)

	if e8Eligible(pdim) {
		e8, qe := encodeE8(vectors, flat, pdim, dim, sigma, b, sc.row, sc.qE8)
		sc.qE8 = qe
		// Compare true wire sizes: E8 additionally carries the coset stream and
		// its varuint length prefix, which the wire layer writes.
		e8Size := len(e8.coords) + uvarintLen(uint64(len(e8.cosets))) + len(e8.cosets)
		if e8.ok && e8Size < bestSize {
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
	row := make([]float64, pdim)
	e8, _ := encodeE8(vectors, flat, pdim, dim, sigma, b, row, nil)
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
	for i := range count {
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
