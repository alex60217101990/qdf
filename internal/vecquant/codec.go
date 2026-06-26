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
	// Slices first so the GC pointer-scan range stops here instead of spanning
	// the whole struct (the trailing scalars are pointer-free).
	Coords  []byte // encodeCoords output over the flattened, row-major indices
	Cosets  []byte // coset bits for VariantE8; nil for VariantScalar
	Dim     int
	Count   int
	Seed    uint64
	Delta   float64
	Variant uint8 // VariantScalar or VariantE8
}

// e8Eligible reports whether the E8 variant is worth attempting for a padded
// dimension. Below 16 (one or two 8-D blocks) the coset/packing overhead
// dominates any packing gain, so we skip the extra encode. This is a perf gate,
// never a correctness gate: scalar is always computed.
func e8Eligible(pdim int) bool { return pdim >= 16 }

// e8TryThreshold is the loosest rel-error at which the E8 variant can still beat
// scalar; above it the coset overhead makes E8 lose, so skip the second encode.
// Set above the measured win boundary (rel ~0.02-0.03) so a winning E8 is never
// skipped.
const e8TryThreshold = 0.04

// e8WorthTrying reports whether the budget is tight enough for E8 to have a
// chance of beating scalar. A perf gate (skips wasted work), never a correctness
// gate: scalar is always computed and the never-worse picker still applies.
func e8WorthTrying(b Budget) bool { return relTarget(b) <= e8TryThreshold }

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

// encodeScalar runs the scalar verify-loop and returns the chosen delta plus
// the (possibly grown) quantized coord slice so the caller can reuse it.
func encodeScalar(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget, row []float64, q []int32) (float64, []int32) {
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
	return delta, q
}

// encodeE8 runs the E8 verify-loop; ok=false if it never meets the budget.
// Returns the chosen delta, the integer coords, and the coset bits. qDst is
// reused as the coords backing across the verify-loop iterations (pass nil for
// a fresh slice each call).
func encodeE8(orig [][]float64, flat []float64, pdim, dim int, sigma float64, b Budget, row []float64, qDst []int32) (delta float64, coords []int32, cosets []byte, ok bool) {
	delta = DeltaFor(b, sigma, lattice.E8G)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma
	}
	coords = qDst
	for range 4 {
		coords, cosets = lattice.QuantizeE8(flat, delta, coords)
		if budgetMetE8(orig, coords, cosets, pdim, dim, delta, b, row) {
			ok = true
			break
		}
		delta *= 0.6
	}
	return delta, coords, cosets, ok
}

// budgetMetStream reconstructs one vector at a time into row (len pdim), inverse-
// rotates it, and accumulates the budget metric against orig — without building
// a [][]float64. fill(i) writes the dequantized rotated coords of vector i into
// row[:pdim].
func budgetMetStream(orig [][]float64, dim int, b Budget, row []float64, fill func(i int)) bool {
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
			for j := range dim {
				dot += o[j] * row[j]
				na += o[j] * o[j]
				nb += row[j] * row[j]
			}
			if cos := dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30); cos < worst {
				worst = cos
			}
		} else {
			var se, ne float64
			for j := range dim {
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
	return budgetMetStream(orig, dim, b, row, func(i int) {
		seg := q[i*pdim : i*pdim+pdim]
		for j := range pdim {
			row[j] = float64(seg[j]) * delta
		}
	})
}

func budgetMetE8(orig [][]float64, coords []int32, cosets []byte, pdim, dim int, delta float64, b Budget, row []float64) bool {
	nbPerVec := pdim / 8
	return budgetMetStream(orig, dim, b, row, func(i int) {
		for blk := range nbPerVec {
			bb := i*nbPerVec + blk
			off := 0.0
			if cosets[bb/8]&(1<<(uint(bb)&7)) != 0 {
				off = 0.5
			}
			base := i*pdim + blk*8
			for k := range 8 {
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
//
// The returned Block.Coords aliases an sc-owned buffer; it stays valid only
// until the next EncodeWith call on the same sc. Callers must consume it (the
// qdf wire layer copies it into the output before reusing the encoder) before
// re-encoding. Use Encode for a one-shot Block whose Coords is independent.
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

	scDelta, qs := encodeScalar(vectors, flat, pdim, dim, sigma, b, sc.row, sc.qScalar)
	sc.qScalar = qs
	scCoords, zb, rb := encodeCoordsInto(qs, sc.coordsScalar, sc.zig, sc.ransDst)
	sc.coordsScalar, sc.zig, sc.ransDst = scCoords, zb, rb
	best := Block{
		Dim: dim, Count: len(vectors), Seed: hadamardSeed,
		Delta: scDelta, Coords: scCoords, Variant: VariantScalar,
	}
	bestSize := len(scCoords)

	if e8Eligible(pdim) && e8WorthTrying(b) {
		e8Delta, e8Coords, e8Cosets, ok := encodeE8(vectors, flat, pdim, dim, sigma, b, sc.row, sc.qE8)
		sc.qE8 = e8Coords
		if ok {
			// E8's coord block uses its own scratch buffer so it can coexist with
			// scalar's for the size comparison; the zig/ransDst staging is reused.
			e8Bytes, zb2, rb2 := encodeCoordsInto(e8Coords, sc.coordsE8, sc.zig, sc.ransDst)
			sc.coordsE8, sc.zig, sc.ransDst = e8Bytes, zb2, rb2
			// Compare true wire sizes: E8 additionally carries the coset stream and
			// its varuint length prefix, which the wire layer writes.
			e8Size := len(e8Bytes) + uvarintLen(uint64(len(e8Cosets))) + len(e8Cosets)
			if e8Size < bestSize {
				best = Block{
					Dim: dim, Count: len(vectors), Seed: hadamardSeed,
					Delta: e8Delta, Coords: e8Bytes, Variant: VariantE8, Cosets: e8Cosets,
				}
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
	e8Delta, e8Coords, e8Cosets, ok := encodeE8(vectors, flat, pdim, dim, sigma, b, row, nil)
	if !ok {
		return Block{}, false
	}
	return Block{
		Dim: dim, Count: len(vectors), Seed: hadamardSeed,
		Delta: e8Delta, Coords: encodeCoords(e8Coords), Variant: VariantE8, Cosets: e8Cosets,
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
