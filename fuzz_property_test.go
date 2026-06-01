package qdf

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

// Property-based round-trip fuzzers. Each one takes raw fuzz bytes,
// interprets them as a deterministic seed for value generation, encodes
// the generated value through Marshal / MarshalQPack / MarshalDense /
// MarshalT / MarshalQPackT / MarshalDenseT, then unmarshals back and
// asserts the result equals the original.
//
// The pattern catches semantic divergences that simple "never panic"
// fuzzing misses. It is the test discipline used by stdlib encoding/json
// and (less formally) by vmihailenco/msgpack: assert an algebraic
// property, not just absence of crashes.

// fuzzReader consumes the fuzz byte stream as a stream of typed pulls.
// Exhaustion wraps back to zero so the generator stays deterministic
// even for very short inputs.
type fuzzReader struct {
	buf []byte
	pos int
}

func newFuzzReader(buf []byte) *fuzzReader { return &fuzzReader{buf: buf} }

func (r *fuzzReader) byteAt(i int) byte {
	if len(r.buf) == 0 {
		return 0
	}
	return r.buf[i%len(r.buf)]
}

func (r *fuzzReader) pull(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = r.byteAt(r.pos + i)
	}
	r.pos += n
	return out
}

func (r *fuzzReader) u8() uint8   { b := r.pull(1); return b[0] }
func (r *fuzzReader) u32() uint32 { return binary.LittleEndian.Uint32(r.pull(4)) }
func (r *fuzzReader) u64() uint64 { return binary.LittleEndian.Uint64(r.pull(8)) }
func (r *fuzzReader) i64() int64  { return int64(r.u64()) }
func (r *fuzzReader) i32() int32  { return int32(r.u32()) }
func (r *fuzzReader) f64() float64 {
	v := math.Float64frombits(r.u64())
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// f64raw returns the float64 bit-for-bit without normalising NaN/Inf/−0.
// Use this in fuzz targets that want to exercise the full IEEE-754 space,
// including the values that f64() previously hid from the codec.
func (r *fuzzReader) f64raw() float64 {
	return math.Float64frombits(r.u64())
}

// boundedLen takes a fuzz-driven byte and clamps it to [0, max].
func (r *fuzzReader) boundedLen(max int) int {
	b := r.u8()
	if max <= 0 {
		return 0
	}
	return int(b) % (max + 1)
}

func (r *fuzzReader) str(maxLen int) string {
	n := r.boundedLen(maxLen)
	out := make([]byte, n)
	for i := range out {
		// Bias towards ASCII to also exercise fixstr-eligible alphabet.
		out[i] = r.u8() & 0x7F
	}
	return string(out)
}

// floatSliceEqual compares two []float64 with fully bit-exact semantics:
// every bit of each element must match, including NaN payloads and the -0 sign.
// This is intentionally stricter than == (which is NaN-unequal and -0==+0).
func floatSliceEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return false
		}
	}
	return true
}

// --- FuzzRoundTrip_Int64Slice ---------------------------------------

func FuzzRoundTrip_Int64Slice(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		n := r.boundedLen(256)
		in := make([]int64, n)
		for i := range in {
			in[i] = r.i64()
		}
		for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("marshal n=%d: %v", n, err)
			}
			var out []int64
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("unmarshal n=%d: %v", n, err)
			}
			if n == 0 {
				if len(out) != 0 {
					t.Fatalf("empty mismatch")
				}
				continue
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("[]int64 mismatch")
			}
		}
	})
}

// --- FuzzRoundTrip_Uint64Slice --------------------------------------

func FuzzRoundTrip_Uint64Slice(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		n := r.boundedLen(256)
		in := make([]uint64, n)
		for i := range in {
			in[i] = r.u64()
		}
		for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out []uint64
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if n == 0 {
				continue
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("[]uint64 mismatch")
			}
		}
	})
}

// --- FuzzRoundTrip_Float64Slice -------------------------------------

