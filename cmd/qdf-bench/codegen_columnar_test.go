package main

import (
	"bytes"
	"testing"

	"github.com/alex60217101990/qdf"
)

// GenService is a string-heavy struct; eligibility rejects its String fields, so
// its codegen must stay row-major (no columnar frame). Guards against a
// regression that would columnar-encode strings and bloat the wire.
func TestGenService_StaysRowMajor(t *testing.T) {
	svcs := make([]GenService, 32)
	for i := range svcs {
		svcs[i].Name = "service-name"
	}
	buf, err := qdf.Marshal(svcs, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("GenService unexpectedly emitted a columnar frame; strings must stay row-major")
	}
}

// A GenMetricHost wraps an all-scalar []GenMetric field, so its generated
// encode MUST take the columnar transpose path (0xEF frame). A bare
// []GenMetric, by contrast, is a Marshaler element and stays row-major — that
// is exactly why the columnar lever lives on the wrapper field.
func TestGenMetricHost_IsColumnar(t *testing.T) {
	ms := make([]GenMetric, 32)
	for i := range ms {
		ms[i] = GenMetric{TS: int64(i), CPU: float64(i), Mem: uint64(i), Errors: uint32(i), Up: true}
	}
	h := GenMetricHost{Host: "h", Metrics: ms}
	buf, err := qdf.Marshal(h, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("GenMetricHost.Metrics should be columnar-encoded (0xEF frame)")
	}
}

const benchMetricN = 2000

func benchMetricHosts() (MetricHost, GenMetricHost) {
	metrics := make([]Metric, benchMetricN)
	gms := make([]GenMetric, benchMetricN)
	for i := range metrics {
		metrics[i] = Metric{
			TS: int64(1_700_000_000 + i), CPU: float64(i%97) * 0.5,
			Mem: uint64(i%512) << 20, Errors: uint32(i % 7), Up: i%3 != 0,
		}
		gms[i] = GenMetric(metrics[i])
	}
	return MetricHost{Host: "host1", Metrics: metrics}, GenMetricHost{Host: "host1", Metrics: gms}
}

// TestColumnarCodegen_WireMatchesReflect asserts the codegen columnar wire is
// no larger than the reflect columnar baseline on the same numeric data — the
// whole point of Phase 2 (recover the columnar win for a generated type).
func TestColumnarCodegen_WireMatchesReflect(t *testing.T) {
	mh, gmh := benchMetricHosts()
	rb, err := qdf.Marshal(mh, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	cb, err := qdf.Marshal(gmh, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wire: reflect-columnar=%d  codegen-columnar=%d", len(rb), len(cb))
	if len(cb) > len(rb)+len(rb)/50 { // allow 2% slack (shape-id framing differences)
		t.Fatalf("codegen columnar wire %d exceeds reflect columnar %d by >2%%", len(cb), len(rb))
	}
	// And a bare []GenMetric must be much larger (row-major fallback) — proves
	// the wrapper is what unlocks the win.
	bare, err := qdf.Marshal(gmh.Metrics, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wire: bare []GenMetric (row-major)=%d", len(bare))
	if len(bare) <= len(cb) {
		t.Fatalf("bare []GenMetric %d should exceed columnar wrapper %d", len(bare), len(cb))
	}
}

func BenchmarkMetricHost_ReflectColumnar_Encode(b *testing.B) {
	mh, _ := benchMetricHosts()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = qdf.Marshal(mh, qdf.OptBalanced)
	}
}

func BenchmarkMetricHost_CodegenColumnar_Encode(b *testing.B) {
	_, gmh := benchMetricHosts()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = qdf.Marshal(gmh, qdf.OptBalanced)
	}
}

func BenchmarkMetricHost_ReflectColumnar_Decode(b *testing.B) {
	mh, _ := benchMetricHosts()
	buf, _ := qdf.Marshal(mh, qdf.OptBalanced)
	b.ReportAllocs()
	for b.Loop() {
		var out MetricHost
		_ = qdf.Unmarshal(buf, &out)
	}
}

func BenchmarkMetricHost_CodegenColumnar_Decode(b *testing.B) {
	_, gmh := benchMetricHosts()
	buf, _ := qdf.Marshal(gmh, qdf.OptBalanced)
	b.ReportAllocs()
	for b.Loop() {
		var out GenMetricHost
		_ = qdf.Unmarshal(buf, &out)
	}
}
