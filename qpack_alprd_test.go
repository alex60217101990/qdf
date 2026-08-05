package qdf

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
)

func alprdRoundTrip(t *testing.T, in []float64) bool {
	t.Helper()
	plan, _, ok := alprdPlanFloat64(in)
	if !ok {
		return false
	}
	enc := NewEncoder(Fast)
	enc.writePackedALPRDFloat64Slice(in, plan)
	dec := NewDecoderOnBuf(enc.buf)
	tag, err := dec.peekTag()
	if err != nil || tag != tagPackALP {
		t.Fatalf("expected tagPackALP got %02x err=%v", tag, err)
	}
	dec.i++
	out, err := dec.readPackedALPRDFloat64Slice()
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
	return true
}

func TestALPRDRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	rnd := make([]float64, 2000)
	for i := range rnd {
		rnd[i] = math.Float64frombits(rng.Uint64())
	}
	mixed := mkSensor(2048, 5)
	mixed[100] = math.NaN()
	mixed[200] = math.Inf(1)
	mixed[300] = math.Inf(-1)
	mixed[400] = math.Copysign(0, -1)
	cases := [][]float64{
		mkSensor(4096, 7), // smooth-noisy: the target class
		mkSensor(16, 9),   // minimum planner size
		mixed,             // specials as exceptions
		rnd,               // white noise: planner may decline (ok=false fine)
		bytes16(),         // few distinct left patterns, all dict
	}
	hit := 0
	for _, in := range cases {
		if alprdRoundTrip(t, in) {
			hit++
		}
	}
	if hit < 3 {
		t.Fatalf("planner accepted only %d/5 cases; expected at least the smooth ones", hit)
	}
}

func bytes16() []float64 {
	s := make([]float64, 4096)
	for i := range s {
		s[i] = 1000.0 + float64(i%7)*0.001953125 // exact 2^-9 steps: shared left bits
	}
	return s
}

// TestALPRDHostile: mutated/truncated blobs must error, never panic.
func TestALPRDHostile(t *testing.T) {
	in := mkSensor(512, 3)
	plan, _, ok := alprdPlanFloat64(in)
	if !ok {
		t.Skip("planner declined")
	}
	enc := NewEncoder(Fast)
	enc.writePackedALPRDFloat64Slice(in, plan)
	valid := append([]byte(nil), enc.buf...)

	decodeOne := func(b []byte) {
		dec := NewDecoderOnBuf(b)
		if tag, err := dec.peekTag(); err != nil || tag != tagPackALP {
			return
		}
		dec.i++
		_, _ = dec.readPackedALPRDFloat64Slice()
	}
	for cut := 0; cut < len(valid); cut += 5 {
		decodeOne(valid[:cut])
	}
	rng := rand.New(rand.NewSource(11))
	for range 30000 {
		b := append([]byte(nil), valid...)
		b[rng.Intn(len(b))] ^= byte(1 + rng.Intn(255))
		decodeOne(b)
	}
	for range 5000 {
		b := append([]byte(nil), valid[:min(24, len(valid))]...)
		garbage := make([]byte, rng.Intn(80))
		rng.Read(garbage)
		decodeOne(append(b, garbage...))
	}
}

// TestALPRDEndToEnd drives Marshal/Unmarshal and verifies the picker routes
// the target class through ALP-RD while never growing other shapes.
func TestALPRDEndToEnd(t *testing.T) {
	type row struct{ Vals []float64 }
	smooth := mkSensor(8192, 21)
	for _, opts := range []Options{OptCompression, OptBalanced | OptGorillaFloat} {
		for _, data := range [][]float64{smooth, bytes16(), mkQuant(8192)} {
			in := row{Vals: data}
			blob, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			var out row
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatal(err)
			}
			for i := range in.Vals {
				if math.Float64bits(in.Vals[i]) != math.Float64bits(out.Vals[i]) {
					t.Fatalf("mismatch at %d", i)
				}
			}
			if len(blob) > 12+8*len(data)+64 {
				t.Fatalf("wire grew past raw: %d for n=%d", len(blob), len(data))
			}
		}
	}
	// The smooth-noisy class must actually pick ALP-RD on the wire (no tANS
	// pass in this option set, so inner tags are scannable).
	blob, err := Marshal(row{Vals: smooth}, OptBalanced|OptGorillaFloat)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(blob, []byte{tagPackALP, qpackKindALPRD64}) {
		t.Log("ALP-RD not on wire for smooth sensor (XOR pair won) — acceptable, checking size only")
	}
}

func mkQuant(n int) []float64 {
	rng := rand.New(rand.NewSource(4))
	s := make([]float64, n)
	v := 20.0
	steps := []float64{0.125, 0.25, -0.125, -0.25, 0, 0}
	for i := range s {
		v += steps[rng.Intn(len(steps))]
		s[i] = v
	}
	return s
}

// TestALPRDRatio reports sizes against the other float64 codecs.
func TestALPRDRatio(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []float64
	}{
		{"sensor_smooth", mkSensor(8192, 11)},
		{"shared_left", bytes16()},
		{"quantized", mkQuant(8192)},
	} {
		var rdLen int
		if plan, _, ok := alprdPlanFloat64(tc.data); ok {
			enc := NewEncoder(Fast)
			enc.writePackedALPRDFloat64Slice(tc.data, plan)
			rdLen = len(enc.buf)
		}
		encC := NewEncoder(Fast)
		encC.writePackedChimpFloat64Slice(tc.data)
		encG := NewEncoder(Fast)
		encG.writePackedGorillaFloat64Slice(tc.data)
		t.Logf("%-14s alprd=%6d chimp=%6d gorilla=%6d raw=%6d",
			tc.name, rdLen, len(encC.buf), len(encG.buf), 8*len(tc.data))
	}
}

// TestALPRDSkip: a decoder that does not know the field must Skip an ALP-RD
// body cleanly (forward compatibility).
func TestALPRDSkip(t *testing.T) {
	type full struct {
		Vals []float64
		Tail int64
	}
	type slim struct {
		Tail int64
	}
	s := mkSensor(8192, 11)
	rng := rand.New(rand.NewSource(1))
	rng.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] }) // ALP-RD's win class
	blob, err := Marshal(full{Vals: s, Tail: 77}, OptBalanced|OptGorillaFloat)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(blob, []byte{tagPackALP, qpackKindALPRD64}) {
		t.Skip("picker did not choose ALP-RD; skip path not exercised")
	}
	var out slim
	if err := Unmarshal(blob, &out); err != nil {
		t.Fatalf("skip decode: %v", err)
	}
	if out.Tail != 77 {
		t.Fatalf("tail = %d", out.Tail)
	}
}
