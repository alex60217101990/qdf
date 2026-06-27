package qdf

import (
	"bytes"
	"math/rand"
	"testing"
)

type zcRow struct {
	TS int64  // ordered predicate column
	U  uint64 // ordered uint column
	V  int64  // payload
}

func zcRows(n int, ordered bool) []zcRow {
	rng := rand.New(rand.NewSource(5))
	rows := make([]zcRow, n)
	ts := int64(1_700_000_000)
	for i := range rows {
		if ordered {
			ts += 1 + rng.Int63n(3)
		} else {
			ts = rng.Int63n(1 << 40)
		}
		rows[i] = zcRow{TS: ts, U: uint64(i) * 7, V: int64(i)}
	}
	return rows
}

// ---- round-trip (decode-all) ----

func TestZoneChunkRoundtrip(t *testing.T) {
	for _, ordered := range []bool{true, false} {
		for _, n := range []int{511, 512, 513, 1000, 4096, 4097} {
			rows := zcRows(n, ordered)
			b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
			if err != nil {
				t.Fatalf("n=%d ord=%v marshal: %v", n, ordered, err)
			}
			var out []zcRow
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("n=%d ord=%v unmarshal: %v", n, ordered, err)
			}
			if len(out) != n {
				t.Fatalf("n=%d len %d", n, len(out))
			}
			for i := range rows {
				if out[i] != rows[i] {
					t.Fatalf("n=%d [%d] %+v != %+v", n, i, out[i], rows[i])
				}
			}
		}
	}
}

// ---- zone-skip correctness + actual skipping ----

func TestZoneChunkSkipCorrect(t *testing.T) {
	const n = 4096
	rows := zcRows(n, true) // ordered TS → high skip
	b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	loTS, hiTS := rows[1000].TS, rows[1100].TS // a narrow range → most zones skippable

	filter := func(pred func(zcRow) bool) []zcRow {
		var w []zcRow
		for _, r := range rows {
			if pred(r) {
				w = append(w, r)
			}
		}
		return w
	}
	eq := func(name string, got, want []zcRow) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: len %d != %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: [%d] %+v != %+v", name, i, got[i], want[i])
			}
		}
	}
	run := func(name string, q []QueryOption, pred func(zcRow) bool, wantSkip bool) {
		t.Helper()
		zoneSkippedZones = 0
		var out []zcRow
		opts := append([]QueryOption{Select("TS", "U", "V")}, q...)
		if err := Unmarshal(b, &out, opts...); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		eq(name, out, filter(pred))
		if wantSkip && zoneSkippedZones == 0 {
			t.Fatalf("%s: expected zones skipped, got 0", name)
		}
		t.Logf("%s: %d rows, %d zones skipped", name, len(out), zoneSkippedZones)
	}

	run("range", []QueryOption{WhereRange("TS", loTS, hiTS)},
		func(r zcRow) bool { return r.TS >= loTS && r.TS <= hiTS }, true)
	run("ge", []QueryOption{WhereCmp("TS", GE, rows[4000].TS)},
		func(r zcRow) bool { return r.TS >= rows[4000].TS }, true)
	run("le", []QueryOption{WhereCmp("TS", LE, rows[100].TS)},
		func(r zcRow) bool { return r.TS <= rows[100].TS }, true)
	run("gt", []QueryOption{WhereCmp("TS", GT, rows[4000].TS)},
		func(r zcRow) bool { return r.TS > rows[4000].TS }, true)
	run("lt", []QueryOption{WhereCmp("TS", LT, rows[100].TS)},
		func(r zcRow) bool { return r.TS < rows[100].TS }, true)
	run("eq-miss", []QueryOption{WhereCmp("TS", EQ, int64(-999))},
		func(r zcRow) bool { return r.TS == -999 }, true)
	// uint column
	run("uint-range", []QueryOption{WhereRange("U", uint64(700), uint64(800))},
		func(r zcRow) bool { return r.U >= 700 && r.U <= 800 }, true)
	// AND of two bounds on the SAME column: declines zone-skip (two leaves union
	// to the whole domain) and full-decodes — correct, just no skip. Use a single
	// WhereRange to get the skip.
	run("and-range", []QueryOption{WhereCmp("TS", GE, loTS), WhereCmp("TS", LE, hiTS)},
		func(r zcRow) bool { return r.TS >= loTS && r.TS <= hiTS }, false)
	// all-match (no skip expected)
	run("all", []QueryOption{WhereCmp("TS", GE, int64(0))},
		func(r zcRow) bool { return r.TS >= 0 }, false)

	// Back-compat: opaque Where(func) over a zone-chunked column → full eval, no
	// skip, same rows.
	zoneSkippedZones = 0
	var out []zcRow
	if err := Unmarshal(b, &out,
		Where("TS", func(v int64) bool { return v >= loTS && v <= hiTS }),
		Select("TS", "U", "V")); err != nil {
		t.Fatal(err)
	}
	eq("opaque-where", out, filter(func(r zcRow) bool { return r.TS >= loTS && r.TS <= hiTS }))
	if zoneSkippedZones != 0 {
		t.Fatalf("opaque Where should not zone-skip, skipped %d", zoneSkippedZones)
	}
}

