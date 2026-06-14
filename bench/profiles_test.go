package bench

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// profiles_test.go runs a head-to-head matrix per "real" usage
// scenario. Each scenario picks the qdf Options combination that
// the docs/CHOOSING.md recipe recommends for it and compares against
// json + msgpack on the same fixture.
//
// Run:
//   go test -C bench -bench=BenchmarkProfile -benchmem -benchtime=300ms
//
// Numbers from the most recent run land in docs/BENCH.md under the
// "Scenario profiles" section.

// ----- shared fixture types ---------------------------------------

type hotPathEvent struct {
	ID       int       `qdf:"id"        json:"id"        msgpack:"id"`
	Source   string    `qdf:"source"    json:"source"    msgpack:"source"`
	Severity int       `qdf:"severity"  json:"severity"  msgpack:"severity"`
	When     time.Time `qdf:"when"      json:"when"      msgpack:"when"`
	Msg      string    `qdf:"msg"       json:"msg"       msgpack:"msg"`
}

type telemetryRow struct {
	Service string   `qdf:"service"  json:"service"  msgpack:"service"`
	Region  string   `qdf:"region"   json:"region"   msgpack:"region"`
	Level   string   `qdf:"level"    json:"level"    msgpack:"level"`
	Host    string   `qdf:"host"     json:"host"     msgpack:"host"`
	TraceID string   `qdf:"trace_id" json:"trace_id" msgpack:"trace_id"`
	Status  int      `qdf:"status"   json:"status"   msgpack:"status"`
	Tags    []string `qdf:"tags"     json:"tags"     msgpack:"tags"`
}

type metricSeries struct {
	Name      string    `qdf:"name"       json:"name"       msgpack:"name"`
	Timestamp []int64   `qdf:"timestamps" json:"timestamps" msgpack:"timestamps"`
	Value     []float64 `qdf:"values"     json:"values"     msgpack:"values"`
	Flag      []bool    `qdf:"flags"      json:"flags"      msgpack:"flags"`
}

type embeddingVec struct {
	ID        string    `qdf:"id"        json:"id"        msgpack:"id"`
	Embedding []float32 `qdf:"embedding" json:"embedding" msgpack:"embedding"`
}

type configRecord struct {
	Name   string            `qdf:"name"   json:"name"   msgpack:"name"`
	Limits map[string]int    `qdf:"limits" json:"limits" msgpack:"limits"`
	Routes map[string]string `qdf:"routes" json:"routes" msgpack:"routes"`
	Tags   []string          `qdf:"tags"   json:"tags"   msgpack:"tags"`
}

type archiveSnapshot struct {
	Generated time.Time      `qdf:"generated" json:"generated" msgpack:"generated"`
	Rows      []telemetryRow `qdf:"rows"      json:"rows"      msgpack:"rows"`
}

// ----- fixture builders ------------------------------------------

func mkHotPathEvent() hotPathEvent {
	return hotPathEvent{
		ID:       42,
		Source:   "ingest",
		Severity: 3,
		When:     time.Unix(1_700_000_000, 0).UTC(),
		Msg:      "user authenticated",
	}
}

func mkTelemetryBatch(n int) []telemetryRow {
	services := []string{"billing", "auth", "ingest", "metrics", "api"}
	regions := []string{"eu-west-1", "us-east-1", "ap-southeast-2"}
	levels := []string{"info", "warn", "error"}
	out := make([]telemetryRow, n)
	rnd := rand.New(rand.NewSource(1))
	for i := range out {
		out[i] = telemetryRow{
			Service: services[rnd.Intn(len(services))],
			Region:  regions[rnd.Intn(len(regions))],
			Level:   levels[rnd.Intn(len(levels))],
			Host:    fmt.Sprintf("ip-10-0-%d", i%256),
			TraceID: fmt.Sprintf("%016x", rnd.Uint64()),
			Status:  200 + rnd.Intn(4)*100,
			Tags:    []string{"prod", "v3"},
		}
	}
	return out
}

func mkMetricSeries(n int) metricSeries {
	ts := make([]int64, n)
	vs := make([]float64, n)
	fs := make([]bool, n)
	rnd := rand.New(rand.NewSource(2))
	for i := range n {
		ts[i] = 1_700_000_000 + int64(i)
		vs[i] = float64(rnd.NormFloat64())
		fs[i] = i%3 == 0
	}
	return metricSeries{Name: "service.cpu.load", Timestamp: ts, Value: vs, Flag: fs}
}

