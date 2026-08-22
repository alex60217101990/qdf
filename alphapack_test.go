package qdf

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

type apRow struct {
	ID int64  `qdf:"id"`
	S  string `qdf:"s"`
}

func apMarshalRows(t *testing.T, vals []string, opt Options) []byte {
	t.Helper()
	rows := make([]apRow, len(vals))
	for i, v := range vals {
		rows[i] = apRow{int64(i), v}
	}
	b, err := Marshal(rows, opt)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return b
}

func apRoundTrip(t *testing.T, vals []string, b []byte) {
	t.Helper()
	want := make([]apRow, len(vals))
	for i, v := range vals {
		want[i] = apRow{int64(i), v}
	}
	var got []apRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

// hexCol returns n distinct lowercase-hex strings of length ln (high-card,
// restricted 16-symbol alphabet — the class dict/FC/FSST miss but positional
// bit-packing halves).
func hexCol(n, ln int, seed int64) []string {
	r := rand.New(rand.NewSource(seed))
	const hexChars = "0123456789abcdef"
	out := make([]string, n)
	for i := range out {
		b := make([]byte, ln)
		for j := range b {
			b[j] = hexChars[r.Intn(16)]
		}
		out[i] = string(b)
	}
	return out
}

type apRowS struct {
	S string `qdf:"s"`
}

// apMarshalSingle marshals a single-string-column []struct. Used where a clean
// tag scan is needed (printable-ASCII bodies never contain 0xFB).
func apMarshalSingle(t *testing.T, vals []string, opt Options) []byte {
	t.Helper()
	rows := make([]apRowS, len(vals))
	for i, v := range vals {
		rows[i] = apRowS{v}
	}
	b, err := Marshal(rows, opt)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return b
}

// apLogRow mirrors the trace/log shape alpha-packing targets: low-card enum
// columns (level/service) that carry the struct into the columnar form, plus a
// high-cardinality restricted-alphabet hex ID (span) the alpha codec packs.
type apLogRow struct {
	TS      int64  `qdf:"ts"`
	Level   int    `qdf:"level"`
	Service string `qdf:"service"`
	Span    string `qdf:"span"`
}

// TestAlphaPackFiresHex: a high-cardinality 16-char hex column inside a
// trace/log-shaped struct (whose enum columns carry it into the columnar form)
// encodes via the alphabet-packed codec, round-trips exactly, and the hex
// column's contribution is strictly smaller than its raw per-value floor (4
// bits/char vs 8). On hex high-card data only alpha-packing beats that floor
// (dict declines on cardinality, FSST is off under Balanced), so the
// string-bytes delta is an unambiguous firing signal.
func TestAlphaPackFiresHex(t *testing.T) {
	const n = 4000
	spans := hexCol(n, 16, 1)
	services := []string{"api", "auth", "billing", "edge"}
	mk := func(spanOf func(i int) string) []apLogRow {
		rows := make([]apLogRow, n)
		for i := range rows {
			rows[i] = apLogRow{
				TS:      int64(1_700_000_000 + i),
				Level:   i % 5,
				Service: services[i%len(services)],
				Span:    spanOf(i),
			}
		}
		return rows
	}
	full := mk(func(i int) string { return spans[i] })
	blank := mk(func(int) string { return "" })

	b, err := Marshal(full, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	bb, err := Marshal(blank, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	strBytes := len(b) - len(bb) // encoded cost attributable to the span column

	rawFloor := 0
	for _, s := range spans {
		rawFloor += uvarintLen(uint64(len(s))) + len(s)
	}
	if strBytes >= rawFloor {
		t.Fatalf("alpha did not fire: span column (%d) not smaller than raw floor (%d)", strBytes, rawFloor)
	}

	var got []apLogRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, full) {
		t.Fatal("round-trip mismatch")
	}
	t.Logf("spanColumn=%d rawFloor=%d (-%.1f%%)", strBytes, rawFloor, 100*float64(rawFloor-strBytes)/float64(rawFloor))
}

// TestAlphaPackDeclinesFullAlphabet: a high-card column over the full byte
// alphabet (>64 distinct bytes) must NOT use alpha-packing (it cannot pack
// below 8 bits) — no regression, the picker keeps the existing form.
func TestAlphaPackDeclinesFullAlphabet(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	vals := make([]string, 3000)
	for i := range vals {
		b := make([]byte, 24)
		for j := range b {
			b[j] = byte(32 + r.Intn(95)) // 95-symbol printable ASCII alphabet
		}
		vals[i] = string(b)
	}
	// Single-column wire: printable-ASCII bodies (0x20–0x7e) and small varints
	// never contain 0xFB, so a tag scan is reliable here.
	b := apMarshalSingle(t, vals, OptBalanced)
	if bytes.ContainsRune(b, rune(tagColStrAlpha)) {
		t.Fatal("alpha-packing fired on a full-alphabet column — cannot pack below 8 bits")
	}
	var got []apRowS
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
}

// TestAlphaPackRoundTripVariety exercises several restricted alphabets and a
// variable-length column to lock the decoder against the encoder.
func TestAlphaPackRoundTripVariety(t *testing.T) {
	cases := []struct {
		name string
		vals []string
	}{
		{"hex32", hexCol(2000, 32, 10)},
		{"hex16-fixed", hexCol(2000, 16, 11)},
		{"uuid", func() []string {
			h := hexCol(2000, 32, 12)
			out := make([]string, len(h))
			for i, s := range h {
				out[i] = s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
			}
			return out
		}()},
		{"varlen-hex", func() []string {
			r := rand.New(rand.NewSource(13))
			const hexChars = "0123456789abcdef"
			out := make([]string, 2000)
			for i := range out {
				ln := 8 + r.Intn(24)
				b := make([]byte, ln)
				for j := range b {
					b[j] = hexChars[r.Intn(16)]
				}
				out[i] = string(b)
			}
			return out
		}()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := apMarshalRows(t, c.vals, OptBalanced)
			apRoundTrip(t, c.vals, b)
			_ = fmt.Sprint // keep fmt imported
		})
	}
}

// TestAlphaPackRoundTripOracle is a randomized round-trip oracle: many column
// shapes over restricted and unrestricted alphabets, fixed and variable
// lengths, low and high cardinality, plus edges (empty strings, single distinct,
// alphabet straddling 64). Every shape must round-trip exactly whether or not
// alpha-packing fires.
func TestAlphaPackRoundTripOracle(t *testing.T) {
	rng := func(seed uint64) func(int) int {
		s := seed
		return func(n int) int {
			s = s*6364136223846793005 + 1
			if n <= 0 {
				return 0
			}
			return int((s >> 33) % uint64(n))
		}
	}
	alphabets := []string{
		"0123456789abcdef",                 // hex (16)
		"0123456789",                       // decimal (10)
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", // base32 (32)
		"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/",       // base64 (64)
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-._~:/?#", // >64
		"ab", // tiny alphabet (2)
	}
	for seed := uint64(1); seed <= 3000; seed++ {
		r := rng(seed)
		alpha := alphabets[r(len(alphabets))]
		fixed := r(2) == 0
		baseLen := 1 + r(34)
		card := 1 + r(300)
		pool := make([]string, card)
		for i := range pool {
			ln := baseLen
			if !fixed {
				ln = r(baseLen + 1)
			}
			b := make([]byte, ln)
			for j := range b {
				b[j] = alpha[r(len(alpha))]
			}
			pool[i] = string(b)
			if r(40) == 0 {
				pool[i] = "" // empty edge
			}
		}
		nrows := r(400)
		rows := make([]apLogRow, nrows)
		services := []string{"api", "auth", "billing", "edge"}
		for i := range rows {
			rows[i] = apLogRow{
				TS:      int64(i),
				Level:   r(5),
				Service: services[r(len(services))],
				Span:    pool[r(len(pool))],
			}
		}
		b, err := Marshal(rows, OptBalanced)
		if err != nil {
			t.Fatalf("seed %d: Marshal: %v", seed, err)
		}
		var got []apLogRow
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("seed %d: Unmarshal: %v", seed, err)
		}
		if len(rows) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, rows) {
			t.Fatalf("seed %d: round-trip mismatch (alpha=%q fixed=%v card=%d nrows=%d)", seed, alpha, fixed, card, nrows)
		}
	}
}

// FuzzAlphaPackHostile feeds mutated bytes into a columnar decode seeded with an
// alphabet-packed string column: a hostile tagColStrAlpha frame must never panic
// or OOM, only error.
func FuzzAlphaPackHostile(f *testing.F) {
	spans := hexCol(200, 16, 7)
	services := []string{"api", "auth", "billing", "edge"}
	rows := make([]apLogRow, 200)
	for i := range rows {
		rows[i] = apLogRow{TS: int64(i), Level: i % 5, Service: services[i%4], Span: spans[i]}
	}
	seed, _ := Marshal(rows, OptBalanced)
	f.Add(seed)
	// also a varlen seed
	for i := range rows {
		rows[i].Span = spans[i][:8+i%8]
	}
	seed2, _ := Marshal(rows, OptBalanced)
	f.Add(seed2)
	f.Fuzz(func(_ *testing.T, b []byte) {
		var out []apLogRow
		_ = Unmarshal(b, &out) // must not panic / OOM
	})
}
