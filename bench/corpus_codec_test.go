package bench

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// This file exercises the adaptive codec set (RLE, Gorilla XOR,
// Delta+FOR, FOR, bitpacked bool, intern, shape interning) against
// production-shaped corpora — not single-column micro fixtures.
// Each fixture is a mix of column types where different codecs win on
// different fields, so the bench reflects what a real-world payload
// looks like end-to-end.

// webRequestColumn matches the column layout an ingest pipeline ships
// after pulling apart per-request structured logs. Columns get long
// stretches of repeated values (status_code, level, service, region)
// and a smooth float gauge (latency p95) — three independent codec
// wins in one payload.
type webRequestColumn struct {
	Service    string    `qdf:"service"     json:"service"     msgpack:"service"`
	Region     string    `qdf:"region"      json:"region"      msgpack:"region"`
	Timestamps []int64   `qdf:"timestamps"  json:"timestamps"  msgpack:"timestamps"`
	StatusCode []int64   `qdf:"status_code" json:"status_code" msgpack:"status_code"`
	LevelInt   []int64   `qdf:"level_int"   json:"level_int"   msgpack:"level_int"`
	LatencyP95 []float64 `qdf:"latency_p95" json:"latency_p95" msgpack:"latency_p95"`
	Errors     []bool    `qdf:"errors"      json:"errors"      msgpack:"errors"`
}

func mkWebRequestColumn(n int) webRequestColumn {
	rnd := rand.New(rand.NewSource(101))
	ts := make([]int64, n)
	status := make([]int64, n)
	level := make([]int64, n)
	latency := make([]float64, n)
	errs := make([]bool, n)
	for i := range n {
		ts[i] = 1_700_000_000 + int64(i)
	}
	// status: burst pattern → RLE win.
	idx := 0
	for idx < n {
		runLen := 80 + rnd.Intn(120)
		if idx+runLen > n {
			runLen = n - idx
		}
		for i := 0; i < runLen; i++ {
			status[idx+i] = 200
		}
		idx += runLen
		if idx >= n {
			break
		}
		fault := 3 + rnd.Intn(12)
		code := int64(500)
		if rnd.Float64() < 0.5 {
			code = 404
		}
		if idx+fault > n {
			fault = n - idx
		}
		for i := 0; i < fault; i++ {
			status[idx+i] = code
		}
		idx += fault
	}
	// level enum: 0=debug,1=info,2=warn,3=error. info dominates,
	// occasional warn/error → RLE-friendly.
	for i := range n {
		r := rnd.Float64()
		switch {
		case r < 0.92:
			level[i] = 1
		case r < 0.98:
			level[i] = 2
		default:
			level[i] = 3
		}
	}
	// latency p95: smooth gauge with ±5 ms steps and 60 % repeats.
	value := 47.5
	for i := range n {
		if i > 0 && rnd.Float64() < 0.6 {
			latency[i] = value
			continue
		}
		if rnd.Float64() < 0.5 {
			value += 5
		} else {
			value -= 5
		}
		if value < 1 {
			value = 1
		}
		latency[i] = value
	}
	// errors: false dominates with occasional true runs → bitpack
	// already wins; we still record it so the slice shape is real.
	for i := range n {
		errs[i] = status[i] >= 400
	}
	return webRequestColumn{
		Service:    "auth-gateway",
		Region:     "eu-west-1",
		Timestamps: ts,
		StatusCode: status,
		LevelInt:   level,
		LatencyP95: latency,
		Errors:     errs,
	}
}

// counterSnapshot models a typical metrics export: timestamps,
// monotonically-non-decreasing counters that mostly hold steady,
// and a parallel sparse counter that sits at zero for long stretches.
// Hits Delta+FOR on the monotonic series and RLE on the sparse one.
type counterSnapshot struct {
	Series    string  `qdf:"series"    json:"series"    msgpack:"series"`
	Timestamp []int64 `qdf:"timestamp" json:"timestamp" msgpack:"timestamp"`
	Counter   []int64 `qdf:"counter"   json:"counter"   msgpack:"counter"`
	Sparse    []int64 `qdf:"sparse"    json:"sparse"    msgpack:"sparse"`
}

func mkCounterSnapshot(n int) counterSnapshot {
	rnd := rand.New(rand.NewSource(202))
	ts := make([]int64, n)
	ctr := make([]int64, n)
	sparse := make([]int64, n)
	cur := int64(0)
	for i := range n {
		ts[i] = 1_700_000_000 + int64(i)*60
		// counter accumulates 0-2 hits per sample with a slight bias
		// toward holding steady — Delta+FOR shines.
		if rnd.Float64() < 0.7 {
			cur += int64(rnd.Intn(3))
		}
		ctr[i] = cur
		// sparse: 0 for long stretches with occasional 1-3 bumps.
		if rnd.Float64() < 0.95 {
			sparse[i] = 0
		} else {
			sparse[i] = int64(1 + rnd.Intn(3))
		}
	}
	return counterSnapshot{
		Series:    "http.requests.total",
		Timestamp: ts,
		Counter:   ctr,
		Sparse:    sparse,
	}
}

