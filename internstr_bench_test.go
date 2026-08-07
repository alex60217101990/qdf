package qdf

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

// Three payload shapes drawn from real traffic rather than synthetic noise.
// They exist so every measurement in this work item is taken against the same
// inputs: log rows (shared-prefix paths, a small service enum, free text),
// telemetry (low-cardinality enum strings beside numeric columns), and an API
// payload (dynamic map keys beside free text).

type logProfileRow struct {
	Path    string
	Service string
	Msg     string
	Code    int64
}

type telemetryProfileRow struct {
	Device string
	Region string
	Metric string
	Value  float64
	TS     int64
}

type apiProfileRow struct {
	Attrs map[string]string
	Body  string
	ID    int64
}

func mkLogProfile(n int) []logProfileRow {
	svc := []string{"api-gateway", "auth-service", "billing-service", "search-service"}
	rows := make([]logProfileRow, n)
	for i := range rows {
		rows[i] = logProfileRow{
			Path:    "/api/v2/tenant/" + strconv.Itoa(i%64) + "/orders/" + strconv.Itoa(i),
			Service: svc[i%len(svc)],
			Msg:     "request completed with status " + strconv.Itoa(200+i%5),
			Code:    int64(200 + i%5),
		}
	}
	return rows
}

func mkTelemetryProfile(n int) []telemetryProfileRow {
	region := []string{"eu-central-1", "us-east-1", "ap-south-1"}
	metric := []string{"cpu.util", "mem.rss", "disk.io", "net.rx"}
	rows := make([]telemetryProfileRow, n)
	for i := range rows {
		rows[i] = telemetryProfileRow{
			Device: "device-" + strconv.Itoa(i%128),
			Region: region[i%len(region)],
			Metric: metric[i%len(metric)],
			Value:  float64(i%1000) * 0.5,
			TS:     int64(1700000000 + i),
		}
	}
	return rows
}

func mkAPIProfile(n int) []apiProfileRow {
	rows := make([]apiProfileRow, n)
	for i := range rows {
		rows[i] = apiProfileRow{
			Attrs: map[string]string{
				"tenant": "tenant-" + strconv.Itoa(i%32),
				"env":    "production",
				"region": "eu-central-1",
			},
			Body: "order " + strconv.Itoa(i) + " accepted for processing",
			ID:   int64(i),
		}
	}
	return rows
}

// assertProfilesRoundTrip is the correctness gate every later task re-runs:
// each profile, under each option set, must decode back to exactly the input.
func assertProfilesRoundTrip(t *testing.T) {
	t.Helper()
	opts := []Options{OptSpeed, OptBalanced, OptQPack, OptCompression}
	for _, o := range opts {
		logs := mkLogProfile(512)
		blob, err := Marshal(logs, o)
		if err != nil {
			t.Fatalf("log/%v encode: %v", o, err)
		}
		var gotLogs []logProfileRow
		if err := Unmarshal(blob, &gotLogs); err != nil {
			t.Fatalf("log/%v decode: %v", o, err)
		}
		if !reflect.DeepEqual(logs, gotLogs) {
			t.Fatalf("log/%v round-trip mismatch", o)
		}

		tel := mkTelemetryProfile(512)
		blob, err = Marshal(tel, o)
		if err != nil {
			t.Fatalf("telemetry/%v encode: %v", o, err)
		}
		var gotTel []telemetryProfileRow
		if err := Unmarshal(blob, &gotTel); err != nil {
			t.Fatalf("telemetry/%v decode: %v", o, err)
		}
		if !reflect.DeepEqual(tel, gotTel) {
			t.Fatalf("telemetry/%v round-trip mismatch", o)
		}

		api := mkAPIProfile(256)
		blob, err = Marshal(api, o)
		if err != nil {
			t.Fatalf("api/%v encode: %v", o, err)
		}
		var gotAPI []apiProfileRow
		if err := Unmarshal(blob, &gotAPI); err != nil {
			t.Fatalf("api/%v decode: %v", o, err)
		}
		if !reflect.DeepEqual(api, gotAPI) {
			t.Fatalf("api/%v round-trip mismatch", o)
		}
	}
}

func TestProfilesRoundTrip(t *testing.T) {
	assertProfilesRoundTrip(t)
}

