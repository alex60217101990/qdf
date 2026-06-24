// Package vecquant orchestrates the lossy vector pipeline: it maps a user
// quality budget to a quantization step, runs Hadamard rotation → scalar
// quantization → rANS entropy coding of the integer coordinates, and verifies
// the achieved error on the real data.
package vecquant

import "math"

// BudgetKind selects how the caller expresses the fidelity target.
type BudgetKind uint8

const (
	KindRelError BudgetKind = iota // max relative L2 error of the vector
	KindCosine                     // min cosine similarity
	KindSNR                        // target SNR in dB
)

// Budget is a single fidelity target. Exactly one is in force per encode.
type Budget struct {
	Kind BudgetKind
	Val  float64
}

// DeltaFor returns the quantization step Δ predicted to meet the budget, given
// the RMS (sigma) of the rotated coordinates and the lattice second moment g.
// Model: per-coordinate MSE ≈ Δ²·g, total rel² ≈ (Δ²·g)/sigma².
func DeltaFor(b Budget, sigma, g float64) float64 {
	switch b.Kind {
	case KindRelError:
		// rel = sqrt(g)*Δ/sigma  =>  Δ = sigma*rel/sqrt(g)
		return sigma * b.Val / math.Sqrt(g)
	case KindCosine:
		// cos ≈ 1 - rel²/2  =>  rel² = 2(1-cos)
		rel2 := 2 * (1 - b.Val)
		if rel2 < 0 {
			rel2 = 0
		}
		return sigma * math.Sqrt(rel2/g)
	case KindSNR:
		// SNR_dB = 10·log10(sigma²/(Δ²·g))  =>  Δ = sigma/sqrt(g) · 10^(-SNR/20)
		return sigma / math.Sqrt(g) * math.Pow(10, -b.Val/20)
	default:
		return sigma // degenerate: ~1 bit
	}
}