// spreadEnumColumn is the shape that pins the dictionary codec: a
// small set of distinct values (≤ qpackDictMaxDistinct) spread far
// enough apart that FOR can't bit-pack them densely, and arranged
// randomly so RLE can't fold runs either. This is what shows up when
// a column carries a categorical id encoded as a wide int (priority
// buckets at 10^k, latency buckets in microseconds, sensor reading
// snap-to-quantum where the quanta are spread out).
type spreadEnumColumn struct {
	Field  string  `qdf:"field"  json:"field"  msgpack:"field"`
	Values []int64 `qdf:"values" json:"values" msgpack:"values"`
}

func mkSpreadEnumColumn(n int) spreadEnumColumn {
	rnd := rand.New(rand.NewSource(404))
	// 4 distinct values, range ≈ 2 million → FOR bitsPer = 21, dict
	// bitsPer = 2. Random distribution kills RLE.
	values := []int64{200, 999_999, 12345, -1_000_000}
	v := make([]int64, n)
	for i := range n {
		v[i] = values[rnd.Intn(len(values))]
	}
	return spreadEnumColumn{Field: "priority_bucket", Values: v}
}

// tracesBatch is a span-batch shaped like an APM trace export. Long
// runs of the same operation / service exercise intern + shape; the
// duration column is a smooth gauge (Gorilla territory under
// OptCompression).
type spanRow struct {
	TraceID    string  `qdf:"trace_id"  json:"trace_id"  msgpack:"trace_id"`
	SpanID     string  `qdf:"span_id"   json:"span_id"   msgpack:"span_id"`
	Service    string  `qdf:"service"   json:"service"   msgpack:"service"`
	Operation  string  `qdf:"operation" json:"operation" msgpack:"operation"`
	DurationMs float64 `qdf:"dur_ms"    json:"dur_ms"    msgpack:"dur_ms"`
}

func mkTracesBatch(n int) []spanRow {
	services := []string{"gateway", "billing", "auth"}
	ops := []string{"/v1/charge", "/v1/login", "/v1/me", "/v1/list"}
	rnd := rand.New(rand.NewSource(303))
	rows := make([]spanRow, n)
	for i := range n {
		rows[i] = spanRow{
			TraceID:    fmt.Sprintf("%032x", rnd.Uint64()),
			SpanID:     fmt.Sprintf("%016x", rnd.Uint64()),
			Service:    services[rnd.Intn(len(services))],
			Operation:  ops[rnd.Intn(len(ops))],
			DurationMs: 0.5 + rnd.Float64()*40,
		}
	}
	return rows
}

// TestCorpusCodec_Sizes prints a wire-size table covering the new
// corpus across OptSpeed / OptBalanced / OptCompression. Run with
// `go test -C bench -run TestCorpusCodec_Sizes -v` and copy the
// table into BENCH.md when codec work lands.
func TestCorpusCodec_Sizes(t *testing.T) {
	type fx struct {
		name string
		v    any
	}
	cases := []fx{
		{"webreq_1024", mkWebRequestColumn(1024)},
		{"counters_1024", mkCounterSnapshot(1024)},
		{"traces_500", mkTracesBatch(500)},
		{"metric_smooth_1024", mkMetricSeriesSmooth(1024)},
		{"status_1024", mkStatusBatch(1024)},
		{"spread_enum_1024", mkSpreadEnumColumn(1024)},
		{"logentries_1024", logEntriesBatch(1024)},
		{"events_1024", eventsBatch(1024)},
	}
	t.Logf("%-22s %10s %10s %10s %12s %12s",
		"scenario", "json", "msgpack", "qdf_speed", "qdf_balanced", "qdf_compress")
	for _, c := range cases {
		jb, _ := json.Marshal(c.v)
		mb, _ := msgpack.Marshal(c.v)
		qs, _ := qdf.Marshal(c.v, qdf.OptSpeed)
		qb, _ := qdf.Marshal(c.v, qdf.OptBalanced)
		qc, _ := qdf.Marshal(c.v, qdf.OptCompression)
		t.Logf("%-22s %10d %10d %10d %12d %12d",
			c.name, len(jb), len(mb), len(qs), len(qb), len(qc))
	}
}

