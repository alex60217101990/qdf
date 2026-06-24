package vecquant

import (
	"math"

	"github.com/alex60217101990/qdf/internal/hadamard"
	"github.com/alex60217101990/qdf/internal/lattice"
)

// hadamardSeed is the fixed rotation seed for Phase 1 (stored on the wire so a
// future encoder may randomize it per column without breaking decode).
const hadamardSeed uint64 = 0x51ed270b9f4d8c3a

// Block is the self-contained encoded form of a set of equal-length vectors.
type Block struct {
	Dim    int
	Count  int
	Seed   uint64
	Delta  float64
	Coords []byte // encodeCoords output over the flattened, row-major indices
}

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

// Encode runs rotate → choose Δ → quantize, shrinking Δ until the sampled
// achieved error meets the budget (bounded retries).
func Encode(vectors [][]float64, b Budget) Block {
	if len(vectors) == 0 {
		return Block{Seed: hadamardSeed, Delta: 1}
	}
	dim := len(vectors[0])
	flat, pdim := rotateAll(vectors, hadamardSeed)
	sigma := rms(flat)
	if sigma == 0 {
		sigma = 1
	}
	delta := DeltaFor(b, sigma, lattice.ScalarG)
	if delta <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		delta = sigma // safe fallback
	}

	var q []int32
	for tries := 0; tries < 4; tries++ {
		q = lattice.QuantizeScalar(flat, delta, q)
		if budgetMet(vectors, flat, q, pdim, dim, delta, b) {
			break
		}
		delta *= 0.6 // tighten and retry
	}

	bl := Block{
		Dim:    dim,
		Count:  len(vectors),
		Seed:   hadamardSeed,
		Delta:  delta,
		Coords: encodeCoords(q),
	}
	return bl
}

// budgetMet reconstructs from q and checks the achieved error against b.
func budgetMet(orig [][]float64, flat []float64, q []int32, pdim, dim int, delta float64, b Budget) bool {
	recon := reconstruct(q, pdim, dim, len(orig), delta, flat)
	got := metric(orig, recon, b.Kind)
	switch b.Kind {
	case KindCosine:
		return got >= b.Val
	default: // RelError, SNR expressed as rel
		return got <= relTarget(b)
	}
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
func reconstruct(q []int32, pdim, dim, count int, delta float64, scratch []float64) [][]float64 {
	out := make([][]float64, count)
	row := make([]float64, pdim)
	for i := 0; i < count; i++ {
		seg := q[i*pdim : i*pdim+pdim]
		for j := 0; j < pdim; j++ {
			row[j] = float64(seg[j]) * delta
		}
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
	return reconstruct(q, pdim, bl.Dim, bl.Count, bl.Delta, nil)
}
