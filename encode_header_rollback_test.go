package qdf

import (
	"math"
	"testing"
)

// TestGorillaRollbackKeepsHeader pins the fix for the header-latch bug: when a
// top-level float64 slice attempts Gorilla (which lazily writes the stream
// header), loses the never-larger gate, and rolls back, the fallback codec must
// still emit the header. Before the fix the rollback truncated the header bytes
// but left e.headerOut latched, so writeHeader became a no-op and the output was
// a headerless, undecodable stream ("bad magic" on Unmarshal).
func TestGorillaRollbackKeepsHeader(t *testing.T) {
	// 65 identical float64 values whose bits are 0x4141414141414141: Gorilla is
	// attempted but ALP/raw wins, forcing the rollback path. (Exact repro of the
	// FuzzRoundTrip_Float64Slice crasher.)
	in := make([]float64, 65)
	for i := range in {
		in[i] = math.Float64frombits(0x4141414141414141)
	}
	for _, opts := range []Options{OptQPack | OptGorillaFloat, OptCompression} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		if len(buf) < 5 || buf[0] != Magic0 || buf[1] != Magic1 || buf[2] != Magic2 {
			t.Fatalf("opts=%d: output missing magic header, head=% x", opts, buf[:min(8, len(buf))])
		}
		var out []float64
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("opts=%d unmarshal: %v", opts, err)
		}
		if len(out) != len(in) {
			t.Fatalf("opts=%d len=%d want %d", opts, len(out), len(in))
		}
		for i := range in {
			if math.Float64bits(out[i]) != math.Float64bits(in[i]) {
				t.Fatalf("opts=%d [%d] = %x want %x", opts, i, math.Float64bits(out[i]), math.Float64bits(in[i]))
			}
		}
	}
}

// TestGorillaRollbackNestedHeaderUnaffected guards the other side: when the
// header was ALREADY written before the float slice (a struct field, so the
// rollback must NOT clear a header that lives before `start`), the round-trip
// still holds.
func TestGorillaRollbackNestedHeaderUnaffected(t *testing.T) {
	type row struct {
		Tag string    `qdf:"tag"`
		V   []float64 `qdf:"v"`
	}
	in := row{Tag: "metrics"}
	in.V = make([]float64, 65)
	for i := range in.V {
		in.V[i] = math.Float64frombits(0x4141414141414141)
	}
	buf, err := Marshal(in, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var out row
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.Tag != in.Tag || len(out.V) != len(in.V) {
		t.Fatalf("mismatch: %+v", out)
	}
	for i := range in.V {
		if math.Float64bits(out.V[i]) != math.Float64bits(in.V[i]) {
			t.Fatalf("V[%d] mismatch", i)
		}
	}
}
