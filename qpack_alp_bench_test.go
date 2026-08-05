package qdf

import (
	"fmt"
	"math"
	"testing"
)

// BenchmarkQPackALP_Float64_Encode measures the full ALP float64 encode path
// (plan + write) for quantized telemetry-style data at exponent 2.
// Used to gate PERF-ALP-1 (double-scan elimination).
func BenchmarkQPackALP_Float64_Encode(b *testing.B) {
	for _, n := range []int{256, 4096, 16384} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			s := make([]float64, n)
			for i := range s {
				// Quantized values: multiples of 0.01 — ALP exponent 2 encodes perfectly.
				s[i] = math.Round(float64(i%1000)*0.01*100) / 100
			}
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				enc.buf = enc.buf[:0]
				plan, _, ok := alpPlanFloat64(s)
				if !ok {
					b.Fatal("ALP plan failed")
				}
				enc.writePackedALPFloat64Slice(s, plan)
			}
		})
	}
}
