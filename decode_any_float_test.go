package qdf

import (
	"math"
	"testing"
)

// A []float64 that Chimp128 (tagPackGorilla) or ALP-RD (tagPackALP) compressed
// carries kind qpackKindChimp64, which decodeAnyPackedSlice did not handle: a
// value decoded into any / map[string]any failed with ErrBadTag while the same
// wire decoded fine into a typed target. Both codecs share that kind byte and
// are told apart by their tag, so one case covers both.
func TestDecodeAny_CompressedFloatSlices(t *testing.T) {
	vals := make([]float64, 512)
	for i := range vals {
		vals[i] = 20.0 + math.Sin(float64(i)/10)*0.5 // smooth: Chimp/Gorilla territory
	}
	v32 := make([]float32, 512)
	for i := range v32 {
		v32[i] = float32(vals[i])
	}
	for _, o := range []Options{
		OptBalanced | OptGorillaFloat,
		OptCompression,
		OptQPack | OptGorillaFloat,
	} {
		b, err := Marshal(vals, o)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", o, err)
		}
		var out any
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("opts=%d float64 into any: %v", o, err)
		}
		got, ok := out.([]float64)
		if !ok {
			t.Fatalf("opts=%d: decoded as %T, want []float64", o, out)
		}
		if len(got) != len(vals) {
			t.Fatalf("opts=%d: got %d values, want %d", o, len(got), len(vals))
		}
		for i := range vals {
			if got[i] != vals[i] {
				t.Fatalf("opts=%d index %d: got %v want %v", o, i, got[i], vals[i])
			}
		}

		b32, err := Marshal(v32, o)
		if err != nil {
			t.Fatalf("opts=%d float32 marshal: %v", o, err)
		}
		var out32 any
		if err := Unmarshal(b32, &out32); err != nil {
			t.Fatalf("opts=%d float32 into any: %v", o, err)
		}
		g32, ok := out32.([]float32)
		if !ok {
			t.Fatalf("opts=%d: decoded as %T, want []float32", o, out32)
		}
		for i := range v32 {
			if g32[i] != v32[i] {
				t.Fatalf("opts=%d float32 index %d: got %v want %v", o, i, g32[i], v32[i])
			}
		}
	}
}
