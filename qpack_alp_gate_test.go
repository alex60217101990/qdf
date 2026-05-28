package qdf

// Isolated feasibility GATE for an ALP (Adaptive Lossless floating-Point,
// CWI 2023) decimal-path float codec, benchmarked against qdf's existing
// Gorilla XOR codec (qpack_gorilla.go). This file is a PROTOTYPE in package
// qdf so it can call the unexported Gorilla encode/decode directly; it does
// NOT touch production code and is gate-only — nothing here is wired into
// the codec dispatch.
//
// Algorithm simplification note: the prototype searches the full (e, f)
// exponent pair as described in the ALP paper (e in 0..15, f in 0..e),
// encoding I = round(v * 10^(e-f)) and decoding v' = I * 10^-(e-f). f only
// ever shifts which power of ten is used, so for the decimal path it is
// equivalent to searching a single effective exponent d = e-f in 0..15; the
// pair loop is kept to stay faithful to the paper's shape. Round-trip uses
// exact float64 equality, with an exception list for any value that does not
// reconstruct bit-for-bit.

import (
	"math"
	"math/bits"
	"math/rand"
	"testing"
)

// ---- ALP power-of-ten tables (float64 + integer) ----

const alpMaxExp = 18

var (
	alpPow10 [alpMaxExp + 1]float64
	alpInv10 [alpMaxExp + 1]float64
)

func init() {
	p := 1.0
	for i := 0; i <= alpMaxExp; i++ {
		alpPow10[i] = p
		alpInv10[i] = 1.0 / p
		p *= 10
	}
}

// ---- tiny LSB-first bit packer (local to the prototype) ----

type alpBitWriter struct {
	buf  []byte
	cur  uint64
	used uint8 // bits filled in cur (0..63)
}

func (w *alpBitWriter) writeBits(v uint64, count uint8) {
	if count < 64 {
		v &= (uint64(1) << count) - 1
	}
	w.cur |= v << w.used
	w.used += count
	for w.used >= 8 {
		w.buf = append(w.buf, byte(w.cur))
		w.cur >>= 8
		w.used -= 8
	}
}

func (w *alpBitWriter) flush() {
	if w.used > 0 {
		w.buf = append(w.buf, byte(w.cur))
		w.cur = 0
		w.used = 0
	}
}

type alpBitReader struct {
	buf  []byte
	pos  int
	cur  uint64
	used uint8 // valid bits currently in cur
}

func (r *alpBitReader) readBits(count uint8) uint64 {
	for r.used < count {
		var b byte
		if r.pos < len(r.buf) {
			b = r.buf[r.pos]
		}
		r.pos++
		r.cur |= uint64(b) << r.used
		r.used += 8
	}
	var mask uint64 = ^uint64(0)
	if count < 64 {
		mask = (uint64(1) << count) - 1
	}
	out := r.cur & mask
	r.cur >>= count
	r.used -= count
	return out
}

// ---- ALP encode / decode ----

// alpTryExp evaluates a single effective exponent d = e-f over the whole
// block: it returns the integer encodings, the FOR min, the bit width, the
// number of exceptions, and an estimated encoded byte cost. round-trip is
// checked with exact float64 equality.
func alpTryExp(s []float64, d int) (forMin int64, bw uint8, exc int, estBytes int) {
	n := len(s)
	pe := alpPow10[d]
	ie := alpInv10[d]
	var mn, mx int64
	mn = math.MaxInt64
	mx = math.MinInt64
	exc = 0
	for _, v := range s {
		I := int64(math.RoundToEven(v * pe))
		if float64(I)*ie != v {
			exc++
			continue
		}
		if I < mn {
			mn = I
		}
		if I > mx {
			mx = I
		}
	}
	if mn > mx { // all exceptions
		mn, mx = 0, 0
	}
	width := uint8(bits.Len64(uint64(mx - mn)))
	estBytes = (int(width)*n+7)/8 + exc*(8+2) + 16
	return mn, width, exc, estBytes
}