// ---- no-flag byte-identical ----

func TestZoneChunkNoFlagIdentical(t *testing.T) {
	rows := zcRows(4096, true)
	withFlag, _ := Marshal(rows, OptBalanced|OptColumnIndex|OptZoneMap)
	noFlag, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	if bytes.Equal(withFlag, noFlag) {
		t.Fatal("OptZoneMap produced identical wire — zone-chunk did not fire")
	}
	// Without the flag the column must NOT carry the zone-chunk tag.
	if bytes.IndexByte(noFlag, tagZoneChunk) >= 0 {
		t.Fatal("no-flag wire contains tagZoneChunk")
	}
}

// ---- malformed / fuzz ----

func TestZoneChunkMalformed(t *testing.T) {
	rows := zcRows(1024, true)
	good, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.IndexByte(good, tagZoneChunk)
	if idx < 0 {
		t.Fatal("no zone-chunk tag")
	}
	decode := func(b []byte) { var out []zcRow; _ = Unmarshal(b, &out) }
	// Deterministic byte-flip fuzz over the zone-chunk region: must error, never
	// panic / OOM.
	for off := idx; off < len(good); off++ {
		for _, bit := range []byte{0x01, 0x80, 0xff} {
			b := append([]byte(nil), good...)
			b[off] ^= bit
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("flip off=%d bit=%#x: panic %v", off, bit, r)
					}
				}()
				decode(b)
			}()
		}
	}
	// Truncations.
	for cut := idx; cut < len(good); cut += 7 {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("truncate %d: panic %v", cut, r)
				}
			}()
			decode(good[:cut])
		}()
	}
}

func FuzzZoneChunkDecode(f *testing.F) {
	s := make([]int64, 600)
	for i := range s {
		s[i] = int64(i * i)
	}
	if b, err := Marshal(s, OptQPack|OptZoneMap); err == nil {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var out []int64
		_ = Unmarshal(data, &out)
	})
}

// ---- property: skip result == full filter across distributions ----

func TestZoneChunkProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	for iter := 0; iter < 100; iter++ {
		n := 512 + rng.Intn(4000)
		ordered := rng.Intn(2) == 0
		rows := zcRows(n, ordered)
		b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
		if err != nil {
			t.Fatalf("iter %d marshal: %v", iter, err)
		}
		// random range over the TS domain
		i1, i2 := rng.Intn(n), rng.Intn(n)
		lo, hi := rows[i1].TS, rows[i2].TS
		if lo > hi {
			lo, hi = hi, lo
		}
		var out []zcRow
		if err := Unmarshal(b, &out, WhereRange("TS", lo, hi), Select("TS", "U", "V")); err != nil {
			t.Fatalf("iter %d unmarshal: %v", iter, err)
		}
		var want []zcRow
		for _, r := range rows {
			if r.TS >= lo && r.TS <= hi {
				want = append(want, r)
			}
		}
		if len(out) != len(want) {
			t.Fatalf("iter %d n=%d ord=%v: len %d != %d", iter, n, ordered, len(out), len(want))
		}
		for i := range want {
			if out[i] != want[i] {
				t.Fatalf("iter %d [%d] mismatch", iter, i)
			}
		}
	}
}
