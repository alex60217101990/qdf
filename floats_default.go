//go:build !qdf_simd

package qdf

// Default bulk float encoders. Element-by-element through WriteFloatN.
// Replaced under -tags qdf_simd by a tighter inlined loop.

func encodeSliceFloat32Impl(e *Encoder, s []float32) error {
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteFloat32(s[i])
	}
	return nil
}

func encodeSliceFloat64Impl(e *Encoder, s []float64) error {
	e.WriteArrayHeader(len(s))
	for i := range s {
		e.WriteFloat64(s[i])
	}
	return nil
}
