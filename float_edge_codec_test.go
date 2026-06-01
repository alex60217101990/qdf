package qdf

import (
	"math"
	"testing"
)

// floatBits64 returns the bit patterns of each float64 for diagnostic messages.
func floatBits64(s []float64) []uint64 {
	out := make([]uint64, len(s))
	for i, v := range s {
		out[i] = math.Float64bits(v)
	}
	return out
}

// f64bitsEqual compares two []float64 with bit-exact semantics:
// NaN bit patterns must match; -0 and +0 are distinct.
func f64bitsEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return false
		}
	}
	return true
}

// f32bitsEqual compares two []float32 with bit-exact semantics.
func f32bitsEqual(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float32bits(a[i]) != math.Float32bits(b[i]) {
			return false
		}
	}
	return true
}

// TestFloatEdge_F64_SliceCodecs drives NaN, ±Inf, negative-zero and other
// edge-case float64 values through every codec bundle that touches []float64.
// Failure here is a REAL qdf bit-preservation bug.
func TestFloatEdge_F64_SliceCodecs(t *testing.T) {
	type rec struct{ V []float64 }
	nz := math.Copysign(0, -1) // genuine negative zero — Go literal -0.0 is +0.0
	v := rec{V: []float64{
		0, nz, 1, -1,
		math.Inf(1), math.Inf(-1),
		math.NaN(), math.MaxFloat64, math.SmallestNonzeroFloat64,
	}}
	for _, opts := range []Options{OptQPack, OptBalanced, OptCompression, OptQPack | OptGorillaFloat} {
		data, err := Marshal(v, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		var out rec
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("opts=%d unmarshal: %v", opts, err)
		}
		if !f64bitsEqual(v.V, out.V) {
			t.Fatalf("opts=%d float64 bits NOT preserved:\n in =%v\n out=%v\n inbits =%x\n outbits=%x",
				opts, v.V, out.V, floatBits64(v.V), floatBits64(out.V))
		}
	}
}

// TestFloatEdge_F32_SliceCodecs drives NaN, ±Inf, negative-zero and other
// edge-case float32 values through every codec bundle that touches []float32.
// Failure here is a REAL qdf bit-preservation bug.
func TestFloatEdge_F32_SliceCodecs(t *testing.T) {
	type rec struct{ V []float32 }
	nz := float32(math.Copysign(0, -1))
	v := rec{V: []float32{
		0, nz, 1, -1,
		float32(math.Inf(1)), float32(math.Inf(-1)),
		float32(math.NaN()), math.MaxFloat32, math.SmallestNonzeroFloat32,
	}}
	for _, opts := range []Options{OptQPack, OptBalanced, OptCompression, OptQPack | OptGorillaFloat} {
		data, err := Marshal(v, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		var out rec
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("opts=%d unmarshal: %v", opts, err)
		}
		if !f32bitsEqual(v.V, out.V) {
			t.Fatalf("opts=%d float32 bits NOT preserved:\n in=%v\n out=%v", opts, v.V, out.V)
		}
	}
}