// alpChooseExp samples ~32 evenly-spaced values to pick the best effective
// exponent d in 0..15, then re-scores the winner over the full block.
func alpChooseExp(s []float64) int {
	n := len(s)
	// Build a sample.
	const sampleN = 32
	var sample []float64
	if n <= sampleN {
		sample = s
	} else {
		sample = make([]float64, 0, sampleN)
		step := n / sampleN
		if step < 1 {
			step = 1
		}
		for i := 0; i < n; i += step {
			sample = append(sample, s[i])
		}
	}
	bestD := 0
	bestCost := math.MaxInt64
	for d := 0; d <= 15; d++ {
		_, _, _, est := alpTryExp(sample, d)
		// Scale estimate up to full block proportionally for fairness.
		if est < bestCost {
			bestCost = est
			bestD = d
		}
	}
	return bestD
}

// alpEncode encodes s with the ALP decimal path.
//
// Layout (all little-endian / LSB-first):
//
//	uvarint n
//	if n == 0: end
//	1 byte   d (effective exponent e-f)
//	8 byte   forMin (int64)
//	1 byte   bit width
//	bit-packed I_i - forMin, width bits each, n values (byte-aligned after)
//	uvarint  exceptionCount
//	per exception: uint16 position (LE) + float64 raw (LE)
func alpEncode(s []float64) []byte {
	n := len(s)
	out := make([]byte, 0, n) // rough
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		return out
	}
	d := alpChooseExp(s)
	forMin, width, _, _ := alpTryExp(s, d)

	out = append(out, byte(d))
	out = appendU64(out, uint64(forMin))
	out = append(out, byte(width))

	pe := alpPow10[d]
	ie := alpInv10[d]

	bw := alpBitWriter{}
	type exc struct {
		pos uint16
		raw uint64
	}
	var excs []exc
	for i, v := range s {
		I := int64(math.RoundToEven(v * pe))
		if float64(I)*ie != v {
			// Exception: store a placeholder (0) in the packed stream
			// and record the raw bits.
			if width > 0 {
				bw.writeBits(0, width)
			}
			excs = append(excs, exc{pos: uint16(i), raw: math.Float64bits(v)})
			continue
		}
		if width > 0 {
			bw.writeBits(uint64(I-forMin), width)
		}
	}
	bw.flush()
	out = append(out, bw.buf...)

	out = appendUvarint(out, uint64(len(excs)))
	for _, e := range excs {
		out = appendU16(out, e.pos)
		out = appendU64(out, e.raw)
	}
	return out
}

// alpDecode reverses alpEncode. Lossless: reproduces input bit-for-bit.
func alpDecode(b []byte) []float64 {
	n64, nr := readUvarint(b)
	pos := nr
	n := int(n64)
	out := make([]float64, n)
	if n == 0 {
		return out
	}
	d := int(b[pos])
	pos++
	forMin := int64(readU64(b[pos:]))
	pos += 8
	width := b[pos]
	pos++

	ie := alpInv10[d]

	// Bit-packed body occupies ceil(width*n/8) bytes.
	bodyBytes := (int(width)*n + 7) / 8
	br := alpBitReader{buf: b[pos : pos+bodyBytes]}
	pos += bodyBytes
	for i := 0; i < n; i++ {
		var I int64
		if width > 0 {
			I = int64(br.readBits(width)) + forMin
		} else {
			I = forMin
		}
		out[i] = float64(I) * ie
	}

	excCount64, nr2 := readUvarint(b[pos:])
	pos += nr2
	for k := 0; k < int(excCount64); k++ {
		p := readU16(b[pos:])
		pos += 2
		raw := readU64(b[pos:])
		pos += 8
		out[p] = math.Float64frombits(raw)
	}
	return out
}

// ---- fixtures ----

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

// ---- Gorilla helpers (measure on a bare []float64) ----

// gorillaEncode returns just the Gorilla payload (tag + kind + body),
// stripped of the 5-byte QDF header, and its length.
func gorillaEncode(s []float64) []byte {
	e := NewEncoderWith(OptSpeed)
	e.writePackedGorillaFloat64Slice(s)
	// Strip the 5-byte QDF header (Magic0,Magic1,Magic2,Version1,flag).
	return e.buf[5:]
}

// gorillaDecode decodes a payload produced by gorillaEncode.
func gorillaDecode(payload []byte) ([]float64, error) {
	d := &Decoder{buf: payload}
	// payload[0] is tagPackGorilla; skip it so the cursor lands on kind.
	d.i = 1
	return d.readPackedGorillaFloat64Slice()
}

