package qdf

import (
	"math"
	"math/bits"
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

// TestALPExactReconstruction pins the reason the exact (division) form exists:
// multiplying the integer mantissa by a precomputed 1/10^d is an
// approximation, so perfectly representable decimals failed the round-trip
// check and were stored raw. A 2-decimal column measured 1087 spurious
// exceptions out of 8192; the exact form must have none.
func TestALPExactReconstruction(t *testing.T) {
	const n = 8192
	rng := rand.New(rand.NewSource(9))
	for _, beta := range []int{1, 2, 3, 4} {
		data := make([]float64, n)
		p := math.Pow(10, float64(beta))
		v := 42.5
		for i := range data {
			v += float64(rng.Intn(21)-10) / p
			data[i] = math.Round(v*p) / p
		}
		plan, _, ok := alpPlanFloat64(data)
		if !ok {
			t.Fatalf("beta=%d: ALP declined a pure decimal column", beta)
		}
		// -0.0 and non-finites are exceptions BY DESIGN (the integer mantissa
		// cannot carry the sign of zero), so the exact form's promise is that
		// nothing else is.
		special := 0
		for _, x := range data {
			if math.Signbit(x) && x == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
				special++
			}
		}
		if plan.exc != special {
			t.Fatalf("beta=%d: %d exceptions, only %d are -0.0/non-finite — the rest are spurious",
				beta, plan.exc, special)
		}
		enc := NewEncoder(Fast)
		enc.writePackedALPFloat64Slice(data, plan)
		dec := NewDecoderOnBuf(enc.buf)
		if _, err := dec.peekTag(); err != nil {
			t.Fatal(err)
		}
		dec.i++
		got, err := dec.readPackedALPFloat64Slice()
		if err != nil {
			t.Fatalf("beta=%d: %v", beta, err)
		}
		for i := range data {
			if math.Float64bits(got[i]) != math.Float64bits(data[i]) {
				t.Fatalf("beta=%d: value %d not bit-exact", beta, i)
			}
		}
	}
}

// TestALPExactFlagIsLoadBearing: decoding an exact-form body through the
// legacy multiply path must produce DIFFERENT values — which is why a decoder
// that predates the flag has to reject the blob (it validates the exponent
// byte against alpMaxExp and returns ErrBadTag) rather than guess.
func TestALPExactFlagIsLoadBearing(t *testing.T) {
	const n = 4096
	rng := rand.New(rand.NewSource(3))
	data := make([]float64, n)
	v := 42.5
	for i := range data {
		v += float64(rng.Intn(21)-10) / 100
		data[i] = math.Round(v*100) / 100
	}
	plan, _, ok := alpPlanFloat64(data)
	if !ok {
		t.Skip("ALP declined")
	}
	enc := NewEncoder(Fast)
	enc.writePackedALPFloat64Slice(data, plan)
	blob := append([]byte(nil), enc.buf...)
	stripped := append([]byte(nil), blob...)
	found := false
	for i := 0; i+1 < len(stripped); i++ {
		if stripped[i] == tagPackALP && stripped[i+1] == qpackKindFloat64 {
			_, nr := readUvarint(stripped[i+2:])
			if stripped[i+2+nr]&alpExactFlag == 0 {
				continue
			}
			stripped[i+2+nr] &^= alpExactFlag
			found = true
			break
		}
	}
	if !found {
		t.Skip("no flagged ALP block on the wire")
	}
	d := NewDecoderOnBuf(stripped)
	if _, err := d.peekTag(); err != nil {
		t.Fatal(err)
	}
	d.i++
	old, err := d.readPackedALPFloat64Slice()
	if err != nil {
		t.Fatalf("legacy path errored: %v", err)
	}
	diff := 0
	for i := range data {
		if math.Float64bits(old[i]) != math.Float64bits(data[i]) {
			diff++
		}
	}
	if diff == 0 {
		t.Fatal("legacy and exact paths agree; the flag would be pointless")
	}
}

