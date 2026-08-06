package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// bf16Bits / fp16Bits mirror the standard fp32 truncations. Weight tensors in
// these formats are the byte-plane codec's target class.
func bf16Bits(f float32) uint16 {
	u := math.Float32bits(f)
	if u&0x7fffffff > 0x7f800000 { // NaN: keep it quiet
		return uint16(u>>16) | 0x0040
	}
	return uint16((u + 0x7fff + ((u >> 16) & 1)) >> 16)
}

func fp16Bits(f float32) uint16 {
	u := math.Float32bits(f)
	sign := uint16((u >> 16) & 0x8000)
	exp := int32((u>>23)&0xff) - 127 + 15
	man := u & 0x7fffff
	if exp <= 0 {
		if exp < -10 {
			return sign
		}
		man |= 0x800000
		return sign | uint16(man>>uint32(14-exp))
	}
	if exp >= 0x1f {
		return sign | 0x7c00
	}
	return sign | uint16(exp)<<10 | uint16(man>>13)
}

func mkWeights16(n int, seed int64, conv func(float32) uint16) []uint16 {
	rng := rand.New(rand.NewSource(seed))
	s := make([]uint16, n)
	for i := range s {
		s[i] = conv(float32(rng.NormFloat64() * 0.02))
	}
	return s
}

func TestPlane16RoundTrip(t *testing.T) {
	type row struct{ V []uint16 }
	rng := rand.New(rand.NewSource(4))
	randU := make([]uint16, 4096)
	for i := range randU {
		randU[i] = uint16(rng.Uint32())
	}
	hiConst := make([]uint16, 4096) // high byte constant, low random: best case
	for i := range hiConst {
		hiConst[i] = 0x3F00 | uint16(rng.Intn(256))
	}
	loConst := make([]uint16, 4096) // low byte constant, high random: worst case
	for i := range loConst {
		loConst[i] = uint16(rng.Intn(256))<<8 | 0x7B
	}
	cases := map[string][]uint16{
		"bf16":      mkWeights16(8192, 11, bf16Bits),
		"fp16":      mkWeights16(8192, 11, fp16Bits),
		"random":    randU,
		"hi_const":  hiConst,
		"lo_const":  loConst,
		"threshold": mkWeights16(plane16MinElems, 3, bf16Bits),
		"below_thr": mkWeights16(plane16MinElems-1, 3, bf16Bits),
		"allzero":   make([]uint16, 4096),
	}
	for name, in := range cases {
		for _, opts := range []Options{OptBalanced, OptCompression, OptQPack} {
			blob, err := Marshal(row{in}, opts)
			if err != nil {
				t.Fatalf("%s/%v encode: %v", name, opts, err)
			}
			var out row
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatalf("%s/%v decode: %v", name, opts, err)
			}
			if !reflect.DeepEqual(in, out.V) {
				t.Fatalf("%s/%v round-trip mismatch", name, opts)
			}
			if len(blob) > 2*len(in)+64 {
				t.Fatalf("%s/%v grew: %d for raw %d", name, opts, len(blob), 2*len(in))
			}
		}
	}
}