// mkMetricSeriesSmooth produces a metric series whose float64
// column is the shape Gorilla XOR coding was designed for:
// quantized sensor readings with frequent repeats and small
// integer-multiple steps. Real telemetry that fits this profile:
//   - a CPU-load gauge with 0.1 %-resolution that holds steady
//     for many samples before stepping;
//   - a temperature sensor reporting 0.5 °C increments;
//   - a request-rate counter that increments in fixed buckets.
//
// On this fixture pickF64Codec hits qpackGorilla: each repeated
// sample collapses to a single control bit and each step emits
// the small XOR delta in ~14-20 bits, well below raw's 64
// bits/sample.
func mkMetricSeriesSmooth(n int) metricSeries {
	ts := make([]int64, n)
	vs := make([]float64, n)
	fs := make([]bool, n)
	rnd := rand.New(rand.NewSource(42))
	value := 50.0
	for i := range n {
		ts[i] = 1_700_000_000 + int64(i)
		// 70 % chance the value stays at its previous quantum;
		// 30 % chance it steps by ±0.1 (one quantum). Both pure
		// repeats and small XOR deltas play to Gorilla's strengths.
		if i > 0 && rnd.Float64() < 0.7 {
			vs[i] = value
		} else {
			if rnd.Float64() < 0.5 {
				value += 0.1
			} else {
				value -= 0.1
			}
			vs[i] = value
		}
		fs[i] = i%3 == 0
	}
	return metricSeries{Name: "service.cpu.load.smooth", Timestamp: ts, Value: vs, Flag: fs}
}

func mkEmbedding(dim int) embeddingVec {
	v := make([]float32, dim)
	rnd := rand.New(rand.NewSource(3))
	for i := range v {
		v[i] = float32(rnd.NormFloat64())
	}
	return embeddingVec{ID: "doc-42", Embedding: v}
}

func mkConfig() configRecord {
	return configRecord{
		Name: "service.config.v3",
		Limits: map[string]int{
			"rps": 1000, "concurrency": 64, "queue": 256,
			"retries": 3, "timeout_ms": 5000,
		},
		Routes: map[string]string{
			"/v1/auth":   "auth-svc",
			"/v1/bill":   "billing-svc",
			"/v1/metric": "metrics-svc",
			"/v1/log":    "ingest-svc",
		},
		Tags: []string{"prod", "v3", "eu-west-1"},
	}
}

func mkArchive(rows int) archiveSnapshot {
	return archiveSnapshot{
		Generated: time.Unix(1_700_000_000, 0).UTC(),
		Rows:      mkTelemetryBatch(rows),
	}
}

// statusBatch isolates the kind of column a real telemetry pipeline
// emits as a dense column: a long run of HTTP-status codes dominated
// by 200, with sporadic 4xx/5xx bursts when an upstream wobbles. The
// shape is the canonical RLE win — 1024 elements collapse to ~6-8
// (value, runLen) varuint pairs once the picker selects qpackRLE.
type statusBatch struct {
	Endpoint string  `qdf:"endpoint" json:"endpoint" msgpack:"endpoint"`
	Codes    []int64 `qdf:"codes"    json:"codes"    msgpack:"codes"`
}

func mkStatusBatch(n int) statusBatch {
	codes := make([]int64, n)
	rnd := rand.New(rand.NewSource(7))
	// Walk the slice in pseudo-incident bursts: long stretches of
	// 200 punctuated by short 500 / 404 runs. Run lengths chosen to
	// emulate a "mostly healthy with occasional outages" SLO trace.
	idx := 0
	for idx < n {
		// healthy run: 50-200 samples of 200
		runLen := 50 + rnd.Intn(150)
		if idx+runLen > n {
			runLen = n - idx
		}
		for i := 0; i < runLen; i++ {
			codes[idx+i] = 200
		}
		idx += runLen
		if idx >= n {
			break
		}
		// fault run: 5-25 samples of 500 or 404
		code := int64(500)
		if rnd.Float64() < 0.5 {
			code = 404
		}
		fault := 5 + rnd.Intn(20)
		if idx+fault > n {
			fault = n - idx
		}
		for i := 0; i < fault; i++ {
			codes[idx+i] = code
		}
		idx += fault
	}
	return statusBatch{Endpoint: "POST /v1/ingest", Codes: codes}
}

