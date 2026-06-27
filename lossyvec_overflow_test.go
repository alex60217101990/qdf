package qdf

import (
	"math"
	"testing"
)

// TestLossyVec_Int32OverflowFallsBackToLossless pins the audit-10 fix for the
// silent int32 saturation in the lattice quantizers. With an extremely tight
// fidelity budget the scalar/E8 quantizer's coordinates (v/delta) exceed the
// int32 range; a raw cast would wrap (sign-flip) and silently corrupt the
// vector. The fix clamps and signals overflow so the encoder keeps the lossless
// encoding instead — the field must round-trip EXACTLY.
func TestLossyVec_Int32OverflowFallsBackToLossless(t *testing.T) {
	type vecRow struct {
		V []float64 `qdf:"v"`
	}

	// A spiky vector: one large coordinate dominates rms (sigma), so at a tiny
	// rel budget delta = sigma*rel/sqrt(g) is small enough that v/delta blows
	// past 2^31 for the spike.
	dim := 64
	v := make([]float64, dim)
	for i := range v {
		v[i] = math.Sin(float64(i) * 0.3)
	}
	v[0] = 1e6 // spike

	rows := []vecRow{{V: v}}

	// MaxRelError(1e-12) is far tighter than the codec can meet without int32
	// overflow on the spike → must fall back to lossless (exact).
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(1e-12))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}

	var out []vecRow
	if err := Unmarshal(enc.Bytes(), &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out) != 1 || len(out[0].V) != dim {
		t.Fatalf("shape: %d rows, len %d", len(out), len(out[0].V))
	}
	for i := range v {
		// Bit-exact: the lossless fallback must reproduce every coordinate.
		if math.Float64bits(out[0].V[i]) != math.Float64bits(v[i]) {
			t.Fatalf("coord %d: got %v want %v (not bit-exact — lossy block was kept despite int32 overflow)",
				i, out[0].V[i], v[i])
		}
	}
}
