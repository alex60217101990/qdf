package main

// THROWAWAY measure-first probe for the parked "polar-split" idea: store each
// vector's L2 norm separately (1 scalar/vector) and quantize the unit-direction
// through the lattice path, instead of quantizing the raw vector with one shared
// step Δ over the whole block.
//
// Hypothesis: the current codec picks ONE Δ for the whole block (driven by the
// hardest vector), so when per-vector NORMS vary a lot it wastes bits on the
// easy (small-norm) vectors. Normalizing first removes that spread, letting one
// tight Δ serve all directions. On unit-norm embeddings (norm ~= const) polar
// should buy nothing; on varying-norm data it might.
//
// Run: go test -run TestPolarSplitProbe -v

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

// scaleRows multiplies each vector by a per-vector factor drawn log-normally,
// producing a corpus with a wide spread of L2 norms (the regime polar targets).
func scaleRows(corpus [][]float64, sigmaLog float64, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for _, v := range corpus {
		s := math.Exp(sigmaLog * r.NormFloat64())
		for j := range v {
			v[j] *= s
		}
	}
}

// rowNorm returns the L2 norm of v.
func rowNorm(v []float64) float64 {
	var s float64
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s)
}

// polarEncodeBytes simulates the polar-split wire cost and reconstruction:
//   - per-vector norm quantized to 16 bits over [min,max]  -> 2 bytes/vector;
//   - unit directions encoded through the current lattice codec at budget b.
//
// It returns the total bytes/vector and the achieved rel-error measured on the
// ORIGINAL (un-normalized) vectors, so it is comparable to the baseline.
func polarEncodeBytes(corpus [][]float64, b vecquant.Budget) (bytesPerVec, relErr float64) {
	n := len(corpus)
	dim := len(corpus[0])
	norms := make([]float64, n)
	dirs := make([][]float64, n)
	mn, mx := math.Inf(1), math.Inf(-1)
	for i, v := range corpus {
		nrm := rowNorm(v)
		norms[i] = nrm
		mn, mx = math.Min(mn, nrm), math.Max(mx, nrm)
		d := make([]float64, dim)
		if nrm > 0 {
			for j := range v {
				d[j] = v[j] / nrm
			}
		}
		dirs[i] = d
	}
	// Encode the unit directions with the real codec.
	bl := vecquant.Encode(dirs, b)
	dirRecon := bl.Decode()
	dirBytes := lossyVecWireBytes(bl)
	// Norm cost: 16-bit quantized per vector + a tiny header (min,max f64).
	normBytes := 2*n + 16
	// Reconstruct v_hat = norm_q * dir_hat, quantizing norm to 16 bits.
	span := mx - mn
	if span == 0 {
		span = 1
	}
	var seSum, neSum float64
	for i := range corpus {
		q := math.RoundToEven((norms[i] - mn) / span * 65535)
		normQ := mn + q/65535*span
		o := corpus[i]
		var se, ne float64
		for j := range o {
			vh := normQ * dirRecon[i][j]
			d := o[j] - vh
			se += d * d
			ne += o[j] * o[j]
		}
		seSum += math.Sqrt(se / (ne + 1e-30))
		neSum++
	}
	bytesPerVec = float64(dirBytes+normBytes) / float64(n)
	relErr = seSum / neSum
	return bytesPerVec, relErr
}

func TestPolarSplitProbe(t *testing.T) {
	const (
		n   = 2000
		dim = 256
	)
	type corpusCase struct {
		name   string
		corpus [][]float64
	}
	// Control: unit-norm (polar expected ~0). Test: wide log-normal norm spread.
	unit := loadSynthetic(n, dim, 42)
	normalizeRows(unit)
	mild := loadSynthetic(n, dim, 42)
	normalizeRows(mild)
	scaleRows(mild, 0.25, 99) // sigma_log=0.25 => ~3-4x norm spread (mild)
	mod := loadSynthetic(n, dim, 42)
	normalizeRows(mod)
	scaleRows(mod, 0.5, 99) // sigma_log=0.5 => ~15-20x spread (realistic weights)
	wide := loadSynthetic(n, dim, 42)
	scaleRows(wide, 1.0, 99) // sigma_log=1 => ~1000x (pathological)

	cases := []corpusCase{
		{"unit-norm", unit},
		{"mild(σ0.25,~4x)", mild},
		{"moderate(σ0.5,~20x)", mod},
		{"wide(σ1.0,~1000x)", wide},
	}
	budgets := []vecquant.Budget{
		{Kind: vecquant.KindRelError, Val: 0.05},
		{Kind: vecquant.KindRelError, Val: 0.10},
	}

	for _, cc := range cases {
		// Report the norm spread for context.
		mn, mx := math.Inf(1), math.Inf(-1)
		for _, v := range cc.corpus {
			nrm := rowNorm(v)
			mn, mx = math.Min(mn, nrm), math.Max(mx, nrm)
		}
		fmt.Printf("\n=== %s — %d×%d, norm range [%.3g, %.3g] (%.0f×) ===\n",
			cc.name, n, dim, mn, mx, mx/mn)
		fmt.Printf("%-10s %14s %10s %14s %10s %9s\n",
			"budget", "cur B/vec", "cur rel", "polar B/vec", "polar rel", "Δsize")
		for _, b := range budgets {
			bl := vecquant.Encode(cc.corpus, b)
			curBytes := float64(lossyVecWireBytes(bl)) / float64(len(cc.corpus))
			curRel := avgRelError(cc.corpus, bl.Decode())
			polBytes, polRel := polarEncodeBytes(cc.corpus, b)
			dPct := 100 * (polBytes - curBytes) / curBytes
			fmt.Printf("rel/%.2f   %14.2f %10.4f %14.2f %10.4f %+8.1f%%\n",
				b.Val, curBytes, curRel, polBytes, polRel, dPct)
		}
	}
	fmt.Println("\nVerdict: polar wins only if Δsize is meaningfully negative AT")
	fmt.Println("equal-or-better rel-error. On unit-norm it should be ~0 or worse.")
}
