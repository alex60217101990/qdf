package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

func TestQPackRaw_Uint64_RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for _, n := range []int{0, 1, 3, 8, 16, 17, 1024, 4096} {
		in := make([]uint64, n)
		for i := range in {
			in[i] = rng.Uint64()
		}
		enc := NewEncoder(Fast)
		enc.SetQPack(true)
		if err := encodeSliceUint64(enc, unsafe.Pointer(&in)); err != nil {
			t.Fatalf("encode: %v", err)
		}
		dec := NewDecoderOnBuf(enc.buf)
		var out []uint64
		if err := decodeSliceUint64(dec, unsafe.Pointer(&out)); err != nil {
			t.Fatalf("decode n=%d: %v", n, err)
		}
		if n == 0 {
			if len(out) != 0 {
				t.Fatalf("empty mismatch n=%d", n)
			}
			continue
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch n=%d", n)
		}
	}
}

func TestQPackRaw_Int64_RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, n := range []int{0, 1, 5, 64, 1024} {
		in := make([]int64, n)
		for i := range in {
			in[i] = int64(rng.Uint64())
		}
		enc := NewEncoder(Fast)
		enc.SetQPack(true)
		if err := encodeSliceInt64(enc, unsafe.Pointer(&in)); err != nil {
			t.Fatal(err)
		}
		dec := NewDecoderOnBuf(enc.buf)
		var out []int64
		if err := decodeSliceInt64(dec, unsafe.Pointer(&out)); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			continue
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch n=%d", n)
		}
	}
}

func TestQPackRaw_Uint32Int32_RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for _, n := range []int{0, 1, 7, 64, 2048} {
		inU := make([]uint32, n)
		inI := make([]int32, n)
		for i := range inU {
			inU[i] = rng.Uint32()
			inI[i] = int32(rng.Uint32())
		}
		encU := NewEncoder(Fast)
		encU.SetQPack(true)
		_ = encodeSliceUint32(encU, unsafe.Pointer(&inU))
		encI := NewEncoder(Fast)
		encI.SetQPack(true)
		_ = encodeSliceInt32(encI, unsafe.Pointer(&inI))

		var outU []uint32
		if err := decodeSliceUint32(NewDecoderOnBuf(encU.buf), unsafe.Pointer(&outU)); err != nil {
			t.Fatal(err)
		}
		var outI []int32
		if err := decodeSliceInt32(NewDecoderOnBuf(encI.buf), unsafe.Pointer(&outI)); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			continue
		}
		if !reflect.DeepEqual(inU, outU) {
			t.Fatalf("uint32 mismatch n=%d", n)
		}
		if !reflect.DeepEqual(inI, outI) {
			t.Fatalf("int32 mismatch n=%d", n)
		}
	}
}

func TestQPackRaw_Float32_RoundTrip(t *testing.T) {
	in := []float32{0, -0, 1.5, -2.25, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}
	for i := range 100 {
		in = append(in, float32(i)*0.1)
	}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	if err := encodeSliceFloat32(enc, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}
	dec := NewDecoderOnBuf(enc.buf)
	var out []float32
	if err := decodeSliceFloat32(dec, unsafe.Pointer(&out)); err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatalf("len %d vs %d", len(in), len(out))
	}
	for i, v := range in {
		if math.IsNaN(float64(v)) {
			if !math.IsNaN(float64(out[i])) {
				t.Fatalf("NaN lost at %d: %v", i, out[i])
			}
			continue
		}
		if v != out[i] {
			t.Fatalf("idx %d: %v vs %v", i, v, out[i])
		}
	}
}

func TestQPackRaw_Float64_RoundTrip(t *testing.T) {
	in := []float64{0, -0, 1.5, -2.25, math.NaN(), math.Inf(1), math.Inf(-1)}
	for i := range 100 {
		in = append(in, float64(i)*0.1)
	}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	if err := encodeSliceFloat64(enc, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}
	dec := NewDecoderOnBuf(enc.buf)
	var out []float64
	if err := decodeSliceFloat64(dec, unsafe.Pointer(&out)); err != nil {
		t.Fatal(err)
	}
	for i, v := range in {
		if math.IsNaN(v) {
			if !math.IsNaN(out[i]) {
				t.Fatalf("NaN lost at %d", i)
			}
			continue
		}
		if v != out[i] {
			t.Fatalf("idx %d: %v vs %v", i, v, out[i])
		}
	}
}

