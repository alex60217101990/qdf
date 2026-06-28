package qdf

import (
	"bytes"
	"errors"
	"math"
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

// ---- float64 zone-chunk ----

type zcFRow struct {
	F   float64 // ordered predicate column
	K   int64   // payload
	J   int64   // low-card payload
	Tag string  // constant string — strong columnar-win signal so the probe transposes
}

func TestZoneChunkFloat64(t *testing.T) {
	const n = 4096
	rng := rand.New(rand.NewSource(9))
	rows := make([]zcFRow, n)
	f := 0.0
	for i := range rows {
		f += rng.Float64() // monotonically increasing → ordered
		v := f
		if i%500 == 0 {
			v = math.NaN() // sprinkle NaN — must round-trip + never match
		}
		rows[i] = zcFRow{F: v, K: int64(i), J: int64(i % 8), Tag: "const"}
	}
	b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}

	// Round-trip incl. NaN (bit-exact).
	var all []zcFRow
	if err := Unmarshal(b, &all); err != nil {
		t.Fatal(err)
	}
	if len(all) != n {
		t.Fatalf("len %d", len(all))
	}
	for i := range rows {
		if math.IsNaN(rows[i].F) {
			if !math.IsNaN(all[i].F) || all[i].K != rows[i].K {
				t.Fatalf("[%d] NaN row not preserved: %+v", i, all[i])
			}
			continue
		}
		if all[i] != rows[i] {
			t.Fatalf("[%d] %+v != %+v", i, all[i], rows[i])
		}
	}

	// Zone-skip range: result == full filter (NaN never matches), zones skipped.
	lo, hi := rows[1001].F, rows[1099].F
	zoneSkippedZones = 0
	var out []zcFRow
	if err := Unmarshal(b, &out, WhereRange("F", lo, hi), Select("F", "K", "J", "Tag")); err != nil {
		t.Fatal(err)
	}
	var want []zcFRow
	for _, r := range rows {
		if r.F >= lo && r.F <= hi { // NaN >= lo is false → excluded, as required
			want = append(want, r)
		}
	}
	if len(out) != len(want) {
		t.Fatalf("range len %d != %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("range [%d] %+v != %+v", i, out[i], want[i])
		}
	}
	if zoneSkippedZones == 0 {
		t.Fatal("float zone-skip skipped 0 zones")
	}
	t.Logf("float64: %d rows, %d zones skipped", len(out), zoneSkippedZones)

	// WhereCmp GE on float.
	zoneSkippedZones = 0
	var ge []zcFRow
	if err := Unmarshal(b, &ge, WhereCmp("F", GE, rows[3997].F), Select("F", "K", "J", "Tag")); err != nil {
		t.Fatal(err)
	}
	cnt := 0
	for _, r := range rows {
		if r.F >= rows[3997].F {
			cnt++
		}
	}
	if len(ge) != cnt {
		t.Fatalf("ge len %d != %d", len(ge), cnt)
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
	for iter := range 100 {
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

// TestZoneChunkProjMatchedViaSibling guards the zone-skip projection path: a
// projected zone-chunked column whose rows are matched via an OR/NOT sibling on
// another column must still project the REAL value. The filter pass skips zones
// the column's own bounds exclude, so projection must follow the final matched
// set, not the leaf bounds. Regression for the "skipped zone projects zero" bug.
func TestZoneChunkProjMatchedViaSibling(t *testing.T) {
	const n = 4096
	rows := zcRows(n, true) // ordered TS ascending; V=int64(i); U=uint64(i)*7
	b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	check := func(name string, q []QueryOption, pred func(zcRow) bool) {
		t.Helper()
		var want []zcRow
		for _, r := range rows {
			if pred(r) {
				want = append(want, r)
			}
		}
		opts := append([]QueryOption{Select("TS", "U", "V")}, q...)
		var out []zcRow
		if err := Unmarshal(b, &out, opts...); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(out) != len(want) {
			t.Fatalf("%s: len %d != %d", name, len(out), len(want))
		}
		for i := range want {
			if out[i] != want[i] {
				t.Fatalf("%s: [%d] %+v != %+v", name, i, out[i], want[i])
			}
		}
	}

	// OR: TS leaf (GE high) skips the low zones; V leaf (LE 5) matches rows 0..5
	// which live in those skipped zones. TS must still project the real value.
	check("or", []QueryOption{Or(WhereCmp("TS", GE, rows[4000].TS), WhereCmp("V", LE, int64(5)))},
		func(r zcRow) bool { return r.TS >= rows[4000].TS || r.V <= 5 })

	// NOT: leaf bounds intersect only the high zones, but NOT flips the match to
	// the low zones — the projected TS column must follow the inverted match.
	check("not", []QueryOption{Not(WhereCmp("TS", GE, rows[4000].TS))},
		func(r zcRow) bool { return r.TS < rows[4000].TS })

	// OR on the uint zone-chunked column matched via a sibling on TS.
	check("or-uint", []QueryOption{Or(WhereCmp("U", GE, uint64(4000)*7), WhereCmp("TS", LE, rows[5].TS))},
		func(r zcRow) bool { return r.U >= 4000*7 || r.TS <= rows[5].TS })
}

// TestZoneChunkDepthGuard verifies the zone-chunk decoders bound recursion depth
// (a zone body can itself be a tagZoneChunk, so nested payloads must not overflow
// the stack). At the depth cap the decoder returns ErrCycleDetected without
// touching the buffer.
func TestZoneChunkDepthGuard(t *testing.T) {
	for _, fn := range []func(d *Decoder) error{
		func(d *Decoder) error { return d.readZoneChunkInt64Into(new([]int64)) },
		func(d *Decoder) error { return d.readZoneChunkUint64Into(new([]uint64)) },
		func(d *Decoder) error { return d.readZoneChunkFloat64Into(new([]float64)) },
	} {
		d := &Decoder{maxDepth: DefaultMaxDepth, depth: DefaultMaxDepth}
		if err := fn(d); !errors.Is(err, ErrCycleDetected) {
			t.Fatalf("want ErrCycleDetected at depth cap, got %v", err)
		}
	}
}

// firstZoneZmap returns the zmap byte of the first int/uint tagZoneChunk column
// in buf, or 0xFF if none. Layout: tag, kind, zmap, blkLog, ...
func firstZoneZmap(buf []byte) byte {
	for i := 0; i+2 < len(buf); i++ {
		if buf[i] == tagZoneChunk && (buf[i+1] == zoneKindInt || buf[i+1] == zoneKindUint) {
			return buf[i+2]
		}
	}
	return 0xFF
}

type linRow struct {
	ID  int64  // perfectly linear → linear zmap
	U   uint64 // perfectly linear
	V   int64  // payload
	Tag string // constant → strong columnar-win signal so the probe transposes
}

// TestZoneChunkLinearChosen: a perfectly linear sorted column picks the linear
// zonemap (smaller wire) and still zone-skips with exact results.
func TestZoneChunkLinearChosen(t *testing.T) {
	const n = 8192
	rows := make([]linRow, n)
	for i := range rows {
		rows[i] = linRow{ID: int64(1_000_000 + i*5), U: uint64(i) * 3, V: int64(i), Tag: "c"}
	}
	lin, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	if z := firstZoneZmap(lin); z != zmapLinear {
		t.Fatalf("expected linear zmap on a linear column, got 0x%02x", z)
	}

	// Round-trip exact.
	var out []linRow
	if err := Unmarshal(lin, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != n {
		t.Fatalf("len %d", len(out))
	}
	for i := range rows {
		if out[i] != rows[i] {
			t.Fatalf("[%d] %+v != %+v", i, out[i], rows[i])
		}
	}

	// Range query zone-skips and returns exact rows.
	zoneSkippedZones = 0
	lo, hi := rows[2000].ID, rows[2050].ID
	var q []linRow
	if err := Unmarshal(lin, &q, WhereRange("ID", lo, hi), Select("ID", "U", "V", "Tag")); err != nil {
		t.Fatal(err)
	}
	var want []linRow
	for _, r := range rows {
		if r.ID >= lo && r.ID <= hi {
			want = append(want, r)
		}
	}
	if len(q) != len(want) {
		t.Fatalf("range len %d != %d", len(q), len(want))
	}
	for i := range want {
		if q[i] != want[i] {
			t.Fatalf("range [%d] %+v != %+v", i, q[i], want[i])
		}
	}
	if zoneSkippedZones == 0 {
		t.Fatal("linear zone-skip skipped no zones")
	}
}

// TestZoneChunkLinearFallback: a non-monotonic column falls back to the min/max
// zonemap (linear undefined for unsorted data) and stays correct.
func TestZoneChunkLinearFallback(t *testing.T) {
	const n = 4096
	rng := rand.New(rand.NewSource(11))
	rows := make([]linRow, n)
	for i := range rows {
		rows[i] = linRow{ID: rng.Int63n(1 << 30), U: uint64(rng.Int63n(1 << 30)), V: int64(i), Tag: "c"}
	}
	b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	if z := firstZoneZmap(b); z != zmapMinMax {
		t.Fatalf("expected minmax zmap on a random column, got 0x%02x", z)
	}
	var out []linRow
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if out[i] != rows[i] {
			t.Fatalf("[%d] %+v != %+v", i, out[i], rows[i])
		}
	}
}

// TestZoneChunkLinearNoFalseSkip is the key correctness guard: over many random
// monotonic columns and random range/comparison queries, the linear zone-skip
// result must equal a brute-force filter (no matching row is ever skipped).
func TestZoneChunkLinearNoFalseSkip(t *testing.T) {
	rng := rand.New(rand.NewSource(2026))
	for iter := range 200 {
		n := 512 + rng.Intn(8000)
		rows := make([]linRow, n)
		v := rng.Int63n(1000)
		for i := range rows {
			// monotonic non-decreasing with varied step (still often linear-fit)
			step := int64(rng.Intn(7))
			if rng.Intn(20) == 0 {
				step += int64(rng.Intn(500)) // occasional jump
			}
			v += step
			rows[i] = linRow{ID: v, U: uint64(v), V: int64(i), Tag: "c"}
		}
		b, err := Marshal(rows, OptBalanced|OptZoneMap|OptColumnIndex)
		if err != nil {
			t.Fatalf("iter %d: %v", iter, err)
		}
		lo := rows[rng.Intn(n)].ID
		hi := lo + int64(rng.Intn(2000))
		var got []linRow
		if err := Unmarshal(b, &got, WhereRange("ID", lo, hi), Select("ID", "U", "V", "Tag")); err != nil {
			t.Fatalf("iter %d: %v", iter, err)
		}
		var want []linRow
		for _, r := range rows {
			if r.ID >= lo && r.ID <= hi {
				want = append(want, r)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("iter %d zmap=0x%02x n=%d [%d,%d]: len %d != %d (FALSE SKIP?)",
				iter, firstZoneZmap(b), n, lo, hi, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("iter %d [%d]: %+v != %+v", iter, i, got[i], want[i])
			}
		}
	}
}
