package qdf

import (
	"reflect"
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
