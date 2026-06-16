package qdf

import "math"

// Canonical encoding (OptCanonical) helpers. Float normalization makes
// logically-equal float values serialize identically: -0.0 collapses to +0.0
// and any NaN bit pattern maps to a single canonical quiet NaN. The pooled
// map-key sort scratch (canonKeys* on encState) is reused across maps so the
// sorted-key emit allocates nothing in steady state.

// canonicalizeFloat64 maps -0.0 → +0.0 and any NaN → a canonical quiet NaN, so
// logically-equal float values serialize identically under OptCanonical.
func canonicalizeFloat64(v float64) float64 {
	if v == 0 {
		return 0 // collapses -0.0 to +0.0 (Go: -0.0 == 0)
	}
	if math.IsNaN(v) {
		return math.Float64frombits(0x7FF8000000000000)
	}
	return v
}

// canonicalizeFloat32 maps -0.0 → +0.0 and any NaN → a canonical quiet NaN.
func canonicalizeFloat32(v float32) float32 {
	if v == 0 {
		return 0
	}
	if v != v { // NaN
		return math.Float32frombits(0x7FC00000)
	}
	return v
}

// canonicalizeFloat32Bits normalizes raw float32 bits (the columnar/nullable f32
// path stores bits in a uint64 and never re-floats them).
func canonicalizeFloat32Bits(u uint64) uint64 {
	b := uint32(u)
	f := math.Float32frombits(b)
	if f == 0 {
		return 0
	}
	if f != f {
		return uint64(uint32(0x7FC00000))
	}
	return u
}
