//go:build qdf_simd

// Bulk float encoders selected by the qdf_simd build tag.
//
//	GOEXPERIMENT=simd go build -tags qdf_simd ./...     # amd64
//	go build -tags qdf_simd ./...                       # arm64
//
// The wire format stores floats in little-endian byte order, which
// matches both amd64 and arm64. The hot path here is therefore a tight
// inlined loop of tag + LE store; the build-tag exists so future work
// can plug in true lane-level intrinsics where they help (batch varint
// decode, UTF-8 validation, intern-table hashing).
package qdf

import (
	"math"
	"slices"
)

func encodeSliceFloat32Impl(e *Encoder, s []float32) error {
	n := len(s)
	e.WriteArrayHeader(n)
	if n == 0 {
		return nil
	}
	e.buf = slices.Grow(e.buf, n*5) // tag + 4-byte body per element
	for i := 0; i < n; i++ {
		e.buf = appendU32(append(e.buf, tagFloat32), math.Float32bits(s[i]))
	}
	return nil
}

func encodeSliceFloat64Impl(e *Encoder, s []float64) error {
	n := len(s)
	e.WriteArrayHeader(n)
	if n == 0 {
		return nil
	}
	e.buf = slices.Grow(e.buf, n*9) // tag + 8-byte body per element
	for i := 0; i < n; i++ {
		e.buf = appendU64(append(e.buf, tagFloat64), math.Float64bits(s[i]))
	}
	return nil
}