func FuzzRoundTrip_Float64Slice(f *testing.F) {
	f.Add([]byte{0})
	f.Add(make([]byte, 64))
	// Seed with bytes that produce NaN, +Inf, -Inf, and -0 so the corpus
	// exercises the previously-blind IEEE-754 edge cases from the start.
	f.Add([]byte{
		// NaN (canonical quiet NaN on amd64/arm64: 0x7FF8000000000000)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF8, 0x7F,
	})
	f.Add([]byte{
		// +Inf (0x7FF0000000000000)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x7F,
	})
	f.Add([]byte{
		// -Inf (0xFFF0000000000000)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0xFF,
	})
	f.Add([]byte{
		// -0 (0x8000000000000000)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80,
	})
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		n := r.boundedLen(256)
		in := make([]float64, n)
		for i := range in {
			// Use f64raw so NaN/Inf/-0 reach the codec — the blind spot fix.
			in[i] = r.f64raw()
		}
		for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced, OptCompression, OptQPack | OptGorillaFloat} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("opts=%d marshal: %v", opts, err)
			}
			var out []float64
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("opts=%d unmarshal: %v", opts, err)
			}
			if n == 0 {
				continue
			}
			if !floatSliceEqual(in, out) {
				t.Fatalf("opts=%d []float64 bits NOT preserved:\n in =%v\n out=%v\n inbits =%x\n outbits=%x",
					opts, in, out, floatBits64(in), floatBits64(out))
			}
		}
	})
}

// --- FuzzRoundTrip_BoolSlice ----------------------------------------

func FuzzRoundTrip_BoolSlice(f *testing.F) {
	f.Add([]byte{0xAA})
	f.Add(make([]byte, 32))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		n := r.boundedLen(1024)
		in := make([]bool, n)
		for i := range in {
			in[i] = r.u8()&1 == 1
		}
		for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			var out []bool
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatal(err)
			}
			if n == 0 {
				continue
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("[]bool mismatch")
			}
		}
	})
}

// --- FuzzRoundTrip_MapStringInt -------------------------------------

func FuzzRoundTrip_MapStringInt(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{3, 5, 'a', 'b', 'c'})
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		n := r.boundedLen(32)
		in := make(map[string]int, n)
		for range n {
			k := r.str(16)
			v := int(r.i32())
			in[k] = v
		}
		for _, opts := range []Options{OptSpeed, OptBalanced} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			out := map[string]int{}
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatal(err)
			}
			if len(in) == 0 {
				continue
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("map mismatch")
			}
		}
	})
}

// --- FuzzRoundTrip_StructTriad --------------------------------------

type fuzzTriad struct {
	ID      int64     `qdf:"id"`
	Name    string    `qdf:"name"`
	Tags    []string  `qdf:"tags"`
	Counts  []int     `qdf:"counts"`
	Vec     []float64 `qdf:"vec"`
	Active  bool      `qdf:"active"`
	Padding []byte    `qdf:"padding"`
}

func FuzzRoundTrip_StructTriad(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	f.Add(make([]byte, 96))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		in := fuzzTriad{
			ID:     r.i64(),
			Name:   r.str(64),
			Active: r.u8()&1 == 1,
		}
		nTags := r.boundedLen(8)
		in.Tags = make([]string, nTags)
		for i := range in.Tags {
			in.Tags[i] = r.str(8)
		}
		nCounts := r.boundedLen(16)
		in.Counts = make([]int, nCounts)
		for i := range in.Counts {
			in.Counts[i] = int(r.i32())
		}
		nVec := r.boundedLen(16)
		in.Vec = make([]float64, nVec)
		for i := range in.Vec {
			in.Vec[i] = r.f64()
		}
		nPad := r.boundedLen(32)
		in.Padding = r.pull(nPad)

		for label, opts := range map[string]Options{
			"Speed":    OptSpeed,
			"QPack":    OptQPack,
			"Balanced": OptBalanced,
		} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("%s: %v", label, err)
			}
			var out fuzzTriad
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("%s decode: %v", label, err)
			}
			if !triadEqual(in, out) {
				t.Fatalf("%s mismatch:\n in=%+v\nout=%+v", label, in, out)
			}
		}
	})
}

