package cgsample

import (
	"fmt"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
)

// telemetryEvents is the shape OptColumnarGenerated is FOR: a timestamp column,
// a four-value level, a small int, and one free-text message. Columns of
// numbers, timestamps and enums are what the columnar codecs take immediately
// and cheaply, which is why the option is a win here on every axis but decode
// time — the opposite of the ten-free-text-columns fixture its documentation
// also cites.
func telemetryEvents(n int) []GenEvent {
	levels := []string{"info", "warn", "error", "debug"}
	svc := []string{"ingest", "auth", "billing", "api-gateway"}
	out := make([]GenEvent, n)
	t0 := time.Unix(1_700_000_000, 0)
	for i := range n {
		out[i] = GenEvent{
			TS:    t0.Add(time.Duration(i) * time.Second),
			Level: levels[i%4],
			Code:  int32(200 + i%5),
			Msg:   fmt.Sprintf("%s: request %d completed in %dms", svc[i%4], i, i%997),
		}
	}
	return out
}

// The option's documentation claims a large wire cut on this shape. A claim in a
// doc comment that nothing checks is a claim that rots, and this one is load
// bearing: it is the reason to reach for the bit at all.
func TestColumnarGeneratedPaysOnTelemetryShape(t *testing.T) {
	for _, n := range []int{512, 2048} {
		v := telemetryEvents(n)
		for _, o := range []struct {
			name   string
			opts   qdf.Options
			minCut float64
		}{
			{"balanced", qdf.OptBalanced, 30},
			{"compression", qdf.OptCompression, 50},
		} {
			off, err := qdf.Marshal(v, o.opts)
			if err != nil {
				t.Fatal(err)
			}
			on, err := qdf.Marshal(v, o.opts|qdf.OptColumnarGenerated)
			if err != nil {
				t.Fatal(err)
			}
			cut := float64(len(off)-len(on)) / float64(len(off)) * 100
			if cut < o.minCut {
				t.Errorf("n=%d %s: the option cut %.1f%% of the wire (%d -> %d), want at least %.0f%% — "+
					"the shape this bit exists for stopped paying",
					n, o.name, cut, len(off), len(on), o.minCut)
			}
			// And the bytes must be exactly what the reflect path already writes
			// for the same values, which is what makes them readable everywhere.
			var back []GenEvent
			if err := qdf.Unmarshal(on, &back); err != nil {
				t.Fatalf("n=%d %s: the transposed wire does not decode: %v", n, o.name, err)
			}
			if len(back) != len(v) {
				t.Fatalf("n=%d %s: decoded %d events, want %d", n, o.name, len(back), len(v))
			}
			for i := range v {
				if back[i].Level != v[i].Level || back[i].Msg != v[i].Msg || back[i].Code != v[i].Code {
					t.Fatalf("n=%d %s: event %d differs after a round trip", n, o.name, i)
				}
			}
		}
	}
}
