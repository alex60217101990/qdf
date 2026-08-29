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
//
// RUN IT BOTH WAYS — the answer changes sign with the build. Nearly all of the
// columnar decode is bit-unpacking an alphabet-packed string column, and
// internal/bitpack has hand-written assembly for it behind the qdf_simd tag,
// which is off by default:
//
//	default (scalar)   columnar 194.6us   row-major 134.5us   columnar LOSES
//	-tags qdf_simd     columnar 118.8us   row-major 133.5us   columnar WINS
//
// -38.96% on the columnar arm (p=0.000, n=10) and no movement on row-major,
// which is the control: the vector path touches only the unpacking that the
// columnar form uses. Quoting either row without naming the build is how a
// wrong verdict gets recorded, and one already was.
func benchDecodeInto(b *testing.B, opts qdf.Options, n int) {
	b.Helper()
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
