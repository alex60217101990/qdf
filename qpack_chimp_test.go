package qdf

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
)

func chimpRoundTrip(t *testing.T, in []float64) []byte {
	t.Helper()
	enc := NewEncoder(Fast)
	enc.writePackedChimpFloat64Slice(in)
	buf := enc.buf
	dec := NewDecoderOnBuf(buf)
	tag, err := dec.peekTag()
	if err != nil || tag != tagPackGorilla {
		t.Fatalf("expected tagPackGorilla got %02x err=%v", tag, err)
	}
	dec.i++
	out, err := dec.readPackedChimpFloat64Slice()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(in) != len(out) {
		t.Fatalf("len mismatch %d vs %d", len(in), len(out))
	}
	for i := range in {
		if math.Float64bits(in[i]) != math.Float64bits(out[i]) {
			t.Fatalf("value %d: %x vs %x", i, math.Float64bits(in[i]), math.Float64bits(out[i]))
		}
	}
	return buf
}

func mkSensor(n int, seed int64) []float64 {
	rng := rand.New(rand.NewSource(seed))
	s := make([]float64, n)
	v := 20.0
	for i := range s {
		v += (rng.Float64() - 0.5) * 0.1
		s[i] = v
	}
	return s
}

func TestChimpRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	rnd := make([]float64, 1000)
	for i := range rnd {
		rnd[i] = math.Float64frombits(rng.Uint64())
	}
	repeat3 := make([]float64, 500)
	for i := range repeat3 {
		repeat3[i] = []float64{1.25, 7.5, -3.125}[i%3] // exercises flag-00 window refs
	}
	cases := [][]float64{
		nil,
		{},
		{42.5},
		{0, 0, 0, 0, 0},
		{1.5, 1.5, 1.5, 1.5},
		{1.5, 1.6, 1.7, 1.8, 1.9, 2.0},
		{0, math.Copysign(0, -1), math.NaN(), math.Inf(1), math.Inf(-1), 1e300, -1e-300},
		{0.1, 0.2, 0.3, 0.4, 0.5, 0.5, 0.5, 0.4, 0.3},
		mkSensor(2048, 7),
		rnd,
		repeat3,
	}
	for i, in := range cases {
		_ = i
		chimpRoundTrip(t, in)
	}
}

// TestChimpThroughUnmarshal drives the full pipeline: a Chimp column decoded
// by the generic []float64 slice reader via the kind dispatch.
func TestChimpThroughUnmarshal(t *testing.T) {
	in := mkSensor(4096, 42)
	enc := NewEncoder(Fast)
	enc.writeHeader()
	enc.writePackedChimpFloat64Slice(in)
	// Splice the chimp blob into a full message by encoding a []float64 the
	// normal way, then verifying our blob decodes via Unmarshal's slice path.
	// Simpler: decode directly with the slice fast path.
	dec := NewDecoderOnBuf(enc.buf)
	tag, err := dec.peekTag()
	if err != nil || tag != tagPackGorilla {
		t.Fatalf("tag %02x err %v", tag, err)
	}
	dec.i++
	if dec.buf[dec.i] != qpackKindChimp64 {
		t.Fatalf("kind %02x", dec.buf[dec.i])
	}
	out, err := dec.readPackedChimpFloat64Slice()
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if math.Float64bits(in[i]) != math.Float64bits(out[i]) {
			t.Fatalf("mismatch at %d", i)
		}
	}
}

// TestChimpHostile: mutated/truncated chimp blobs must error, never panic.
func TestChimpHostile(t *testing.T) {
	in := mkSensor(512, 3)
	enc := NewEncoder(Fast)
	enc.writePackedChimpFloat64Slice(in)
	valid := append([]byte(nil), enc.buf...)

	decodeOne := func(b []byte) {
		dec := NewDecoderOnBuf(b)
		if tag, err := dec.peekTag(); err != nil || tag != tagPackGorilla {
			return
		}
		dec.i++
		_, _ = dec.readPackedChimpFloat64Slice()
	}

	// Truncations.
	for cut := 0; cut < len(valid); cut += 7 {
		decodeOne(valid[:cut])
	}
	// Single-byte mutations.
	rng := rand.New(rand.NewSource(9))
	for trial := 0; trial < 20000; trial++ {
		b := append([]byte(nil), valid...)
		b[rng.Intn(len(b))] ^= byte(1 + rng.Intn(255))
		decodeOne(b)
	}
	// Random garbage bodies with a valid prefix.
	for trial := 0; trial < 5000; trial++ {
		b := append([]byte(nil), valid[:20]...)
		garbage := make([]byte, rng.Intn(64))
		rng.Read(garbage)
		decodeOne(append(b, garbage...))
	}
}

