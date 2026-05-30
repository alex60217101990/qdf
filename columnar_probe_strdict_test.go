package qdf

import (
	"testing"
)

// RED until columnarProbe prices the string dictionary: a struct whose
// fields are ALL enum-like strings (no numeric column to tip the probe)
// must still commit to columnar so the per-column string dictionary fires.
// Today the probe estimates string columns at per-value/repeat cost only,
// so it declines columnar and the struct stays row-major (≈10× larger).
func TestColumnarProbe_PricesStringDict(t *testing.T) {
	type dimRow struct {
		Level   string
		Service string
		Region  string
	}
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	services := []string{"api", "auth", "billing", "user"}
	regions := []string{"us-east-1", "eu-west-1", "ap-south-1"}
	n := 1000
	rows := make([]dimRow, n)
	// Strictly cycling — low cardinality but NO consecutive repeats, so the
	// probe's repeat heuristic sees no columnar gain and declines today, even
	// though the string dictionary would crush these columns.
	for i := range rows {
		rows[i] = dimRow{
			Level:   levels[i%len(levels)],
			Service: services[(i+1)%len(services)],
			Region:  regions[(i+2)%len(regions)],
		}
	}

	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var got []dimRow
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d mismatch", i)
		}
	}

	// With columnar + the string dictionary, three low-cardinality columns
	// over n rows cost roughly the distinct tables plus ~2 bits/row each —
	// well under one byte per row total. Row-major (the current behaviour)
	// spends several bytes per row, so this threshold separates the two.
	if len(enc) >= n {
		t.Fatalf("pure-string-enum struct encoded to %d bytes for %d rows (>= 1 byte/row) — columnar string-dict did not engage", len(enc), n)
	}
	t.Logf("n=%d wire=%d (%.2f bytes/row)", n, len(enc), float64(len(enc))/float64(n))
}
