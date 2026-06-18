package qdf

import "testing"

// TestThreadedSliceDecodeAllocs decodes a slice whose element type implements
// DecoderUnmarshaler through the reflect path. With threading, the whole slice
// shares one decoder instead of allocating a fresh decoder (plus its scratch
// state) per element. Reuses threadProbe from marshaler_decodethread_test.go.
func TestThreadedSliceDecodeAllocs(t *testing.T) {
	const n = 200
	in := make([]*threadProbe, n)
	for i := range in {
		in[i] = &threadProbe{V: int64(i)}
	}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out []*threadProbe
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != n || out[n-1].V != int64(n-1) {
		t.Fatalf("round-trip: len=%d", len(out))
	}
	allocs := testing.AllocsPerRun(50, func() {
		var o []*threadProbe
		_ = Unmarshal(buf, &o)
	})
	// Threaded: one shared decoder for the whole slice, so allocs are ~one result
	// struct per element plus a small constant (~n). The fresh-decoder-per-element
	// path adds a decoder per element (~2n). A ceiling below 2n fails if threading
	// regresses; n + n/2 leaves headroom for the result structs + slice growth.
	if allocs > float64(n+n/2) {
		t.Fatalf("decode allocs=%.0f (>%d) — per-element decoder not eliminated (threading regressed)", allocs, n+n/2)
	}
}