// benchLogEntry is a structured-log row where Level/Service/Host repeat
// heavily row-to-row (exercises the column-conditional repeat predictor),
// while Msg and TraceID are unique per row.
type benchLogEntry struct {
	Level   string `qdf:"level"    json:"level"    msgpack:"level"`
	Service string `qdf:"service"  json:"service"  msgpack:"service"`
	Host    string `qdf:"host"     json:"host"     msgpack:"host"`
	Msg     string `qdf:"msg"      json:"msg"      msgpack:"msg"`
	TraceID string `qdf:"trace_id" json:"trace_id" msgpack:"trace_id"`
}

// logEntriesBatch models a structured-log batch: Level/Service/Host repeat
// heavily row-to-row (exercises the column-conditional repeat predictor),
// while Msg and TraceID are unique per row.
func logEntriesBatch(n int) []benchLogEntry {
	levels := []string{"INFO", "INFO", "INFO", "WARN", "ERROR"}
	services := []string{"api-gateway", "auth", "billing"}
	hosts := []string{"node-1", "node-2", "node-3", "node-4"}
	out := make([]benchLogEntry, n)
	for i := range out {
		out[i] = benchLogEntry{
			Level:   levels[i%len(levels)],
			Service: services[i%len(services)],
			Host:    hosts[i%len(hosts)],
			Msg:     "handled request id=" + strconv.Itoa(i),
			TraceID: "trace-" + strconv.Itoa(i*2654435761&0xffffff),
		}
	}
	return out
}

// BenchmarkCorpusCodec_WebRequest exercises the most codec-diverse
// fixture: status RLE + level RLE + timestamps Delta+FOR + latency
// raw under OptBalanced, latency Gorilla under OptCompression. The
// service/region strings ride the intern table.
func BenchmarkCorpusCodec_WebRequest(b *testing.B) {
	runProfile(b, "webreq_1024", qdf.OptBalanced, mkWebRequestColumn(1024))
}

// BenchmarkCorpusCodec_Counters exercises the monotonic (Delta+FOR)
// and sparse-zero (RLE) integer columns side-by-side.
func BenchmarkCorpusCodec_Counters(b *testing.B) {
	runProfile(b, "counters_1024", qdf.OptQPack, mkCounterSnapshot(1024))
}

// BenchmarkCorpusCodec_SpreadEnum exercises the dictionary codec:
// 4 distinct values spread across a wide range, with no runs to
// fold. The picker should select qpackDict and the body should
// collapse to ~2 bits per element.
func BenchmarkCorpusCodec_SpreadEnum(b *testing.B) {
	runProfile(b, "spread_enum_1024", qdf.OptQPack, mkSpreadEnumColumn(1024))
}

// BenchmarkCorpusCodec_Traces exercises shape interning + heavy
// string intern on the service / operation columns, plus a float
// duration column. The trace/span IDs are unique each row — intern
// hit rate stays low for those, so the test also pins that the
// intern table does not pay an outsized cost when most strings are
// unique.
func BenchmarkCorpusCodec_Traces(b *testing.B) {
	runProfile(b, "traces_500", qdf.OptBalanced, mkTracesBatch(500))
}

// benchEvent is a structured-event row where Level/Service repeat heavily
// row-to-row, Code is a small enum-like set, LatencyMs clusters in a
// tight window, and OK is a high-cardinality boolean column. All numeric
// fields are columnar-friendly: Code and LatencyMs hit FOR/delta/dict
// codecs they never reached in row-major encoding.
type benchEvent struct {
	Level     string `qdf:"level"      json:"level"      msgpack:"level"`
	Service   string `qdf:"service"    json:"service"    msgpack:"service"`
	Code      int    `qdf:"code"       json:"code"       msgpack:"code"`
	LatencyMs int    `qdf:"latency_ms" json:"latency_ms" msgpack:"latency_ms"`
	OK        bool   `qdf:"ok"         json:"ok"         msgpack:"ok"`
}

// eventsBatch models an event/log batch: Level/Service repeat row-to-row,
// Code is a small enum-like set, LatencyMs clusters, OK is mostly true.
// All-columnar-friendly: exercises the columnar []struct codec.
func eventsBatch(n int) []benchEvent {
	levels := []string{"INFO", "INFO", "INFO", "WARN", "ERROR"}
	services := []string{"api-gateway", "auth", "billing"}
	codes := []int{200, 200, 200, 404, 500}
	out := make([]benchEvent, n)
	for i := range out {
		out[i] = benchEvent{
			Level:     levels[i%len(levels)],
			Service:   services[i%len(services)],
			Code:      codes[i%len(codes)],
			LatencyMs: 10 + (i % 40),
			OK:        i%7 != 0,
		}
	}
	return out
}

// Reference timer to keep the file from accidentally depending on
// time package only via fixtures.
var _ = time.Now
