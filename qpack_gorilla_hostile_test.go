package qdf

import (
	"math"
	"testing"
)

// TestGorillaHostileWindowRejected pins the fix for the Gorilla new-window
// underflow: a malformed bitstream whose leading-zeros + meaningful-bits exceed
// the float word width (lz+mb > 64 / > 32) must be rejected, not silently
// decoded to a wrong value (the unsigned `width - lz - mb` underflowed, the
// shift collapsed to 0, and prevTZ was poisoned for the rest of the slice).
func TestGorillaHostileWindowRejected(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		var buf []byte
		buf = append(buf, qpackKindFloat64) // tag already consumed upstream
		buf = appendUvarint(buf, 2)         // n = 2
		buf = appendU64(buf, math.Float64bits(1.0))
		bw := bitWriter{}
		bw.writeBit(true)        // ctl: this element differs
		bw.writeBit(true)        // ctl2: new window
		bw.writeBits(31, 5)      // lz64 = 31
		bw.writeBits(63, 6)      // mbLen64 = 63 -> mb = 64; lz+mb = 95 > 64
		bw.writeBits(0xABCD, 64) // meaningful-bits payload
		total := bw.flush()
		buf = appendUvarint(buf, uint64(total))
		buf = append(buf, bw.buf...)

		d := &Decoder{buf: buf}
		if _, err := d.readPackedGorillaFloat64Slice(); err == nil {
			t.Fatal("hostile lz+mb>64 window accepted (silent wrong decode); want error")
		}
	})
	t.Run("float32", func(t *testing.T) {
		var buf []byte
		buf = append(buf, qpackKindFloat32)
		buf = appendUvarint(buf, 2)
		buf = appendU64(buf, uint64(math.Float32bits(1.0))) // header reads w=4 bytes via readU32
		bw := bitWriter{}
		bw.writeBit(true)        // differs
		bw.writeBit(true)        // new window
		bw.writeBits(15, 4)      // lz64 = 15
		bw.writeBits(31, 5)      // mbLen64 = 31 -> mb = 32; lz+mb = 47 > 32
		bw.writeBits(0xABCD, 32) // payload
		total := bw.flush()
		buf = appendUvarint(buf, uint64(total))
		buf = append(buf, bw.buf...)

		d := &Decoder{buf: buf}
		if _, err := d.readPackedGorillaFloat32Slice(); err == nil {
			t.Fatal("hostile lz+mb>32 window accepted (silent wrong decode); want error")
		}
	})
}

// TestGorillaValidStillRoundTrips guards that the new window-width check does
// not reject any legitimately-encoded Gorilla stream (a valid window always has
// lz + mb <= word width).
func TestGorillaValidStillRoundTrips(t *testing.T) {
	type series struct {
		F64 []float64 `qdf:"f64"`
		F32 []float32 `qdf:"f32"`
	}
	in := series{
		F64: make([]float64, 256),
		F32: make([]float32, 256),
	}
	v := 1000.0
	for i := range in.F64 {
		v += 0.5 + float64(i%7)*0.01 // smooth-ish -> Gorilla XOR windows
		in.F64[i] = v
		in.F32[i] = float32(v)
	}
	buf, err := Marshal(in, OptCompression) // Gorilla fires under OptCompression
	if err != nil {
		t.Fatal(err)
	}
	var out series
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	for i := range in.F64 {
		if math.Float64bits(out.F64[i]) != math.Float64bits(in.F64[i]) {
			t.Fatalf("f64[%d] = %v, want %v", i, out.F64[i], in.F64[i])
		}
		if math.Float32bits(out.F32[i]) != math.Float32bits(in.F32[i]) {
			t.Fatalf("f32[%d] = %v, want %v", i, out.F32[i], in.F32[i])
		}
	}
}
