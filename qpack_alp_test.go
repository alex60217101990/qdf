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

// alpTagOf marshals a struct holding one []float64 field under OptCompression
// and returns the codec tag the encoder picked for that slice. The corpus
// wraps the slice in a struct so we exercise the real encode path.
type alpProbe struct {
	V []float64 `qdf:"v"`
}

func TestALPPickerChoosesByData(t *testing.T) {
	quant := alpFixtureQuantized()
	smooth := alpFixtureSmooth()

	// Inspect the codec choice on the raw wire, so exclude the rANS entropy
	// post-pass (it would wrap the body and hide the inner tag).
	codecOpts := OptCompression &^ OptRANS
	encQuant, err := Marshal(alpProbe{V: quant}, codecOpts)
	if err != nil {
		t.Fatal(err)
	}
	encSmooth, err := Marshal(alpProbe{V: smooth}, codecOpts)
	if err != nil {
		t.Fatal(err)
	}

	if !containsTag(encQuant, tagPackALP) {
		t.Errorf("quantized data: expected ALP tag on wire, not found")
	}
	if containsTag(encSmooth, tagPackALP) {
		t.Errorf("pure-smooth data: ALP should lose to Gorilla/raw, but ALP tag present")
	}

	// Round-trips regardless of codec.
	for name, enc := range map[string][]byte{"quant": encQuant, "smooth": encSmooth} {
		var got alpProbe
		if err := Unmarshal(enc, &got); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
	}
	var gotQ alpProbe
	_ = Unmarshal(encQuant, &gotQ)
	for i := range quant {
		if math.Float64bits(gotQ.V[i]) != math.Float64bits(quant[i]) {
			t.Fatalf("quant round-trip mismatch at %d", i)
		}
	}
}

// containsTag reports whether b carries a packed-float64 slice payload led by
// the given codec tag. A bare byte-scan is not reliable: a raw-LE float64
// payload contains arbitrary data bytes that frequently collide with the tag
// values (0xF4 / 0xE7 / 0xE4), so a single-byte match yields false positives.
// Every packed-float64 codec frames its payload as the two-byte structural
// header {tag, qpackKindFloat64}; scanning for that pair distinguishes a real
// codec choice from a coincidental data byte.
func containsTag(b []byte, tag byte) bool {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == tag && b[i+1] == qpackKindFloat64 {
			return true
		}
	}
	return false
}

// alpFixtureWithExceptions returns a 1024-element quantized slice with ~5%
// non-representable values (NaN, ±Inf, -0.0) so BenchmarkWritePackedALPExc
// exercises the exception-collection and exception-emit paths.
func alpFixtureWithExceptions() []float64 {
	s := alpFixtureQuantized()
	// Scatter exceptions at fixed positions so the benchmark is deterministic.
	for i := 0; i < len(s); i += 20 {
		switch i % 60 {
		case 0:
			s[i] = math.NaN()
		case 20:
			s[i] = math.Inf(1)
		case 40:
			s[i] = math.Copysign(0, -1)
		}
	}
	return s
}

// BenchmarkWritePackedALPExc measures the ALP float64 encode path on data
// that contains exceptions (~5% non-representable values). This is the hot
// path Task 9 optimises: the pre-Task-9 baseline re-scanned all n elements a
// second time to locate exceptions; head collects them during the first pass.
func BenchmarkWritePackedALPExc(b *testing.B) {
	s := alpFixtureWithExceptions()
	plan, _, ok := alpPlanFloat64(s)
	if !ok {
		b.Skip("ALP not applicable for fixture")
	}
	e := NewEncoderWith(OptCompression)
	b.ReportAllocs()
	for b.Loop() {
		e.buf = e.buf[:0]
		e.writePackedALPFloat64Slice(s, plan)
	}
}

func TestALPSkip(t *testing.T) {
	s := alpFixtureQuantized()
	e := NewEncoderWith(OptCompression)
	plan, _, ok := alpPlanFloat64(s)
	if !ok {
		t.Skip("ALP not applicable for fixture")
	}
	e.writePackedALPFloat64Slice(s, plan)
	payload := e.buf[5:] // strip header; payload[0] == tagPackALP
	d := &Decoder{buf: payload}
	d.MarkHeaderRead() // payload begins at the tag, not the 5-byte QDF header
	if err := d.Skip(); err != nil {
		t.Fatalf("Skip over ALP payload: %v", err)
	}
	if d.i != len(payload) {
		t.Fatalf("Skip left cursor at %d, want %d (full payload consumed)", d.i, len(payload))
	}
}
