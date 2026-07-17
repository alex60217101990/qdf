package hadamard

import (
	"math"
	"strconv"
	"testing"
)

func TestForwardInverseRoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 4, 8, 16, 64, 256} {
		x := make([]float64, n)
		orig := make([]float64, n)
		for i := range x {
			x[i] = math.Sin(float64(i)*0.7) * 3.0
			orig[i] = x[i]
		}
		const seed = 0x9e3779b97f4a7c15
		Forward(x, seed)
		Inverse(x, seed)
		for i := range x {
			if math.Abs(x[i]-orig[i]) > 1e-9 {
				t.Fatalf("n=%d i=%d: round-trip %v != %v", n, i, x[i], orig[i])
			}
		}
	}
}

func TestForwardIsOrthonormal(t *testing.T) {
	// Energy (L2 norm) is preserved by an orthonormal rotation.
	n := 64
	x := make([]float64, n)
	var e0 float64
	for i := range x {
		x[i] = float64(i%7) - 3
		e0 += x[i] * x[i]
	}
	Forward(x, 42)
	var e1 float64
	for _, v := range x {
		e1 += v * v
	}
	if math.Abs(e0-e1) > 1e-9 {
		t.Fatalf("energy not preserved: %v != %v", e0, e1)
	}
}

func TestNextPow2(t *testing.T) {
	cases := map[int]int{1: 1, 2: 2, 3: 4, 5: 8, 768: 1024, 1024: 1024}
	for in, want := range cases {
		if got := NextPow2(in); got != want {
			t.Fatalf("NextPow2(%d)=%d want %d", in, got, want)
		}
	}
}

func BenchmarkForward(b *testing.B) {
	for _, n := range []int{64, 256, 512, 1024} {
		x := make([]float64, n)
		for i := range x {
			x[i] = math.Sin(float64(i) * 0.7)
		}
		tmp := make([]float64, n)
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				copy(tmp, x)
				Forward(tmp, 0x9e3779b97f4a7c15)
			}
		})
	}
}

func BenchmarkInverse(b *testing.B) {
	for _, n := range []int{64, 256, 512, 1024} {
		x := make([]float64, n)
		for i := range x {
			x[i] = math.Sin(float64(i) * 0.7)
		}
		Forward(x, 0x9e3779b97f4a7c15)
		tmp := make([]float64, n)
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				copy(tmp, x)
				Inverse(tmp, 0x9e3779b97f4a7c15)
			}
		})
	}
}
