package qdf

import (
	"math"
	"testing"
)

// TestLossyVecBareTopLevelSlice guards against the headerless-output regression:
// when a bare []float32/[]float64 (not wrapped in a struct) is the top-level
// Marshal value under OptLossyVec and the lossy block wins the never-worse
// comparison, the lossless write's stream header is rolled back. The lossy
// append must re-emit the header, or the output starts with 0xFD and decodes
// as "not a qdf stream".
func TestLossyVecBareTopLevelSlice(t *testing.T) {
	for _, n := range []int{32, 64, 128, 256, 512} {
		f32 := make([]float32, n)
		f64 := make([]float64, n)
		for i := range f32 {
			x := math.Sin(float64(i) * 0.1)
			f32[i] = float32(x)
			f64[i] = x
		}
		opts := OptBalanced | OptLossyVec
		b32, err := Marshal(f32, opts)
		if err != nil {
			t.Fatalf("marshal f32 n=%d: %v", n, err)
		}
		var out32 []float32
		if err := Unmarshal(b32, &out32); err != nil {
			t.Fatalf("unmarshal bare f32 n=%d: %v (head=% x)", n, err, b32[:min(8, len(b32))])
		}
		if len(out32) != n {
			t.Fatalf("f32 n=%d: got len %d", n, len(out32))
		}

		b64, err := Marshal(f64, opts)
		if err != nil {
			t.Fatalf("marshal f64 n=%d: %v", n, err)
		}
		var out64 []float64
		if err := Unmarshal(b64, &out64); err != nil {
			t.Fatalf("unmarshal bare f64 n=%d: %v (head=% x)", n, err, b64[:min(8, len(b64))])
		}
		if len(out64) != n {
			t.Fatalf("f64 n=%d: got len %d", n, len(out64))
		}
	}
}