// ----- benchmark harness -----------------------------------------

// runProfile defines one scenario: a fixture, the recommended qdf
// Options for that scenario, and a label. It runs encode + decode
// benchmarks for json, msgpack, qdf.
func runProfile[T any](b *testing.B, name string, opts qdf.Options, in T) {
	// Pre-compute the wire size each codec produces so every subtest
	// can report MB/s alongside ns/op + allocs. SetBytes makes the
	// throughput column comparable across json / msgpack / qdf.
	jsonBuf, _ := json.Marshal(in)
	msgpBuf, _ := msgpack.Marshal(in)
	qdfBuf, _ := qdf.Marshal(in, opts)

	b.Run(name+"/encode/json", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBuf)))
		for b.Loop() {
			_, err := json.Marshal(in)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run(name+"/encode/msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(msgpBuf)))
		for b.Loop() {
			_, err := msgpack.Marshal(in)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run(name+"/encode/qdf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(qdfBuf)))
		for b.Loop() {
			_, err := qdf.Marshal(in, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run(name+"/decode/json", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBuf)))
		for b.Loop() {
			var out T
			if err := json.Unmarshal(jsonBuf, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run(name+"/decode/msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(msgpBuf)))
		for b.Loop() {
			var out T
			if err := msgpack.Unmarshal(msgpBuf, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run(name+"/decode/qdf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(qdfBuf)))
		for b.Loop() {
			var out T
			if err := qdf.Unmarshal(qdfBuf, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ----- scenarios -------------------------------------------------

// BenchmarkProfile_HotPath — single small struct, tightest CPU
// budget. Recommended: OptSpeed.
func BenchmarkProfile_HotPath(b *testing.B) {
	runProfile(b, "hot_path", qdf.OptSpeed, mkHotPathEvent())
}

// BenchmarkProfile_TelemetryBatch — 1000 log/trace rows with
// repeating service / region / level columns. Recommended:
// OptBalanced.
func BenchmarkProfile_TelemetryBatch(b *testing.B) {
	runProfile(b, "telemetry_1k", qdf.OptBalanced, mkTelemetryBatch(1000))
}

// BenchmarkProfile_TelemetryBatch_PreIntern measures the encode
// win when the caller registers the hot string pool via
// Encoder.PreIntern before encoding. Real services with a known
// vocabulary (service names, region codes, severity levels)
// can use the API to skip the intern table's hash + probe on
// every WriteString that hits the pool.
func BenchmarkProfile_TelemetryBatch_PreIntern(b *testing.B) {
	const records = 1000
	rows := mkTelemetryBatch(records)
	// Hot pool: same backing slices the fixture builder reused.
	services := []string{"billing", "auth", "ingest", "metrics", "api"}
	regions := []string{"eu-west-1", "us-east-1", "ap-southeast-2"}
	levels := []string{"info", "warn", "error"}
	tags := []string{"prod", "v3"}
	hotPool := append(append(append(append([]string{}, services...), regions...), levels...), tags...)

	// Probe a single encode so the throughput column matches the
	// default Profile_TelemetryBatch/encode/qdf line in MB/s.
	probe := qdf.NewEncoderWith(qdf.OptBalanced)
	probe.PreIntern(hotPool...)
	_ = probe.EncodeValue(rows)
	wireBytes := int64(len(probe.Bytes()))

	b.Run("telemetry_1k_preintern/encode/qdf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(wireBytes)
		enc := qdf.NewEncoderWith(qdf.OptBalanced)
		for b.Loop() {
			enc.Reset()
			enc.ApplyOpts(qdf.OptBalanced)
			enc.PreIntern(hotPool...)
			if err := enc.EncodeValue(rows); err != nil {
				b.Fatal(err)
			}
			_ = enc.Bytes()
		}
	})
}

// BenchmarkProfile_MetricSeries — numeric / boolean columns, no
// repeating strings. Recommended: OptQPack (skip intern overhead).
func BenchmarkProfile_MetricSeries(b *testing.B) {
	runProfile(b, "metric_1024", qdf.OptQPack, mkMetricSeries(1024))
}

// BenchmarkProfile_MetricSeriesSmooth — same shape as MetricSeries
// but with smoothly-varying float64 values. OptQPack (no Gorilla):
// raw-LE bulk floats, fast path. The wire size matches the random
// variant since both ship 8 bytes/sample.
func BenchmarkProfile_MetricSeriesSmooth(b *testing.B) {
	runProfile(b, "metric_smooth_1024", qdf.OptQPack, mkMetricSeriesSmooth(1024))
}

// BenchmarkProfile_MetricSeriesSmoothCompress — same fixture under
// OptCompression. Gorilla XOR fires (~6-20 bits/sample on smooth
// data) and shrinks the float body ~10×, at the cost of bit-level
// encode/decode (~10× more CPU per slice). Use this preset when
// wire size dominates and the payload is dominated by smooth
// time-series floats.
func BenchmarkProfile_MetricSeriesSmoothCompress(b *testing.B) {
	runProfile(b, "metric_smooth_1024_compress", qdf.OptCompression, mkMetricSeriesSmooth(1024))
}

// BenchmarkProfile_StatusBatch — long []int64 column dominated by
// repeated values (HTTP-status pattern). Picker selects qpackRLE
// and wire collapses from ~1024 raw int64 values to a handful of
// (value, runLen) pairs. Recommended: OptQPack (Dense unused for
// this single-slice payload).
func BenchmarkProfile_StatusBatch(b *testing.B) {
	runProfile(b, "status_1024", qdf.OptQPack, mkStatusBatch(1024))
}

// BenchmarkProfile_EmbeddingVec — single dense float32 vector.
// Recommended: OptQPack (Gorilla XOR catches smooth-varying floats;
// raw-LE bulk otherwise).
func BenchmarkProfile_EmbeddingVec(b *testing.B) {
	runProfile(b, "embed_768", qdf.OptQPack, mkEmbedding(768))
}

// BenchmarkProfile_Config — map-heavy stable shape, repeated keys
// across encodes (intern table). Recommended: OptBalanced.
func BenchmarkProfile_Config(b *testing.B) {
	runProfile(b, "config", qdf.OptBalanced, mkConfig())
}

// BenchmarkProfile_Archive — large heterogeneous payload, size is
// king. Recommended: OptCompression (alias for OptBalanced today).
func BenchmarkProfile_Archive(b *testing.B) {
	runProfile(b, "archive_5k", qdf.OptCompression, mkArchive(5000))
}

// BenchmarkProfile_SizesSummary writes a one-shot encode-size
// comparison per scenario into the testing log. Run with:
//
//	go test -C bench -run BenchmarkProfile_SizesSummary -v
//
// (It is a Test, not a Benchmark, despite the prefix — we keep
// the naming so the size column lives next to the latency
// columns in BENCH.md.)
func TestProfile_SizesSummary(t *testing.T) {
	type row struct {
		name string
		v    any
		opts qdf.Options
	}
	cases := []row{
		{"hot_path", mkHotPathEvent(), qdf.OptSpeed},
		{"telemetry_1k", mkTelemetryBatch(1000), qdf.OptBalanced},
		{"metric_1024", mkMetricSeries(1024), qdf.OptQPack},
		{"metric_smooth_qpack", mkMetricSeriesSmooth(1024), qdf.OptQPack},
		{"metric_smooth_zip", mkMetricSeriesSmooth(1024), qdf.OptCompression},
		{"status_1024", mkStatusBatch(1024), qdf.OptQPack},
		{"embed_768", mkEmbedding(768), qdf.OptQPack},
		{"config", mkConfig(), qdf.OptBalanced},
		{"archive_5k", mkArchive(5000), qdf.OptCompression},
	}
	t.Logf("%-14s %10s %10s %10s   ratio_vs_json", "scenario", "json", "msgpack", "qdf")
	for _, c := range cases {
		jb, _ := json.Marshal(c.v)
		mb, _ := msgpack.Marshal(c.v)
		qb, _ := qdf.Marshal(c.v, c.opts)
		t.Logf("%-14s %10d %10d %10d   %.3fx",
			c.name, len(jb), len(mb), len(qb),
			float64(len(qb))/float64(len(jb)))
	}
}