// TestALPChoosesCheaperReconstruction pins the invariant that the whole
// exact/multiply choice rests on: the block must take whichever reconstruction
// costs FEWER BYTES, counting both the FOR width its accepted set implies and
// its exceptions. Scoring only the exception counts produced three separate
// regressions — a one-exception ratio cliff flipping a 40% size difference,
// short blocks taking the newer wire form for zero gain, and a case where the
// values only division accepts widened FOR from 12 to 50 bits and the "better"
// form came out 2.17x larger.
func TestALPChoosesCheaperReconstruction(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	mkNoisy := func(n, pct int) []float64 {
		s := make([]float64, n)
		v := 100.0
		for i := range s {
			v += (rng.Float64() - 0.5) * 2
			if rng.Intn(100) < pct {
				s[i] = v + rng.NormFloat64()*1e-7
			} else {
				s[i] = math.Round(v*100) / 100
			}
		}
		return s
	}
	// Values only division accepts, planted so they widen the FOR range hard.
	planted := make([]float64, 5000)
	for i := range planted {
		planted[i] = math.Round(rng.Float64()*4095) / 100
	}
	for i := range 12 {
		planted[i*400] = float64(int64(1)<<49) / 100
	}
	corpus := map[string][]float64{
		"noisy_5":  mkNoisy(3000, 5),
		"noisy_10": mkNoisy(3000, 10),
		"noisy_25": mkNoisy(3000, 25),
		"planted":  planted,
		"pure_2dp": mkNoisy(3000, 0),
		"short_32": mkNoisy(32, 0),
		"sensor":   mkSensor(2048, 7),
		"allsame":  make([]float64, 512),
	}
	const excBytes = 13
	for name, data := range corpus {
		plan, _, ok := alpPlanFloat64(data)
		if !ok {
			continue
		}
		// Re-derive both candidates at the chosen exponent and confirm the plan
		// took the cheaper one, with ties going to multiply (older wire, faster
		// decode).
		pe, ie := alpPow10[plan.d], alpInv10[plan.d]
		var mnD, mxD, mnM, mxM int64 = math.MaxInt64, math.MinInt64, math.MaxInt64, math.MinInt64
		excD, excM := 0, 0
		for _, v := range data {
			iv := int64(math.RoundToEven(v * pe))
			bv := math.Float64bits(v)
			if math.Float64bits(float64(iv)/pe) == bv {
				mnD, mxD = min(mnD, iv), max(mxD, iv)
			} else {
				excD++
			}
			if math.Float64bits(float64(iv)*ie) == bv {
				mnM, mxM = min(mnM, iv), max(mxM, iv)
			} else {
				excM++
			}
		}
		width := func(lo, hi int64) int {
			if lo > hi {
				return 0
			}
			return bits.Len64(uint64(hi - lo))
		}
		costD := (width(mnD, mxD)*len(data)+7)/8 + excD*excBytes
		costM := (width(mnM, mxM)*len(data)+7)/8 + excM*excBytes
		wantExact := costD < costM
		if plan.exact != wantExact {
			t.Fatalf("%s: chose exact=%v but costs are div=%d mul=%d", name, plan.exact, costD, costM)
		}
		if !plan.exact && excM == excD && excM == 0 {
			// Nothing to gain: must stay on the original wire form.
			if plan.exact {
				t.Fatalf("%s: took the newer wire form for zero gain", name)
			}
		}
	}
}

// TestALPFloat32NeverFlagged: the float32 pack loop only ever uses the
// multiply predicate, so its wire must never claim the exact form — a
// mismatch there would have the reader divide what the writer multiplied.
func TestALPFloat32NeverFlagged(t *testing.T) {
	rng := rand.New(rand.NewSource(8))
	for _, n := range []int{64, 1000, 4096} {
		s := make([]float32, n)
		v := float32(12.0)
		for i := range s {
			v += float32(rng.Intn(21)-10) / 100
			s[i] = float32(math.Round(float64(v)*100) / 100)
		}
		plan, _, ok := alpPlanFloat32(s)
		if !ok {
			continue
		}
		if plan.exact {
			t.Fatalf("n=%d: float32 plan claims the exact form", n)
		}
		e := NewEncoder(Fast)
		e.MarkHeaderWritten()
		e.writePackedALPFloat32Slice(s, plan)
		// exponent byte sits after tag, kind, uvarint(n), first value
		_, nr := readUvarint(e.buf[2:])
		if e.buf[2+nr]&alpExactFlag != 0 {
			t.Fatalf("n=%d: float32 wire carries alpExactFlag", n)
		}
		dec := NewDecoderOnBuf(e.buf)
		dec.MarkHeaderRead()
		dec.i++
		got, err := dec.readPackedALPFloat32Slice()
		if err != nil {
			t.Fatal(err)
		}
		for i := range s {
			if math.Float32bits(got[i]) != math.Float32bits(s[i]) {
				t.Fatalf("n=%d: lossy at %d", n, i)
			}
		}
	}
}
