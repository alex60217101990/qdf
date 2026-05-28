package qdf

import (
	"math"
	"math/rand"
	"testing"
)

func alpFixtureQuantized() []float64 {
	r := rand.New(rand.NewSource(1))
	s := make([]float64, 1024)
	for i := range s {
		s[i] = float64(r.Intn(10_000_000)) / 100.0 // 2-decimal, 0..100000.00
	}
	return s
}

func alpFixtureSmooth() []float64 {
	s := make([]float64, 1024)
	for i := range s {
		s[i] = 100 + 50*math.Sin(float64(i)*0.01)
	}
	return s
}

func alpFixtureSmoothQuantized() []float64 {
	s := make([]float64, 1024)
	for i := range s {
		v := 100 + 50*math.Sin(float64(i)*0.01)
		s[i] = math.Round(v*100) / 100
	}
	return s
}

// alpRoundTrip encodes via the production Encoder helper and decodes via the
// production Decoder helper, asserting bit-exact reconstruction.
func alpRoundTrip(t *testing.T, name string, s []float64) {
	t.Helper()
	e := NewEncoderWith(OptCompression)
	plan, _, ok := alpPlanFloat64(s)
	if !ok && len(s) > 0 {
		// All-exception or width>56: ALP not applicable; skip — the picker
		// would never choose it here.
		return
	}
	e.writePackedALPFloat64Slice(s, plan)
	payload := e.buf[5:] // strip the 5-byte QDF header
	d := &Decoder{buf: payload}
	d.i = 1 // payload[0] is tagPackALP; land cursor on kind
	got, err := d.readPackedALPFloat64Slice()
	if err != nil {
		t.Fatalf("%s: decode error: %v", name, err)
	}
	if len(got) != len(s) {
		t.Fatalf("%s: len mismatch got %d want %d", name, len(got), len(s))
	}
	for i := range s {
		if math.Float64bits(got[i]) != math.Float64bits(s[i]) {
			t.Fatalf("%s: value %d mismatch got %v (%#x) want %v (%#x)",
				name, i, got[i], math.Float64bits(got[i]), s[i], math.Float64bits(s[i]))
		}
	}
}

func TestALPCodecRoundTrip(t *testing.T) {
	cases := map[string][]float64{
		"quantized":       alpFixtureQuantized(),
		"smooth":          alpFixtureSmooth(),
		"smoothQuantized": alpFixtureSmoothQuantized(),
		"empty":           {},
		"single":          {42.42},
		"allEqual":        {2.5, 2.5, 2.5, 2.5, 2.5, 2.5, 2.5, 2.5, 2.5, 2.5},
		"withNaNInf":      {1.5, math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1), 3.14},
	}
	for name, s := range cases {
		alpRoundTrip(t, name, s)
	}
}

// TestALPDecodeHostileCount feeds a hand-crafted payload whose n header is far
// larger than the buffer can back. With width==0 there are no per-element body
// bytes, so the body-size check never bounds n; the alpMaxElems guard must
// reject it with an error rather than reaching make([]float64, n) and panicking.
// Byte layout after the tag, exactly as readPackedALPFloat64Slice reads it:
// kind, n (varuint), d (byte), forMin (zigzag-varuint), width (byte), excN
// (varuint). width==0 means no body.
func TestALPDecodeHostileCount(t *testing.T) {
	var payload []byte
	payload = append(payload, qpackKindFloat64)         // kind
	payload = appendUvarint(payload, 1<<40)             // n: oversized count
	payload = append(payload, 2)                        // d: effective exponent
	payload = appendUvarint(payload, zigzagEncode64(0)) // forMin
	payload = append(payload, 0)                        // width==0 (constant slice, no body)
	payload = appendUvarint(payload, 0)                 // excN

	d := &Decoder{buf: payload}
	got, err := d.readPackedALPFloat64Slice()
	if err == nil {
		t.Fatalf("expected error for hostile n, got %d values", len(got))
	}
}

// TestALPEncoderDeclinesHugeConstant verifies the encode-side cap: the picker
// must decline ALP for slices beyond alpMaxElems so it never emits a payload
// the decoder would reject.
func TestALPEncoderDeclinesHugeConstant(t *testing.T) {
	_, _, ok := alpPlanFloat64(make([]float64, alpMaxElems+1))
	if ok {
		t.Fatal("alpPlanFloat64 should decline a slice larger than alpMaxElems")
	}
}