func triadEqual(a, b fuzzTriad) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Active != b.Active {
		return false
	}
	if !reflect.DeepEqual(a.Tags, b.Tags) {
		return false
	}
	if !reflect.DeepEqual(a.Counts, b.Counts) {
		return false
	}
	if !floatSliceEqual(a.Vec, b.Vec) {
		return false
	}
	if !bytes.Equal(a.Padding, b.Padding) {
		return false
	}
	return true
}

// --- FuzzRoundTrip_AllModesAgree ------------------------------------
//
// The strongest property: every encoder path decodes back into the same
// value as every other encoder path. If only one path mis-encodes, the
// fuzz catches the divergence rather than a wire-corruption symptom.

func FuzzRoundTrip_AllModesAgree(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)
		in := fuzzTriad{
			ID:   r.i64(),
			Name: r.str(32),
		}
		nTags := r.boundedLen(4)
		in.Tags = make([]string, nTags)
		for i := range in.Tags {
			in.Tags[i] = r.str(8)
		}

		var outFast, outQPack, outDense fuzzTriad
		bufFast, err := Marshal(in, OptSpeed)
		if err != nil {
			t.Fatal(err)
		}
		bufQPack, err := Marshal(in, OptQPack)
		if err != nil {
			t.Fatal(err)
		}
		bufDense, err := Marshal(in, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		if err := Unmarshal(bufFast, &outFast); err != nil {
			t.Fatal(err)
		}
		if err := Unmarshal(bufQPack, &outQPack); err != nil {
			t.Fatal(err)
		}
		if err := Unmarshal(bufDense, &outDense); err != nil {
			t.Fatal(err)
		}
		if !triadEqual(outFast, outQPack) {
			t.Fatalf("Fast vs QPack diverge")
		}
		if !triadEqual(outFast, outDense) {
			t.Fatalf("Fast vs Dense diverge")
		}
	})
}

// --- FuzzRoundTrip_Compression -----------------------------------------
//
// Mirrors FuzzRoundTrip_Int64Slice but drives generated int/uint/bool/string
// values through the heavy codec bundles (OptCompression and OptDense|OptRANS)
// that were previously not covered by the roundtrip fuzz corpus. A bug in the
// RANS decode path for non-string payloads would surface here.

type fuzzCompInput struct {
	Ints  []int64  `qdf:"ints"`
	Uints []uint64 `qdf:"uints"`
	Bools []bool   `qdf:"bools"`
	Strs  []string `qdf:"strs"`
}

func FuzzRoundTrip_Compression(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(make([]byte, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzReader(data)

		nInts := r.boundedLen(32)
		in := fuzzCompInput{}
		in.Ints = make([]int64, nInts)
		for i := range in.Ints {
			in.Ints[i] = r.i64()
		}
		nUints := r.boundedLen(32)
		in.Uints = make([]uint64, nUints)
		for i := range in.Uints {
			in.Uints[i] = r.u64()
		}
		nBools := r.boundedLen(64)
		in.Bools = make([]bool, nBools)
		for i := range in.Bools {
			in.Bools[i] = r.u8()&1 == 1
		}
		nStrs := r.boundedLen(16)
		in.Strs = make([]string, nStrs)
		for i := range in.Strs {
			in.Strs[i] = r.str(16)
		}

		for _, opts := range []Options{OptCompression, OptDense | OptRANS} {
			buf, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("opts=%d marshal: %v", opts, err)
			}
			var out fuzzCompInput
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("opts=%d unmarshal: %v", opts, err)
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("opts=%d mismatch:\n in=%+v\nout=%+v", opts, in, out)
			}
		}
	})
}
