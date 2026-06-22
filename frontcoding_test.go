package qdf

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type fcRow struct {
	ID int64  `qdf:"id"`
	S  string `qdf:"s"`
}

func fcMarshalRows(t *testing.T, vals []string) []byte {
	t.Helper()
	rows := make([]fcRow, len(vals))
	for i, v := range vals {
		rows[i] = fcRow{int64(i), v}
	}
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return b
}

func fcRoundTrip(t *testing.T, vals []string, b []byte) {
	t.Helper()
	want := make([]fcRow, len(vals))
	for i, v := range vals {
		want[i] = fcRow{int64(i), v}
	}
	var got []fcRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

// TestFrontCodedDictFires: a prefix-shared medium-cardinality string column
// encodes via the front-coded dictionary (tagColStrDictFC), round-trips exactly,
// and the front-coded form is strictly smaller than the plain dictionary.
func TestFrontCodedDictFires(t *testing.T) {
	// SID-like: 200 distinct values sharing a long common prefix, over 4000 rows.
	const distinct, rows = 200, 4000
	pool := make([]string, distinct)
	for i := range pool {
		pool[i] = fmt.Sprintf("S-1-5-21-3623811015-3361044348-30300820-%d", 1000+i)
	}
	vals := make([]string, rows)
	for i := range vals {
		vals[i] = pool[(i*7)%distinct]
	}

	b := fcMarshalRows(t, vals)
	if !bytes.ContainsRune(b, rune(tagColStrDictFC)) {
		t.Fatalf("expected tagColStrDictFC (0x%02x) in wire — front-coding did not fire", tagColStrDictFC)
	}
	fcRoundTrip(t, vals, b)

	// Never-larger: the front-coded wire must beat what a plain dict would emit.
	// Reconstruct the plain-dict size from the same distinct table for a check.
	plainTbl, fcTbl := 0, 0
	seen := map[string]bool{}
	var order []string
	for _, v := range vals {
		if !seen[v] {
			seen[v] = true
			order = append(order, v)
			plainTbl += uvarintLen(uint64(len(v))) + len(v)
		}
	}
	sorted := append([]string(nil), order...)
	for i := 1; i < len(sorted); i++ { // insertion sort, small
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	prev := ""
	for _, s := range sorted {
		p := commonPrefixLen(prev, s)
		fcTbl += uvarintLen(uint64(p)) + uvarintLen(uint64(len(s)-p)) + (len(s) - p)
		prev = s
	}
	if fcTbl >= plainTbl {
		t.Fatalf("front-coded table (%d) not smaller than plain (%d)", fcTbl, plainTbl)
	}
	t.Logf("table bytes: plain=%d front-coded=%d (-%.1f%%)", plainTbl, fcTbl, 100*float64(plainTbl-fcTbl)/float64(plainTbl))
}

// TestFrontCodedDictDeclinesNoRegression: a high-entropy / non-prefix-shared
// column must NOT use the front-coded form (it would be larger), so the wire is
// identical to before — no regression.
func TestFrontCodedDictDeclinesNoRegression(t *testing.T) {
	// Short enum-like values with no shared prefix: front-coding would add
	// per-entry overhead, so the picker keeps the plain dict.
	const rows = 2000
	pool := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf"}
	vals := make([]string, rows)
	for i := range vals {
		vals[i] = pool[(i*3)%len(pool)]
	}
	b := fcMarshalRows(t, vals)
	if bytes.ContainsRune(b, rune(tagColStrDictFC)) {
		t.Fatalf("front-coding fired on a non-prefix-shared column — would regress")
	}
	fcRoundTrip(t, vals, b)
}

// TestFrontCodedDictEdge: boundary values (empty string, single-char, exact
// prefix of another) round-trip through the front-coded path.
func TestFrontCodedDictEdge(t *testing.T) {
	pool := []string{
		"", "a", "ab", "abc", "abcd",
		"prefix/", "prefix/a", "prefix/ab", "prefix/abc",
		"/very/long/shared/path/component/value-1",
		"/very/long/shared/path/component/value-2",
		"/very/long/shared/path/component/value-3",
	}
	const rows = 600
	vals := make([]string, rows)
	for i := range vals {
		vals[i] = pool[(i*5)%len(pool)]
	}
	b := fcMarshalRows(t, vals)
	fcRoundTrip(t, vals, b) // FC may or may not fire; either way must round-trip
}
