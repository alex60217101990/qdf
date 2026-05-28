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
