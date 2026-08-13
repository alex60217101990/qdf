package main

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// The bit must be its own, not an alias of a neighbour.
func TestColumnarGeneratedBitIsDistinct(t *testing.T) {
	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"OptDense", qdf.OptDense},
		{"OptQPack", qdf.OptQPack},
		{"OptStringAlphabet", qdf.OptStringAlphabet},
		{"OptColumnIndex", qdf.OptColumnIndex},
		{"OptRANS", qdf.OptRANS},
	} {
		if qdf.OptColumnarGenerated&o.opts != 0 {
			t.Errorf("OptColumnarGenerated overlaps %s", o.name)
		}
	}
	if qdf.OptBalanced&qdf.OptColumnarGenerated != 0 {
		t.Error("OptBalanced includes OptColumnarGenerated — it must be opt-in")
	}
	if qdf.OptCompression&qdf.OptColumnarGenerated != 0 {
		t.Error("OptCompression includes OptColumnarGenerated — it must be opt-in")
	}
}