// TestPlane16Wins pins the reason the codec exists: bf16/fp16 weight tensors
// must land on the plane codec and beat the native raw column.
func TestPlane16Wins(t *testing.T) {
	type row struct{ V []uint16 }
	for _, tc := range []struct {
		name    string
		data    []uint16
		minGain float64
	}{
		{"bf16", mkWeights16(65536, 7, bf16Bits), 1.40},
		{"fp16", mkWeights16(65536, 7, fp16Bits), 1.12},
	} {
		blob, err := Marshal(row{tc.data}, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		gain := float64(2*len(tc.data)) / float64(len(blob))
		if gain < tc.minGain {
			t.Fatalf("%s: gain %.3fx below the %.2fx this codec exists for", tc.name, gain, tc.minGain)
		}
		t.Logf("%s: %d bytes for %d raw (%.3fx)", tc.name, len(blob), 2*len(tc.data), gain)
	}
}

// TestPlane16Hostile: corrupted plane bodies must error, never panic.
func TestPlane16Hostile(t *testing.T) {
	type row struct{ V []uint16 }
	blob, err := Marshal(row{mkWeights16(4096, 5, bf16Bits)}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var probe row
	if err := Unmarshal(blob, &probe); err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(17))
	for range 30000 {
		b := append([]byte(nil), blob...)
		b[rng.Intn(len(b))] ^= byte(1 + rng.Intn(255))
		var out row
		_ = Unmarshal(b, &out) // must not panic
	}
	for cut := 0; cut < len(blob); cut += 7 {
		var out row
		_ = Unmarshal(blob[:cut], &out)
	}
}

// TestPlane16Skip: an unknown plane-coded field must Skip to the right offset.
func TestPlane16Skip(t *testing.T) {
	type full struct {
		V    []uint16
		Tail int64
	}
	type slim struct{ Tail int64 }
	for _, opts := range []Options{OptBalanced, OptCompression} {
		blob, err := Marshal(full{mkWeights16(8192, 9, bf16Bits), 1234}, opts)
		if err != nil {
			t.Fatal(err)
		}
		var out slim
		if err := Unmarshal(blob, &out); err != nil {
			t.Fatalf("skip %v: %v", opts, err)
		}
		if out.Tail != 1234 {
			t.Fatalf("tail=%d", out.Tail)
		}
	}
}

// TestPlane16BeatsNarrowRange pins the projection gate: all-positive weight
// tensors fit a narrow value range, so FOR posts a modest win and the old
// "only try when the integer codecs decline" gate skipped the plane form
// entirely — leaving ~11% on the table.
func TestPlane16BeatsNarrowRange(t *testing.T) {
	const n = 65536
	rng := rand.New(rand.NewSource(7))
	pos := make([]uint16, n)
	for i := range pos {
		v := float32(rng.NormFloat64() * 0.02)
		if v < 0 {
			v = -v
		}
		pos[i] = bf16Bits(v)
	}
	type row struct{ V []uint16 }
	blob, err := Marshal(row{pos}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out row
	if err := Unmarshal(blob, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pos, out.V) {
		t.Fatal("round-trip mismatch")
	}
	gain := float64(2*n) / float64(len(blob))
	if gain < 1.55 { // FOR alone gives 1.454x here
		t.Fatalf("gain %.3fx: the plane codec was not reached", gain)
	}
	t.Logf("all-positive bf16: %d bytes (%.3fx)", len(blob), gain)
}

// TestPlane16AnyTyped pins the any-typed path: the 16-bit fast paths made
// []uint16/[]int16 reach pack tags for the first time, which decodeAnyPackedSlice
// did not know about — an any-typed field silently started failing with
// ErrBadTag until the kinds were added.
func TestPlane16AnyTyped(t *testing.T) {
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression} {
		in := map[string]any{
			"small":  []uint16{1, 2, 3, 40000},
			"signed": []int16{-1, 0, 32767, -32768},
			"plane":  mkWeights16(4096, 2, bf16Bits),
		}
		blob, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("%v encode: %v", opts, err)
		}
		var out map[string]any
		if err := Unmarshal(blob, &out); err != nil {
			t.Fatalf("%v any decode: %v", opts, err)
		}
		if got, ok := out["plane"].([]uint16); ok {
			if !reflect.DeepEqual(got, in["plane"]) {
				t.Fatalf("%v plane column mismatch", opts)
			}
		} else if opts != OptSpeed { // OptSpeed keeps the plain-array form
			t.Fatalf("%v plane column decoded as %T", opts, out["plane"])
		}
	}
}

// TestPlane16RejectsRawHiPlane pins the wire invariant: the encoder always
// compresses the high plane strictly below n bytes, so a body claiming
// hiLen >= n is malformed and must error rather than decode as a raw plane
// (which would let a hostile stream supply arbitrary high bytes silently).
func TestPlane16RejectsRawHiPlane(t *testing.T) {
	type row struct{ V []uint16 }
	blob, err := Marshal(row{mkWeights16(4096, 5, bf16Bits)}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Locate the plane field and rewrite hiLen to exactly n.
	for i := 0; i+1 < len(blob); i++ {
		if blob[i] != tagPackRaw || blob[i+1] != qpackKindPlane16 {
			continue
		}
		d := NewDecoderOnBuf(blob[i:])
		d.MarkHeaderRead()
		d.i++
		if _, err := d.readPackedPlane16Slice(); err != nil {
			continue // false positive inside a tANS blob
		}
		n64, nr := readUvarint(blob[i+2:])
		hiOff := i + 2 + nr
		_, hnr := readUvarint(blob[hiOff:])
		if hnr != uvarintLen(n64) {
			t.Skip("hiLen varint width differs from n's; splice not applicable")
		}
		bad := append([]byte(nil), blob...)
		_ = appendUvarint(bad[hiOff:hiOff], n64)
		bd := NewDecoderOnBuf(bad[i:])
		bd.MarkHeaderRead()
		bd.i++
		if _, err := bd.readPackedPlane16Slice(); err == nil {
			t.Fatal("hiLen == n accepted; a hostile raw high plane decodes silently")
		}
		return
	}
	t.Skip("no plane field located")
}
