package qdf

import (
	"bytes"
	"testing"
)

// OptCanonical outranks the map-shape branch, on the generated fast paths as
// well as in the reflect encoder.
//
// The reflect encoder documents the rule — "canonical emit takes precedence over
// the OptMapShape/OptDense shape branch … regardless of the other shape bits" —
// and the generated map paths used to test the shape branch FIRST. Both forms
// are deterministic on their own, so the symptom was never randomness: it was
// that one logical value got two encodings depending on whether its (K,V) pair
// happened to have a generated fast path. OptCanonical promises bytes a caller
// can hash, sign and content-address, which that breaks.
//
// Asserted by comparing against canonical WITHOUT the shape bit rather than by
// scanning for tagMapShape: that tag also carries struct shapes under
// OptShapeIntern, so a byte scan catches shapes this test is not about.
func TestCanonicalOutranksTheShapeBranch(t *testing.T) {
	for _, c := range []struct {
		what string
		v    any
	}{
		{"generated pair map[string]string", any(map[string]string{"bb": "2", "aa": "1", "cc": "3"})},
		{"generated pair map[string]int64", any(map[string]int64{"bb": 2, "aa": 1, "cc": 3})},
	} {
		plain, err := Marshal(c.v, OptCanonical|OptDense)
		if err != nil {
			t.Fatalf("%s: %v", c.what, err)
		}
		withShape, err := Marshal(c.v, OptCanonical|OptDense|OptMapShape)
		if err != nil {
			t.Fatalf("%s: %v", c.what, err)
		}
		if !bytes.Equal(plain, withShape) {
			t.Errorf("%s: adding OptMapShape changed a CANONICAL encoding (%d -> %d bytes) — "+
				"the shape branch outranked canonical, so the same value encodes two ways",
				c.what, len(plain), len(withShape))
		}
	}
}