// ---- round-trip test ----

func TestALPRoundTrip(t *testing.T) {
	cases := map[string][]float64{
		"quantized":       alpFixtureQuantized(),
		"smooth":          alpFixtureSmooth(),
		"smoothQuantized": alpFixtureSmoothQuantized(),
		"empty":           {},
		"single":          {42.42},
		"withNaNInf":      {1.5, math.NaN(), math.Inf(1), math.Inf(-1), -0.0, 3.14},
	}
	for name, s := range cases {
		enc := alpEncode(s)
		got := alpDecode(enc)
		if len(got) != len(s) {
			t.Fatalf("%s: len mismatch: got %d want %d", name, len(got), len(s))
		}
		for i := range s {
			gb := math.Float64bits(got[i])
			wb := math.Float64bits(s[i])
			if gb != wb {
				t.Fatalf("%s: value %d mismatch: got %v (%#x) want %v (%#x)",
					name, i, got[i], gb, s[i], wb)
			}
		}
	}

	// Also verify the Gorilla measurement harness round-trips, so the
	// benchmark comparison is apples-to-apples lossless.
	for name, s := range cases {
		if len(s) == 0 {
			continue
		}
		payload := gorillaEncode(s)
		got, err := gorillaDecode(payload)
		if err != nil {
			t.Fatalf("%s: gorilla decode error: %v", name, err)
		}
		if len(got) != len(s) {
			t.Fatalf("%s: gorilla len mismatch: got %d want %d", name, len(got), len(s))
		}
		for i := range s {
			if math.Float64bits(got[i]) != math.Float64bits(s[i]) {
				t.Fatalf("%s: gorilla value %d mismatch: got %v want %v", name, i, got[i], s[i])
			}
		}
	}
}

// ---- size report (run with -v) ----

func TestALPSizeReport(t *testing.T) {
	fixtures := []struct {
		name string
		data []float64
	}{
		{"quantized", alpFixtureQuantized()},
		{"smooth", alpFixtureSmooth()},
		{"smoothQuantized", alpFixtureSmoothQuantized()},
	}
	for _, f := range fixtures {
		raw := len(f.data) * 8
		alpSz := len(alpEncode(f.data))
		gorSz := len(gorillaEncode(f.data))
		t.Logf("%-16s n=%d raw=%dB | ALP=%dB (%.2f%% of raw, %.2fx) | Gorilla=%dB (%.2f%% of raw, %.2fx) | ALP/Gorilla=%.2f%%",
			f.name, len(f.data), raw,
			alpSz, 100*float64(alpSz)/float64(raw), float64(raw)/float64(alpSz),
			gorSz, 100*float64(gorSz)/float64(raw), float64(raw)/float64(gorSz),
			100*float64(alpSz)/float64(gorSz))
	}
}

// ---- decode benchmarks ----

var alpSink []float64

func benchALPDecode(b *testing.B, data []float64) {
	enc := alpEncode(data)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alpSink = alpDecode(enc)
	}
}

func benchGorillaDecode(b *testing.B, data []float64) {
	payload := gorillaEncode(data)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, err := gorillaDecode(payload)
		if err != nil {
			b.Fatal(err)
		}
		alpSink = v
	}
}

func BenchmarkALPDecodeQuantized(b *testing.B)     { benchALPDecode(b, alpFixtureQuantized()) }
func BenchmarkGorillaDecodeQuantized(b *testing.B) { benchGorillaDecode(b, alpFixtureQuantized()) }
func BenchmarkALPDecodeSmooth(b *testing.B)        { benchALPDecode(b, alpFixtureSmooth()) }
func BenchmarkGorillaDecodeSmooth(b *testing.B)    { benchGorillaDecode(b, alpFixtureSmooth()) }
func BenchmarkALPDecodeSmoothQuant(b *testing.B)   { benchALPDecode(b, alpFixtureSmoothQuantized()) }
func BenchmarkGorillaDecodeSmoothQuant(b *testing.B) {
	benchGorillaDecode(b, alpFixtureSmoothQuantized())
}