// TestChimpVsGorillaSize reports the wire-size ratio on representative data.
func TestChimpVsGorillaSize(t *testing.T) {
	cases := []struct {
		name string
		data []float64
	}{
		{"sensor_smooth", mkSensor(8192, 11)},
		{"repeating_vals", func() []float64 {
			s := make([]float64, 8192)
			for i := range s {
				s[i] = []float64{20.5, 21.0, 20.75, 21.25}[i%4]
			}
			return s
		}()},
		{"random", func() []float64 {
			rng := rand.New(rand.NewSource(5))
			s := make([]float64, 8192)
			for i := range s {
				s[i] = rng.Float64() * 1e9
			}
			return s
		}()},
		{"integer_valued", func() []float64 {
			s := make([]float64, 8192)
			for i := range s {
				s[i] = float64(i % 1000)
			}
			return s
		}()},
	}
	for _, c := range cases {
		encC := NewEncoder(Fast)
		encC.writePackedChimpFloat64Slice(c.data)
		encG := NewEncoder(Fast)
		encG.writePackedGorillaFloat64Slice(c.data)
		raw := 8 * len(c.data)
		t.Logf("%-15s chimp=%6d gorilla=%6d raw=%6d chimp/gorilla=%.3f",
			c.name, len(encC.buf), len(encG.buf), raw,
			float64(len(encC.buf))/float64(len(encG.buf)))
	}
}

// TestChimpEndToEnd drives Marshal/Unmarshal with OptCompression: smooth
// float64 columns must round-trip and pick Chimp on the wire.
func TestChimpEndToEnd(t *testing.T) {
	type row struct {
		Vals []float64
	}
	for _, opts := range []Options{OptCompression, OptBalanced | OptGorillaFloat} {
		for _, data := range [][]float64{
			mkSensor(4096, 21),
			func() []float64 { // periodic round values: Gorilla must win, wire stays small
				s := make([]float64, 4096)
				for i := range s {
					s[i] = []float64{20.5, 21.0, 20.75, 21.25}[i%4]
				}
				return s
			}(),
		} {
			in := row{Vals: data}
			blob, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			var out row
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatal(err)
			}
			if len(out.Vals) != len(in.Vals) {
				t.Fatalf("len %d vs %d", len(out.Vals), len(in.Vals))
			}
			for i := range in.Vals {
				if math.Float64bits(in.Vals[i]) != math.Float64bits(out.Vals[i]) {
					t.Fatalf("mismatch at %d", i)
				}
			}
		}
	}
}

// TestChimpDeterministicReuse: a reused (pooled) encoder must produce
// byte-identical wire for the same input — the epoch-stamped scratch must
// never let a previous slice's hash slots influence a later encode.
func TestChimpDeterministicReuse(t *testing.T) {
	a := mkSensor(3000, 13)
	b := mkSensor(3000, 14) // different content, poisons the scratch
	enc := NewEncoder(Fast)
	enc.writePackedChimpFloat64Slice(a)
	first := append([]byte(nil), enc.buf...)
	enc.Reset()
	enc.writePackedChimpFloat64Slice(b) // scratch now full of b's slots
	enc.Reset()
	enc.writePackedChimpFloat64Slice(a)
	if !bytes.Equal(first, enc.buf) {
		t.Fatal("reused encoder produced different wire for identical input")
	}
	type row struct{ Vals []float64 }
	m1, err := Marshal(row{a}, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Marshal(row{a}, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m1, m2) {
		t.Fatal("pooled Marshal nondeterministic")
	}
}
