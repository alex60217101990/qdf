package qdf

// Regression guards for the deltaScratch reuse in readPackedALPFloat64Slice
// (qpack_alp.go) and readStringColumnDict (qpack_strdict.go). Both switched a
// per-call make([]uint64, n) to the shared d.deltaScratch. The hazard class is
// reused-buffer under-fill / stale-data leak: if the bit-unpack did not fully
// overwrite the n reused slots, prior-call garbage would surface as wrong
// decoded values. These tests PRE-DIRTY deltaScratch with a sentinel and assert
// the decode is still bit-exact, turning an order-dependent heisenbug into a
// deterministic check.

import (
	"math"
	"testing"
)

const scratchDirt = uint64(0xDEADBEEFDEADBEEF)

func dirtyScratch(d *Decoder, n int) {
	d.deltaScratch = make([]uint64, n)
	for i := range d.deltaScratch {
		d.deltaScratch[i] = scratchDirt
	}
}

func TestDeltaScratchReuse_ALPNoStaleLeak(t *testing.T) {
	for _, s := range [][]float64{
		alpFixtureQuantized(),
		alpFixtureSmoothQuantized(),
	} {
		plan, _, ok := alpPlanFloat64(s)
		if !ok {
			continue
		}
		e := NewEncoderWith(OptCompression)
		e.writePackedALPFloat64Slice(s, plan)
		payload := e.buf[5:]

		d := &Decoder{buf: payload}
		dirtyScratch(d, len(s)+128) // larger than n, all sentinel
		d.i = 1
		got, err := d.readPackedALPFloat64Slice()
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if len(got) != len(s) {
			t.Fatalf("len %d != %d", len(got), len(s))
		}
		for i := range s {
			if math.Float64bits(got[i]) != math.Float64bits(s[i]) {
				t.Fatalf("idx %d: stale-scratch leak — got %#x want %#x",
					i, math.Float64bits(got[i]), math.Float64bits(s[i]))
			}
		}
	}
}

func TestDeltaScratchReuse_StrDictNoStaleLeak(t *testing.T) {
	// Low-cardinality strings so the dictionary codec fires (count >= 2).
	pool := []string{"alpha", "bravo", "charlie", "delta"}
	n := 1500
	strs := make([]string, n)
	for i := range strs {
		strs[i] = pool[(i*7)%len(pool)]
	}

	e := NewEncoderWith(OptBalanced &^ OptRANS)
	if !e.tryWriteStringColumnDict(strs) {
		t.Skip("dict codec declined for this input")
	}

	// tryWriteStringColumnDict calls writeHeader first (5-byte QDF header), so
	// the tagColStrDict lands at offset 5 — land the cursor on the tag.
	d := &Decoder{buf: e.buf, i: 5}
	dirtyScratch(d, n+128) // sentinel-fill before decode
	table, idx, err := d.readStringColumnDict(n)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(idx) != n {
		t.Fatalf("idx len %d != %d", len(idx), n)
	}
	for i := range strs {
		if int(idx[i]) >= len(table) || table[idx[i]] != strs[i] {
			t.Fatalf("idx %d: stale-scratch leak — got %q want %q",
				i, table[idx[i]], strs[i])
		}
	}
}
