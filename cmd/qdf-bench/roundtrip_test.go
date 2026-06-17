package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// sampleFile returns a localmachine JSON path: QDF_BENCH_SAMPLE or a dev default.
func sampleFile(t *testing.T) string {
	t.Helper()
	p := os.Getenv("QDF_BENCH_SAMPLE")
	if p == "" {
		p = "/tmp/lm.json"
	}
	if _, err := os.Stat(p); err != nil {
		t.Skipf("no sample file at %s (set QDF_BENCH_SAMPLE)", p)
	}
	return p
}

func TestRoundtripTyped(t *testing.T) {
	info, err := loadTyped(sampleFile(t))
	if err != nil {
		t.Fatalf("loadTyped: %v", err)
	}
	b, err := qdf.MarshalT(*info, qdf.OptBalanced)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Info
	if err := qdf.UnmarshalT(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*info, got) {
		t.Fatal("typed roundtrip mismatch")
	}
	t.Logf("typed: services=%d tasks=%d wire=%d bytes", len(info.Services), len(info.Tasks), len(b))
}

func TestRoundtripMap(t *testing.T) {
	m, err := loadMap(sampleFile(t))
	if err != nil {
		t.Fatalf("loadMap: %v", err)
	}
	b, err := qdf.Marshal(m, qdf.OptBalanced)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := qdf.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(m, got) {
		t.Fatal("map roundtrip mismatch")
	}
	t.Logf("map: keys=%d wire=%d bytes", len(m), len(b))
}