func TestQPackRaw_DecoderAcceptsBothForms(t *testing.T) {
	in := []uint64{1, 2, 3, 100, math.MaxUint64}
	legacy := NewEncoder(Fast)
	_ = encodeSliceUint64(legacy, unsafe.Pointer(&in))
	qp := NewEncoder(Fast)
	qp.SetQPack(true)
	_ = encodeSliceUint64(qp, unsafe.Pointer(&in))

	for _, raw := range [][]byte{legacy.buf, qp.buf} {
		var out []uint64
		if err := decodeSliceUint64(NewDecoderOnBuf(raw), unsafe.Pointer(&out)); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch on form")
		}
	}
}

func TestQPackRaw_KindMismatch(t *testing.T) {
	in := []uint64{1, 2, 3}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	_ = encodeSliceUint64(enc, unsafe.Pointer(&in))
	var out []int64
	if err := decodeSliceInt64(NewDecoderOnBuf(enc.buf), unsafe.Pointer(&out)); err == nil {
		t.Fatalf("expected kind mismatch error")
	}
}

func TestQPackRaw_Skip(t *testing.T) {
	in := []uint64{10, 20, 30, 40}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	_ = encodeSliceUint64(enc, unsafe.Pointer(&in))
	enc.WriteInt(99)
	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatalf("skip: %v", err)
	}
	v, err := dec.ReadInt()
	if err != nil || v != 99 {
		t.Fatalf("after skip: %v %v", v, err)
	}
}

func FuzzQPackRawUint64(f *testing.F) {
	f.Add(uint64(0), 0)
	f.Add(uint64(1), 1)
	f.Add(uint64(0xdeadbeef), 13)
	f.Fuzz(func(t *testing.T, seed uint64, n int) {
		if n < 0 || n > 4096 {
			return
		}
		in := make([]uint64, n)
		rng := rand.New(rand.NewSource(int64(seed)))
		for i := range in {
			in[i] = rng.Uint64()
		}
		enc := NewEncoder(Fast)
		enc.SetQPack(true)
		_ = encodeSliceUint64(enc, unsafe.Pointer(&in))
		var out []uint64
		if err := decodeSliceUint64(NewDecoderOnBuf(enc.buf), unsafe.Pointer(&out)); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if n == 0 {
			return
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatal("mismatch")
		}
	})
}

func BenchmarkQPackRaw_Uint64(b *testing.B) {
	for _, n := range []int{64, 1024, 16384} {
		in := make([]uint64, n)
		for i := range in {
			in[i] = uint64(i)*0x9E3779B97F4A7C15 + 1
		}
		b.Run("encode/legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceUint64(enc, unsafe.Pointer(&in))
			}
		})
		b.Run("encode/qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			enc.SetQPack(true)
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceUint64(enc, unsafe.Pointer(&in))
			}
		})
		legacy := NewEncoder(Fast)
		_ = encodeSliceUint64(legacy, unsafe.Pointer(&in))
		legacyBuf := append([]byte(nil), legacy.buf...)
		qp := NewEncoder(Fast)
		qp.SetQPack(true)
		_ = encodeSliceUint64(qp, unsafe.Pointer(&in))
		qpBuf := append([]byte(nil), qp.buf...)
		b.Run("decode/legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(legacyBuf)
				var out []uint64
				_ = decodeSliceUint64(dec, unsafe.Pointer(&out))
			}
		})
		b.Run("decode/qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(qpBuf)
				var out []uint64
				_ = decodeSliceUint64(dec, unsafe.Pointer(&out))
			}
		})
	}
}

func BenchmarkQPackRaw_Float64(b *testing.B) {
	for _, n := range []int{64, 1024, 16384} {
		in := make([]float64, n)
		for i := range in {
			in[i] = float64(i) * 0.5
		}
		b.Run("encode/legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceFloat64(enc, unsafe.Pointer(&in))
			}
		})
		b.Run("encode/qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			enc.SetQPack(true)
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceFloat64(enc, unsafe.Pointer(&in))
			}
		})
		legacy := NewEncoder(Fast)
		_ = encodeSliceFloat64(legacy, unsafe.Pointer(&in))
		legacyBuf := append([]byte(nil), legacy.buf...)
		qp := NewEncoder(Fast)
		qp.SetQPack(true)
		_ = encodeSliceFloat64(qp, unsafe.Pointer(&in))
		qpBuf := append([]byte(nil), qp.buf...)
		b.Run("decode/legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(legacyBuf)
				var out []float64
				_ = decodeSliceFloat64(dec, unsafe.Pointer(&out))
			}
		})
		b.Run("decode/qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(qpBuf)
				var out []float64
				_ = decodeSliceFloat64(dec, unsafe.Pointer(&out))
			}
		})
	}
}
