package qdf

import (
	"math/rand"
	"reflect"
	"testing"
)

// Realistic-payload corpus modeled after the embedded fixtures
// go-json-experiment ships (Twitter status, citm catalog, FHIR,
// telemetry). Each builder produces a sizeable structured Go value
// representative of the workload qdf targets in production: log
// batches, metric series, and a wide column-store row. Each value
// must round-trip identically through every Marshal entry point.

// ---------- TelemetryBatch (log batch with repeating attribute keys)

type telemetryEvent struct {
	TS       int64             `qdf:"ts"`
	Service  string            `qdf:"service"`
	Region   string            `qdf:"region"`
	Level    string            `qdf:"level"`
	Host     string            `qdf:"host"`
	Msg      string            `qdf:"msg"`
	Span     uint64            `qdf:"span"`
	Trace    uint64            `qdf:"trace"`
	Duration int64             `qdf:"duration_us"`
	Tags     map[string]string `qdf:"tags"`
}

type telemetryBatch struct {
	Source string           `qdf:"source"`
	Events []telemetryEvent `qdf:"events"`
}

func makeTelemetryBatch(n int) telemetryBatch {
	rng := rand.New(rand.NewSource(1))
	services := []string{"ingest", "auth", "billing", "api-gateway"}
	regions := []string{"eu-west-1", "us-east-1", "ap-south-1"}
	levels := []string{"info", "warn", "error", "debug"}
	hosts := []string{"node-001", "node-002", "node-003", "node-004"}
	out := telemetryBatch{Source: "qdf-realistic-corpus"}
	out.Events = make([]telemetryEvent, n)
	t0 := int64(1_700_000_000)
	for i := range out.Events {
		out.Events[i] = telemetryEvent{
			TS:       t0 + int64(i),
			Service:  services[rng.Intn(len(services))],
			Region:   regions[rng.Intn(len(regions))],
			Level:    levels[rng.Intn(len(levels))],
			Host:     hosts[rng.Intn(len(hosts))],
			Msg:      "user request completed",
			Span:     rng.Uint64(),
			Trace:    rng.Uint64(),
			Duration: rng.Int63n(1_000_000),
			Tags: map[string]string{
				"version": "v3.42.1",
				"client":  "go-client/1.20",
			},
		}
	}
	return out
}

// ---------- MetricSeries (timestamps + values + low-cardinality tags)

type metricSeries struct {
	Name      string    `qdf:"name"`
	Unit      string    `qdf:"unit"`
	Timestamp []int64   `qdf:"ts"`
	Value     []float64 `qdf:"v"`
	Tags      []string  `qdf:"tags"`
}

func makeMetricSeries(n int) metricSeries {
	rng := rand.New(rand.NewSource(2))
	ts := make([]int64, n)
	vs := make([]float64, n)
	base := int64(1_700_000_000)
	for i := range ts {
		ts[i] = base + int64(i)*15
		vs[i] = 50 + rng.NormFloat64()
	}
	return metricSeries{
		Name:      "cpu.usage.percent",
		Unit:      "percent",
		Timestamp: ts,
		Value:     vs,
		Tags:      []string{"host:node-001", "service:ingest", "region:eu-west-1"},
	}
}

// ---------- WideRow (column-store row mixing every QPack-eligible type)

type wideRow struct {
	ID       uint64    `qdf:"id"`
	Hash     []byte    `qdf:"hash"`
	Source   string    `qdf:"source"`
	Active   bool      `qdf:"active"`
	Score    float64   `qdf:"score"`
	Bools    []bool    `qdf:"bools"`
	Counts   []int64   `qdf:"counts"`
	Vec      []float64 `qdf:"vec"`
	Vec32    []float32 `qdf:"vec32"`
	Labels   []string  `qdf:"labels"`
	IDs      []uint64  `qdf:"ids"`
	Children []wideRow `qdf:"children"`
}

