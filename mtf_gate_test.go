package qdf

import (
	"bytes"
	"fmt"
	"maps"
	"testing"
)

// TestMTFRanksStillReachTheWire pins that the encoder actually emits move-to-front
// ranks, which nothing else in the suite checks.
//
// This exists because a refactor deleted the MRU ring's only writer and silently
// disabled MTF on the encode side, and NOTHING caught it: not the round-trip
// tests (raw state-refs decode fine), not the 112-encoding wire digest, and not
// the benchmarks — which got FASTER, because not emitting ranks is cheaper than
// emitting them.
//
// The digest missed it for a specific reason worth writing down: emitStateRef
// takes a small-id fast path below 128 and never consults the ring there, and no
// digest fixture interns enough distinct strings to leave that range. So the
// fixture here is built to clear it: >128 distinct interned values, then repeats
// of the high ids, which is the only shape where a rank can be shorter than the
// id it replaces.
func TestMTFRanksStillReachTheWire(t *testing.T) {
	// A slice of structs would route these strings through the columnar
	// dictionary and never reach the intern path at all; reflect-driven maps
	// are what actually emit state refs.
	const distinct = 200 // comfortably past the 128-id fast path
	names := make([]string, distinct)
	kinds := make([]string, distinct)
	for i := range distinct {
		names[i] = fmt.Sprintf("com.acme.platform.service.instance.%06d", i)
		kinds[i] = fmt.Sprintf("workload-class-%06d", i)
	}

	rows := make([]map[string]string, 0, distinct*3)
	for i := range distinct {
		rows = append(rows, map[string]string{"name": names[i], "kind": kinds[i]})
	}
	// Revisit the high ids: a repeat of a recently-emitted high id is what MTF
	// encodes as a short rank.
	for pass := range 2 {
		for i := distinct - 1; i >= distinct/2; i-- {
			rows = append(rows, map[string]string{
				"name": names[i],
				"kind": kinds[(i+pass)%distinct],
			})
		}
	}

	wire, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(wire, []byte{tagStateMTF}) {
		t.Fatalf("no tagStateMTF (0x%02X) anywhere in %d bytes of wire: the encoder "+
			"is not emitting move-to-front ranks. Check that encState.mruPush still "+
			"has callers — mruRank answers from that ring, and an unfilled ring makes "+
			"every rank lookup miss and fall back to a raw state-ref.",
			tagStateMTF, len(wire))
	}

	// The ranks must also survive a round trip, so the gate cannot be satisfied
	// by emitting the tag incorrectly.
	var back []map[string]string
	if err := Unmarshal(wire, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back) != len(rows) {
		t.Fatalf("round trip changed length: got %d want %d", len(back), len(rows))
	}
	for i := range rows {
		if !maps.Equal(back[i], rows[i]) {
			t.Fatalf("row %d differs: got %+v want %+v", i, back[i], rows[i])
		}
	}

	// And MTF must genuinely be shortening the wire, not merely appearing in it.
	raw, err := Marshal(rows, OptBalanced&^OptMTF)
	if err != nil {
		t.Fatalf("marshal without MTF: %v", err)
	}
	if len(wire) >= len(raw) {
		t.Fatalf("MTF did not shrink the wire: %d bytes with it, %d without", len(wire), len(raw))
	}
	t.Logf("MTF present; wire %d bytes vs %d without it (-%.1f%%)",
		len(wire), len(raw), 100*float64(len(raw)-len(wire))/float64(len(raw)))
}
