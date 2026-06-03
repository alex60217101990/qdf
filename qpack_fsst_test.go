package qdf

import "testing"

type fsstRow struct {
	Msg string `qdf:"msg"`
}

// fsstActive reports whether FSST fired for this corpus, reliably and without
// parsing the wire: FSST only emits when it strictly shrinks the column, so a
// large OptFSST-vs-OptBalanced size drop (>=10%, the columnar gain threshold)
// happens only when FSST was actually chosen. containsByte(b, 0xF6) is NOT used
// — 0xF6 occurs in compressed data bytes and gives false positives.
func fsstActive(t *testing.T, rows []fsstRow) bool {
	t.Helper()
	withF, err := Marshal(rows, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	without, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	return len(withF)*10 < len(without)*9
}

func TestFSSTColumnRoundTrip(t *testing.T) {
	rows := mkRows(genURLs(512))
	if !fsstActive(t, rows) {
		t.Fatal("FSST did not activate on the URL corpus")
	}
	b, err := Marshal(rows, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	var back []fsstRow
	if err := Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	assertRowsEqual(t, "roundtrip", rows, back)

	// Same data via the full compression tier (adds rANS over the FSST body)
	// must also round-trip exactly.
	b2, err := Marshal(rows, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var back2 []fsstRow
	if err := Unmarshal(b2, &back2); err != nil {
		t.Fatal(err)
	}
	assertRowsEqual(t, "roundtrip-compression", rows, back2)
}

// TestFSSTNeverLarger: enabling FSST must never grow the wire vs the same opts
// without it, across diverse corpora (incl. incompressible/tiny/low-card).
func TestFSSTNeverLarger(t *testing.T) {
	corpora := map[string][]fsstRow{
		"urls":           mkRows(genURLs(1000)),
		"incompressible": mkRows(genRandom(1000)),
		"tiny":           mkRows([]string{"a", "b", "c"}),
		"low_card":       mkRows(repeatEach([]string{"INFO", "WARN", "ERROR"}, 1000)),
	}
	for name, rows := range corpora {
		base, err := Marshal(rows, OptBalanced)
		if err != nil {
			t.Fatalf("%s balanced: %v", name, err)
		}
		withF, err := Marshal(rows, OptBalanced|OptFSST)
		if err != nil {
			t.Fatalf("%s fsst: %v", name, err)
		}
		if len(withF) > len(base) {
			t.Fatalf("%s: OptBalanced|OptFSST %d > OptBalanced %d (never-larger violated)", name, len(withF), len(base))
		}
		// And with the column index too (the selective-decode path).
		baseIdx, _ := Marshal(rows, OptBalanced|OptColumnIndex)
		fsstIdx, _ := Marshal(rows, OptBalanced|OptFSST|OptColumnIndex)
		if len(fsstIdx) > len(baseIdx) {
			t.Fatalf("%s(idx): %d > %d (never-larger violated)", name, len(fsstIdx), len(baseIdx))
		}
		var back []fsstRow
		if err := Unmarshal(withF, &back); err != nil {
			t.Fatalf("%s decode: %v", name, err)
		}
		assertRowsEqual(t, name, rows, back)
	}
	if !fsstActive(t, mkRows(genURLs(1000))) {
		t.Fatal("expected FSST to be chosen on the URL corpus")
	}
}

// TestFSSTSelectiveDecode exercises predicate pushdown + projection over a FSST
// string column (the decodeColumnInto path). genURLs activates FSST and, being
// columnar, supports selective decode.
func TestFSSTSelectiveDecode(t *testing.T) {
	rows := mkRows(genURLs(1000))
	if !fsstActive(t, rows) {
		t.Fatal("FSST not active; predicate test would not exercise the FSST path")
	}
	b, err := Marshal(rows, OptBalanced|OptFSST|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var hot []fsstRow
	err = Unmarshal(b, &hot, Where("msg", func(s string) bool {
		return len(s) > 0 && s[0] == 'G' // GET lines
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, r := range rows {
		if r.Msg[0] == 'G' {
			want++
		}
	}
	if len(hot) != want {
		t.Fatalf("matched %d rows, want %d", len(hot), want)
	}
	for _, r := range hot {
		if r.Msg == "" || r.Msg[0] != 'G' {
			t.Fatalf("predicate leaked a non-match: %q", r.Msg)
		}
	}
}

// TestFSSTWithNoCopySurvivesBufferReuse: FSST strings decompress into an owned
// slab, so they survive the input buffer being overwritten under WithNoCopy.
func TestFSSTWithNoCopySurvivesBufferReuse(t *testing.T) {
	rows := mkRows(genURLs(256))
	if !fsstActive(t, rows) {
		t.Skip("FSST not active on this corpus")
	}
	b, err := Marshal(rows, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	buf := append([]byte(nil), b...)
	var back []fsstRow
	if err := Unmarshal(buf, &back, WithNoCopy()); err != nil {
		t.Fatal(err)
	}
	keep := append([]string(nil), rowsMsgs(back)...)
	for i := range buf {
		buf[i] = 0xAA // scribble the input
	}
	for i := range back {
		if back[i].Msg != keep[i] {
			t.Fatalf("FSST string aliased the input buffer (row %d): %q != %q", i, back[i].Msg, keep[i])
		}
	}
}

// TestFSSTMalformedDecode: every truncated prefix of a valid FSST wire must
// error, never panic or OOM.
func TestFSSTMalformedDecode(t *testing.T) {
	rows := mkRows(genURLs(256))
	if !fsstActive(t, rows) {
		t.Skip("FSST not active on this corpus")
	}
	good, err := Marshal(rows, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	for cut := 1; cut < len(good); cut++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncated input at %d/%d: %v", cut, len(good), r)
				}
			}()
			var back []fsstRow
			_ = Unmarshal(good[:cut], &back) // error expected; panic forbidden
		}()
	}
}

// ----- deterministic test corpora (no RNG; index-derived) -----

func genURLs(n int) []string {
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	paths := []string{"/api/v1/users/", "/api/v1/orders/", "/static/img/", "/health/"}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = methods[i%len(methods)] + " https://service.example.com" +
			paths[i%len(paths)] + itoaTiny(i*7%100000) +
			"?token=" + itoaTiny(i*2654435761&0xFFFFFF) + " HTTP/1.1"
	}
	return out
}

func genRandom(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		var b [24]byte
		x := uint32(i)*2654435761 + 1
		for j := range b {
			x = x*1664525 + 1013904223
			b[j] = byte(33 + x%94)
		}
		out[i] = string(b[:])
	}
	return out
}

func repeatEach(vals []string, total int) []string {
	out := make([]string, total)
	for i := range out {
		out[i] = vals[i%len(vals)]
	}
	return out
}

func mkRows(msgs []string) []fsstRow {
	rows := make([]fsstRow, len(msgs))
	for i, m := range msgs {
		rows[i].Msg = m
	}
	return rows
}

func rowsMsgs(rows []fsstRow) []string {
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].Msg
	}
	return out
}

func assertRowsEqual(t *testing.T, name string, want, got []fsstRow) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len %d want %d", name, len(got), len(want))
	}
	for i := range want {
		if got[i].Msg != want[i].Msg {
			t.Fatalf("%s row %d: %q != %q", name, i, got[i].Msg, want[i].Msg)
		}
	}
}

func itoaTiny(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}