func makeWideRow(rng *rand.Rand, depth int) wideRow {
	row := wideRow{
		ID:     rng.Uint64(),
		Hash:   []byte{0xDE, 0xAD, 0xBE, 0xEF, byte(rng.Intn(256))},
		Source: "qdf",
		Active: rng.Intn(2) == 0,
		Score:  rng.NormFloat64(),
	}
	for range 16 {
		row.Bools = append(row.Bools, rng.Intn(2) == 0)
		row.Counts = append(row.Counts, int64(rng.Intn(1<<16)))
		row.Vec = append(row.Vec, rng.NormFloat64())
		row.Vec32 = append(row.Vec32, float32(rng.NormFloat64()))
		row.Labels = append(row.Labels, "tag")
		row.IDs = append(row.IDs, uint64(1_700_000_000+rng.Intn(1<<16)))
	}
	// Initialize as empty (non-nil) so the round-trip's reflect.DeepEqual
	// does not flag nil-vs-empty-slice as a wire bug. Both are valid
	// wire encodings, but the test fixture commits to one.
	row.Children = []wideRow{}
	if depth > 0 {
		row.Children = []wideRow{makeWideRow(rng, depth-1)}
	}
	return row
}

// ---------- Round-trip helpers

func roundTripAll(t *testing.T, in any, makeOut func() any) {
	t.Helper()
	for label, opts := range map[string]Options{
		"Speed":    OptSpeed,
		"QPack":    OptQPack,
		"Balanced": OptBalanced,
	} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("%s encode: %v", label, err)
		}
		outPtr := makeOut()
		if err := Unmarshal(buf, outPtr); err != nil {
			t.Fatalf("%s decode: %v", label, err)
		}
		out := reflect.ValueOf(outPtr).Elem().Interface()
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("%s round-trip mismatch (size=%d):\n in=%+v\nout=%+v", label, len(buf), in, out)
		}
	}
}

// ---------- Tests

func TestCorpus_TelemetryBatch(t *testing.T) {
	for _, n := range []int{1, 16, 256, 1000} {
		t.Run("", func(t *testing.T) {
			in := makeTelemetryBatch(n)
			roundTripAll(t, in, func() any { return &telemetryBatch{} })
		})
	}
}

func TestCorpus_MetricSeries(t *testing.T) {
	for _, n := range []int{1, 16, 256, 1024} {
		t.Run("", func(t *testing.T) {
			in := makeMetricSeries(n)
			roundTripAll(t, in, func() any { return &metricSeries{} })
		})
	}
}

func TestCorpus_WideRow(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, depth := range []int{0, 1, 3, 6} {
		t.Run("", func(t *testing.T) {
			in := makeWideRow(rng, depth)
			roundTripAll(t, in, func() any { return &wideRow{} })
		})
	}
}

// Cross-encoder agreement: every encoder produces a buffer that
// decodes to the same Go value. Catches a single-encoder divergence
// rather than just a round-trip symptom.
func TestCorpus_AllEncodersAgree(t *testing.T) {
	in := makeTelemetryBatch(64)
	bufFast, _ := Marshal(in, OptSpeed)
	bufQPack, _ := Marshal(in, OptQPack)
	bufDense, _ := Marshal(in, OptBalanced)
	var aFast, aQPack, aDense telemetryBatch
	if err := Unmarshal(bufFast, &aFast); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(bufQPack, &aQPack); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(bufDense, &aDense); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(aFast, aQPack) {
		t.Fatalf("Fast vs QPack diverge")
	}
	if !reflect.DeepEqual(aFast, aDense) {
		t.Fatalf("Fast vs Dense diverge")
	}
}

// Bench so the realistic-shape numbers stay reproducible alongside the
// synthetic micro-benchmarks.
func BenchmarkCorpus_TelemetryBatch1000(b *testing.B) {
	in := makeTelemetryBatch(1000)
	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in, OptSpeed)
		}
	})
	b.Run("MarshalQPack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in, OptQPack)
		}
	})
	b.Run("MarshalDense", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in, OptBalanced)
		}
	})
}
