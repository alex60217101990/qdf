package qdf

import (
	"strconv"
	"testing"
)

// A field where the delta never wins should stop being probed — the prefix
// compare is real work on data it cannot help. But the gate must RE-ARM: a
// field whose data turns compressible part-way through has to be recovered, or
// the gate becomes a guess that costs wire, which is the one thing this design
// refuses.
func TestStrDeltaGateRearms(t *testing.T) {
	const n = 4096
	rows := make([]sdRow, n)
	for i := range rows {
		if i < n/2 {
			// Nothing shared: the delta loses every time and the gate mutes.
			rows[i] = sdRow{Seq: int64(i), URL: strconv.Itoa(i*2654435761) + "-" + strconv.Itoa(i*40503)}
		} else {
			// Long shared prefixes: the gate must have re-armed to see them.
			rows[i] = sdRow{Seq: int64(i), URL: "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+i)}
		}
	}
	before := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if fired := strDeltaEmitted.Load() - before; fired == 0 {
		t.Fatal("gate never re-armed: the compressible second half emitted nothing")
	}
	var got []sdRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d: got %+v want %+v", i, got[i], rows[i])
		}
	}
}

// The gate's effect is invisible in emission counts — a field the delta cannot
// help emits nothing with or without it. Only the probe count shows it: on
// prefix-free data the comparison must stop being spent.
//
// The values below are multiplicative-hash hex, so their leading characters
// scramble and consecutive rows share nothing. Sequential decimal integers
// would NOT do: "1234567890" and "1234567891" share nine bytes, and the delta
// legitimately wins on them.
func TestStrDeltaGateStopsProbingHopelessFields(t *testing.T) {
	const n = 4096
	rows := make([]sdRow, n)
	for i := range rows {
		h := uint64(i+1) * 11400714819323198485
		rows[i] = sdRow{Seq: int64(i), URL: strconv.FormatUint(h, 16) + strconv.FormatUint(h>>17, 16)}
	}
	beforeP := strDeltaProbes.Load()
	beforeE := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	probes := strDeltaProbes.Load() - beforeP
	fired := strDeltaEmitted.Load() - beforeE
	t.Logf("prefix-free field: %d values, %d probes, %d emissions, wire=%d", n, probes, fired, len(b))
	// Muted after the first 32 with no win, re-armed every 512: the ceiling is
	// roughly n/strDeltaRearmN probe windows. Allow generous slack; what must
	// not happen is a probe per value.
	if probes > int64(n/4) {
		t.Fatalf("gate never muted: %d probes for %d values", probes, n)
	}
	var got []sdRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d mismatch", i)
		}
	}
}
