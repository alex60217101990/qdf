package vecquant

import (
	"math"
	"testing"

	"github.com/alex60217101990/qdf/internal/lattice"
)

func TestDeltaForRelError(t *testing.T) {
	// rel^2 = (delta^2 * g) / sigma^2  =>  delta = sigma*sqrt(eps^2/g... )
	// For RelError eps: delta = sigma * eps / sqrt(g).
	sigma := 2.0
	eps := 0.01
	d := DeltaFor(Budget{Kind: KindRelError, Val: eps}, sigma, lattice.ScalarG)
	// achieved rel error ≈ sqrt(g)*delta/sigma
	rel := math.Sqrt(lattice.ScalarG) * d / sigma
	if math.Abs(rel-eps)/eps > 1e-9 {
		t.Fatalf("rel=%v want %v", rel, eps)
	}
}

func TestDeltaForSNRMonotone(t *testing.T) {
	sigma := 1.5
	dLo := DeltaFor(Budget{Kind: KindSNR, Val: 20}, sigma, lattice.ScalarG)
	dHi := DeltaFor(Budget{Kind: KindSNR, Val: 40}, sigma, lattice.ScalarG)
	if !(dHi < dLo) {
		t.Fatalf("higher SNR must give smaller delta: dHi=%v dLo=%v", dHi, dLo)
	}
}
