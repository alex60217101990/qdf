package vecquant

import (
	"math/rand"
	"testing"
)

func gv(count, dim int, seed int64) [][]float64 {
	r := rand.New(rand.NewSource(seed))
	out := make([][]float64, count)
	for i := range out {
		v := make([]float64, dim)
		for j := range v {
			v[j] = r.NormFloat64()
		}
		out[i] = v
	}
	return out
}

// TestEncodeWithReuseByteIdentical proves a reused Scratch yields the exact same
// Block bytes as a fresh encode (no stale-data leak across calls).
func TestEncodeWithReuseByteIdentical(t *testing.T) {
	a := gv(40, 256, 1)
	bdg := Budget{Kind: KindRelError, Val: 0.02}

	fresh := Encode(a, bdg)

	var sc Scratch
	_ = EncodeWith(gv(40, 256, 9), bdg, &sc) // dirty the scratch with other data
	sc.Reset()
	reused := EncodeWith(a, bdg, &sc)

	if fresh.Variant != reused.Variant || string(fresh.Coords) != string(reused.Coords) || string(fresh.Cosets) != string(reused.Cosets) || fresh.Delta != reused.Delta {
		t.Fatalf("reused scratch produced different bytes: fresh{v=%d,len=%d} reused{v=%d,len=%d}",
			fresh.Variant, len(fresh.Coords), reused.Variant, len(reused.Coords))
	}
}

func TestScratchResetDropsOversized(t *testing.T) {
	var sc Scratch
	// Grow flat past the retention ceiling via a large encode.
	big := gv(2, (maxRetainedScratch/2)+8, 1) // count*pdim > maxRetainedScratch
	_ = EncodeWith(big, Budget{Kind: KindRelError, Val: 0.05}, &sc)
	if cap(sc.flat) <= maxRetainedScratch {
		t.Skip("flat did not exceed ceiling on this shape; nothing to assert")
	}
	sc.Reset()
	if sc.flat != nil {
		t.Fatalf("Reset kept oversized flat (cap=%d > %d)", cap(sc.flat), maxRetainedScratch)
	}
}
