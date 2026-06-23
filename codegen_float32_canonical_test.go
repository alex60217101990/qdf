package qdf

import (
	"bytes"
	"math"
	"testing"
)

// The codegen float32 column writer (WriteFloat32Column) must honor OptCanonical
// exactly like the reflect colKindFloat32 path: -0.0 normalizes to +0.0 and every
// NaN to one quiet NaN, so semantically-equal columns are byte-identical. Before
// the fix it always wrote raw Float32bits, so a -0.0 column diverged from a +0.0
// column (and from the reflect wire) under OptCanonical.
func TestCodegenFloat32ColumnCanonical(t *testing.T) {
	neg0 := float32(math.Copysign(0, -1))
	pos0 := float32(0)

	encCol := func(opt Options, v float32) []byte {
		e := &Encoder{}
		e.applyOpts(opt)
		if err := e.WriteFloat32Column([]float32{v}); err != nil {
			t.Fatal(err)
		}
		return append([]byte(nil), e.buf...)
	}

	// Under OptCanonical the -0.0 and +0.0 columns must encode identically.
	if !bytes.Equal(encCol(OptBalanced|OptCanonical, neg0), encCol(OptBalanced|OptCanonical, pos0)) {
		t.Fatal("codegen float32 column not canonical under OptCanonical: -0.0 encodes differently from +0.0")
	}
	// Sanity: without OptCanonical the raw sign bit is preserved, so they differ.
	if bytes.Equal(encCol(OptBalanced, neg0), encCol(OptBalanced, pos0)) {
		t.Fatal("expected -0.0 and +0.0 to encode differently without OptCanonical")
	}

	// NaN canonicalization: two different NaN bit patterns collapse to one under
	// OptCanonical.
	nanA := math.Float32frombits(0x7FC00001)
	nanB := math.Float32frombits(0x7FE00000)
	if !bytes.Equal(encCol(OptBalanced|OptCanonical, nanA), encCol(OptBalanced|OptCanonical, nanB)) {
		t.Fatal("codegen float32 column did not canonicalize distinct NaN patterns under OptCanonical")
	}
}
