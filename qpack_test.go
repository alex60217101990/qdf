package qdf

import (
	"bytes"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

func TestQPackBool_RoundTrip(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 63, 64, 65, 127, 128, 1000, 4096}
	rng := rand.New(rand.NewSource(1))
	for _, n := range sizes {
		t.Run("", func(t *testing.T) {
			in := make([]bool, n)
			for i := range in {
				in[i] = rng.Int31()&1 == 1
			}
			enc := NewEncoder(Fast)
			enc.SetQPack(true)
			if err := encodeSliceBool(enc, unsafe.Pointer(&in)); err != nil {
				t.Fatalf("encode: %v", err)
			}
			buf := enc.Bytes()
			// Header(5) + tag(1) + varuint(<=2 for n<=16384) + ceil(n/8).
			minSize := 5 + 1 + 1 + (n+7)/8
			if len(buf) < minSize-1 {
				t.Fatalf("encoded too short: %d, want >=%d", len(buf), minSize-1)
			}
			dec := NewDecoderOnBuf(buf)
			var out []bool
			if err := decodeSliceBool(dec, unsafe.Pointer(&out)); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(in) == 0 {
				if len(out) != 0 {
					t.Fatalf("empty mismatch: %v", out)
				}
				return
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("round-trip mismatch n=%d", n)
			}
		})
	}
}

func TestQPackBool_FlagSet(t *testing.T) {
	in := []bool{true, false, true}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	if err := encodeSliceBool(enc, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}
	if enc.buf[4]&FlagQPack == 0 {
		t.Fatalf("FlagQPack not set in header byte %02x", enc.buf[4])
	}
}

func TestQPackBool_DecoderAcceptsBothForms(t *testing.T) {
	in := []bool{true, true, false, true, false, false, true, true, false}

	// Old form
	encOld := NewEncoder(Fast)
	if err := encodeSliceBool(encOld, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}

	// New form
	encNew := NewEncoder(Fast)
	encNew.SetQPack(true)
	if err := encodeSliceBool(encNew, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(encOld.buf, encNew.buf) {
		t.Fatalf("expected different wire bytes")
	}
	if len(encNew.buf) >= len(encOld.buf) {
		t.Fatalf("packed (%d) not smaller than per-element (%d)", len(encNew.buf), len(encOld.buf))
	}

	for _, raw := range [][]byte{encOld.buf, encNew.buf} {
		dec := NewDecoderOnBuf(raw)
		var out []bool
		if err := decodeSliceBool(dec, unsafe.Pointer(&out)); err != nil {
			t.Fatalf("decode form len=%d: %v", len(raw), err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("decode form len=%d: mismatch", len(raw))
		}
	}
}

func TestQPackBool_Skip(t *testing.T) {
	in := []bool{false, true, false, true, true, true, false}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	if err := encodeSliceBool(enc, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}
	// append a sentinel value after the packed slice
	enc.WriteInt(42)
	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatalf("skip: %v", err)
	}
	v, err := dec.ReadInt()
	if err != nil {
		t.Fatalf("read after skip: %v", err)
	}
	if v != 42 {
		t.Fatalf("sentinel %d", v)
	}
}

func TestQPackBool_TruncatedPayload(t *testing.T) {
	in := make([]bool, 100)
	for i := range in {
		in[i] = i%3 == 0
	}
	enc := NewEncoder(Fast)
	enc.SetQPack(true)
	if err := encodeSliceBool(enc, unsafe.Pointer(&in)); err != nil {
		t.Fatal(err)
	}
	bad := enc.buf[:len(enc.buf)-3]
	dec := NewDecoderOnBuf(bad)
	var out []bool
	if err := decodeSliceBool(dec, unsafe.Pointer(&out)); err == nil {
		t.Fatalf("expected error on truncated payload")
	}
}

func FuzzQPackBool(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x03, 0x04, 0xFF, 0xAB, 0x55, 0xCC, 0x00, 0xFE})
	f.Add([]byte{})
	f.Add(make([]byte, 256))
	f.Fuzz(func(t *testing.T, data []byte) {
		n := len(data)
		in := make([]bool, n)
		for i, b := range data {
			in[i] = b&1 == 1
		}
		enc := NewEncoder(Fast)
		enc.SetQPack(true)
		if err := encodeSliceBool(enc, unsafe.Pointer(&in)); err != nil {
			t.Fatal(err)
		}
		dec := NewDecoderOnBuf(enc.buf)
		var out []bool
		if err := decodeSliceBool(dec, unsafe.Pointer(&out)); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if n == 0 {
			if len(out) != 0 {
				t.Fatal("empty mismatch")
			}
			return
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatal("mismatch")
		}
	})
}

func makeBoolPattern(n int) []bool {
	s := make([]bool, n)
	for i := range s {
		s[i] = i%3 == 0
	}
	return s
}

func BenchmarkQPackBool_Encode(b *testing.B) {
	sizes := []int{16, 64, 1024, 16384}
	for _, n := range sizes {
		in := makeBoolPattern(n)
		b.Run("legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(n))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceBool(enc, unsafe.Pointer(&in))
			}
		})
		b.Run("qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			enc := NewEncoder(Fast)
			enc.SetQPack(true)
			b.ReportAllocs()
			b.SetBytes(int64(n))
			for b.Loop() {
				enc.Reset()
				_ = encodeSliceBool(enc, unsafe.Pointer(&in))
			}
		})
	}
}

func BenchmarkQPackBool_Decode(b *testing.B) {
	sizes := []int{16, 64, 1024, 16384}
	for _, n := range sizes {
		in := makeBoolPattern(n)
		legacy := NewEncoder(Fast)
		_ = encodeSliceBool(legacy, unsafe.Pointer(&in))
		legacyBuf := append([]byte(nil), legacy.buf...)
		qpack := NewEncoder(Fast)
		qpack.SetQPack(true)
		_ = encodeSliceBool(qpack, unsafe.Pointer(&in))
		qpackBuf := append([]byte(nil), qpack.buf...)
		b.Run("legacy/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n))
			for b.Loop() {
				dec := NewDecoderOnBuf(legacyBuf)
				var out []bool
				_ = decodeSliceBool(dec, unsafe.Pointer(&out))
			}
		})
		b.Run("qpack/n="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n))
			for b.Loop() {
				dec := NewDecoderOnBuf(qpackBuf)
				var out []bool
				_ = decodeSliceBool(dec, unsafe.Pointer(&out))
			}
		})
	}
}
