package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func TestGorillaFloat64_RoundTrip(t *testing.T) {
	cases := [][]float64{
		nil,
		{0},
		{0, 0, 0, 0, 0},
		{1.5, 1.5, 1.5, 1.5},
		{1.5, 1.6, 1.7, 1.8, 1.9, 2.0},
		{0, -0, math.NaN(), math.Inf(1), math.Inf(-1), 1e300, -1e-300},
		{0.1, 0.2, 0.3, 0.4, 0.5, 0.5, 0.5, 0.4, 0.3},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedGorillaFloat64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackGorilla {
			t.Fatalf("expected tagPackGorilla got %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedGorillaFloat64Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(in) == 0 {
			if len(out) != 0 {
				t.Fatal("empty mismatch")
			}
			continue
		}
		if len(in) != len(out) {
			t.Fatalf("len mismatch %d vs %d", len(in), len(out))
		}
		for i := range in {
			if math.IsNaN(in[i]) {
				if !math.IsNaN(out[i]) {
					t.Fatalf("[%d] NaN lost", i)
				}
				continue
			}
			if math.Float64bits(in[i]) != math.Float64bits(out[i]) {
				t.Fatalf("[%d] %v != %v", i, in[i], out[i])
			}
		}
	}
}

func TestGorillaFloat32_RoundTrip(t *testing.T) {
	cases := [][]float32{
		nil,
		{0},
		{1.5, 1.5, 1.5},
		{0, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))},
		{0.1, 0.2, 0.3, 0.4, 0.5},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedGorillaFloat32Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackGorilla {
			t.Fatalf("expected tagPackGorilla got %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedGorillaFloat32Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(in) != len(out) {
			t.Fatalf("len %d vs %d", len(in), len(out))
		}
		for i := range in {
			if math.IsNaN(float64(in[i])) {
				if !math.IsNaN(float64(out[i])) {
					t.Fatalf("[%d] NaN lost", i)
				}
				continue
			}
			if math.Float32bits(in[i]) != math.Float32bits(out[i]) {
				t.Fatalf("[%d] %v != %v", i, in[i], out[i])
			}
		}
	}
}

func TestGorillaFloat64_RandomRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, n := range []int{0, 1, 2, 8, 64, 1024} {
		in := make([]float64, n)
		base := rng.Float64() * 1e6
		for i := range in {
			in[i] = base + rng.NormFloat64()
			base = in[i]
		}
		enc := NewEncoder(Fast)
		enc.writePackedGorillaFloat64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		_, _ = dec.peekTag()
		dec.i++
		out, err := dec.readPackedGorillaFloat64Slice()
		if err != nil {
			t.Fatalf("decode n=%d: %v", n, err)
		}
		if !reflect.DeepEqual(in, out) {
			for i := range in {
				if math.Float64bits(in[i]) != math.Float64bits(out[i]) {
					t.Fatalf("n=%d [%d] %v != %v", n, i, in[i], out[i])
				}
			}
		}
	}
}

func TestGorillaSizeBetter(t *testing.T) {
	in := make([]float64, 1024)
	for i := range in {
		in[i] = 25.0 + 0.01*float64(i%32) // slow varying sensor-like
	}
	encG := NewEncoder(Fast)
	encG.writePackedGorillaFloat64Slice(in)
	encR := NewEncoder(Fast)
	encR.SetQPack(true)
	encR.writePackedFloat64Slice(in)
	t.Logf("gorilla=%d  raw=%d  ratio=%.2fx", len(encG.buf), len(encR.buf), float64(len(encR.buf))/float64(len(encG.buf)))
	if len(encG.buf) >= len(encR.buf) {
		t.Fatal("gorilla not smaller than raw")
	}
}

func TestGorilla_SkipIntegration(t *testing.T) {
	in := []float64{1, 2, 3, 4, 5}
	enc := NewEncoder(Fast)
	enc.writePackedGorillaFloat64Slice(in)
	enc.WriteInt(123)
	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatal(err)
	}
	v, err := dec.ReadInt()
	if err != nil || v != 123 {
		t.Fatalf("after skip v=%d err=%v", v, err)
	}
}

func makeSensorFloat64(n int) []float64 {
	s := make([]float64, n)
	rng := rand.New(rand.NewSource(42))
	base := 25.0
	for i := range s {
		base += (rng.Float64() - 0.5) * 0.05
		s[i] = base
	}
	return s
}

func BenchmarkQPackGorilla_Float64(b *testing.B) {
	for _, n := range []int{1024, 16384} {
		in := makeSensorFloat64(n)
		encG := NewEncoder(Fast)
		encG.writePackedGorillaFloat64Slice(in)
		encR := NewEncoder(Fast)
		encR.SetQPack(true)
		encR.writePackedFloat64Slice(in)
		b.Logf("n=%d  raw=%d  gorilla=%d  ratio=%.2fx", n, len(encR.buf), len(encG.buf), float64(len(encR.buf))/float64(len(encG.buf)))

		b.Run("encode/gorilla/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				enc.Reset()
				enc.writePackedGorillaFloat64Slice(in)
			}
		})
		gorBuf := append([]byte(nil), encG.buf...)
		b.Run("decode/gorilla/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(gorBuf)
				_, _ = dec.peekTag()
				dec.i++
				_, _ = dec.readPackedGorillaFloat64Slice()
			}
		})
	}
}