func BenchmarkProfileEncode(b *testing.B) {
	cases := []struct {
		name string
		v    any
	}{
		{"log", mkLogProfile(4096)},
		{"telemetry", mkTelemetryProfile(4096)},
		{"api", mkAPIProfile(2048)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Marshal(c.v, OptBalanced); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkProfileDecode(b *testing.B) {
	logs := mkLogProfile(4096)
	tel := mkTelemetryProfile(4096)
	api := mkAPIProfile(2048)
	logBlob, err := Marshal(logs, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	telBlob, err := Marshal(tel, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	apiBlob, err := Marshal(api, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("log", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var out []logProfileRow
			if err := Unmarshal(logBlob, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("telemetry", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var out []telemetryProfileRow
			if err := Unmarshal(telBlob, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("api", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var out []apiProfileRow
			if err := Unmarshal(apiBlob, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// heapInuseAfter runs f the given number of times through pooled encoders and
// decoders, then reports HeapInuse after two collections. Two are needed: the
// first makes the garbage unreachable, the second frees what the first
// discovered. B/op answers "what did one call allocate"; this answers "what
// does the process still hold", which is the question the retention thresholds
// were written for.
func heapInuseAfter(iters int, f func()) uint64 {
	for i := 0; i < iters; i++ {
		f()
	}
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapInuse
}

// BenchmarkProfileRSS reports resident memory as a benchmark metric rather
// than a test log: benchstat can then compare it across runs the same way it
// compares ns/op, and there is no assertion-free test pretending to be a gate.
// Each sub-benchmark runs its shape once per iteration and reports the bytes
// the process still holds afterwards.
func BenchmarkProfileRSS(b *testing.B) {
	logs := mkLogProfile(4096)
	b.Run("steady", func(b *testing.B) {
		var last uint64
		for i := 0; i < b.N; i++ {
			last = heapInuseAfter(10000, func() {
				blob, err := Marshal(logs, OptBalanced)
				if err != nil {
					b.Fatal(err)
				}
				var out []logProfileRow
				if err := Unmarshal(blob, &out); err != nil {
					b.Fatal(err)
				}
			})
		}
		b.ReportMetric(float64(last), "heap_inuse_B")
	})

	// Burst shape: the case the retention thresholds exist for — a few large
	// messages followed by many small ones. A pooled encoder that keeps the
	// large scratch shows up here and nowhere else.
	big := mkLogProfile(65536)
	small := mkLogProfile(16)
	b.Run("burst", func(b *testing.B) {
		var last uint64
		for i := 0; i < b.N; i++ {
			last = heapInuseAfter(1, func() {
				for j := 0; j < 100; j++ {
					blob, err := Marshal(big, OptBalanced)
					if err != nil {
						b.Fatal(err)
					}
					var out []logProfileRow
					if err := Unmarshal(blob, &out); err != nil {
						b.Fatal(err)
					}
				}
				for j := 0; j < 10000; j++ {
					blob, err := Marshal(small, OptBalanced)
					if err != nil {
						b.Fatal(err)
					}
					var out []logProfileRow
					if err := Unmarshal(blob, &out); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
		b.ReportMetric(float64(last), "heap_inuse_B")
	})
}

// The index must answer for a string it recorded, must not answer for a
// different string, and must answer for a substring taken at the same offset
// and length — which is the same bytes, sharing the parent's backing array.
func TestPtrInternIndex(t *testing.T) {
	st := newEncState()
	parent := "abcdefghijklmnop"
	a := parent[2:8]

	if _, ok := st.ptrLookup(a); ok {
		t.Fatal("empty index answered")
	}
	st.ptrRecord(a, 7)
	got, ok := st.ptrLookup(a)
	if !ok || got != 7 {
		t.Fatalf("recorded string: got (%d,%v), want (7,true)", got, ok)
	}

	// Same bytes, same backing, same offset and length: must hit.
	again := parent[2:8]
	got, ok = st.ptrLookup(again)
	if !ok || got != 7 {
		t.Fatalf("identical substring: got (%d,%v), want (7,true)", got, ok)
	}

	// Same backing, different length: different bytes, must miss.
	if _, ok := st.ptrLookup(parent[2:9]); ok {
		t.Fatal("different length answered")
	}
	// Same length, different offset: different bytes, must miss.
	if _, ok := st.ptrLookup(parent[3:9]); ok {
		t.Fatal("different offset answered")
	}
	// Equal content, different backing: allowed to miss, must not answer wrong.
	other := string([]byte(a))
	if got, ok := st.ptrLookup(other); ok && got != 7 {
		t.Fatalf("equal-content copy answered with the wrong id %d", got)
	}

	st.ptrIndexReset()
	if _, ok := st.ptrLookup(a); ok {
		t.Fatal("index answered after reset")
	}
}
