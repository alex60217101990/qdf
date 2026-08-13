package main

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// What the fallback costs against what it replaces.
//
// The comparison is columnar-vs-row-major for the SAME values decoded into the
// SAME type, not fallback-vs-error: before this branch the columnar arm did not
// decode at all, so there is no earlier number for it. OptSpeed writes the
// row-major form of the identical slice, which is what a caller would have had
// to fall back to.
func benchDecodeInto(b *testing.B, opts qdf.Options, n int) {
	wire, err := qdf.Marshal(mkServices(n), opts)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(wire)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		var into []GenService
		if err := qdf.Unmarshal(wire, &into); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFallbackColumnar512(b *testing.B) { benchDecodeInto(b, qdf.OptBalanced, 512) }
func BenchmarkFallbackRowMajor512(b *testing.B) { benchDecodeInto(b, qdf.OptSpeed, 512) }
