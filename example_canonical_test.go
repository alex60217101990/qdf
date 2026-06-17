package qdf_test

import (
	"crypto/sha256"
	"fmt"
	"math"

	"github.com/alex60217101990/qdf"
)

// ExampleOptions_canonical shows OptCanonical: two values that are logically
// equal but built differently — maps inserted in a different order, and one
// cost carrying -0.0 instead of +0.0 — serialize to byte-identical output, so a
// hash of the bytes is stable. That makes a canonical encoding safe to hash,
// sign, content-address, or deduplicate.
//
// Without OptCanonical the two encodings may differ: Go randomizes map iteration
// order per range, and -0.0 carries a distinct sign bit. OptCanonical sorts map
// keys (every key kind) and normalizes floats (-0.0 → +0.0, any NaN → one quiet
// NaN), collapsing both to a single stable encoding. It is encode-side only —
// the bytes decode like any other qdf output.
func ExampleOptions_canonical() {
	type Record struct {
		ID   string         `qdf:"id"`
		Tags map[string]int `qdf:"tags"`
		Cost float64        `qdf:"cost"`
	}

	a := Record{ID: "x", Tags: map[string]int{"a": 1, "b": 2}, Cost: 0.0}
	b := Record{ID: "x", Tags: map[string]int{"b": 2, "a": 1}, Cost: math.Copysign(0, -1)}

	ba, _ := qdf.Marshal(a, qdf.OptBalanced|qdf.OptCanonical)
	bb, _ := qdf.Marshal(b, qdf.OptBalanced|qdf.OptCanonical)

	fmt.Println("hashes equal:", sha256.Sum256(ba) == sha256.Sum256(bb))

	// Output:
	// hashes equal: true
}
