package qdf

import (
	"math/rand"
	"testing"
)

// TestNullable_RANS_RoundTrip verifies that nullable columnar payloads
// round-trip correctly under OptCompression (which includes OptRANS).
// The existing TestNullable_RoundTrip explicitly excludes OptRANS via
// OptBalanced &^ OptRANS; this test closes that gap.
func TestNullable_RANS_RoundTrip(t *testing.T) {
	for _, nullPct := range []int{0, 30, 95, 100} {
		for _, n := range []int{20, 1000} {
			rows := mkNullRows(n, nullPct, int64(nullPct*7+n+1))
			enc, err := Marshal(rows, OptCompression) // RANS included
			if err != nil {
				t.Fatalf("null%%=%d n=%d marshal: %v", nullPct, n, err)
			}
			var got []nullRow
			if err := Unmarshal(enc, &got); err != nil {
				t.Fatalf("null%%=%d n=%d unmarshal: %v", nullPct, n, err)
			}
			if len(got) != len(rows) {
				t.Fatalf("null%%=%d n=%d: len %d != %d", nullPct, n, len(got), len(rows))
			}
			for i := range rows {
				if !eqNullRow(got[i], rows[i]) {
					t.Fatalf("null%%=%d n=%d row %d mismatch:\n want=%+v\n  got=%+v",
						nullPct, n, i, rows[i], got[i])
				}
			}
		}
	}
}

// TestNullable_RANS_ColIndex_RoundTrip exercises nullable columns under
// OptCompression|OptColumnIndex (B7), the heaviest bundle, which was
// never tested with nullable fields before.
func TestNullable_RANS_ColIndex_RoundTrip(t *testing.T) {
	for _, nullPct := range []int{0, 30, 95, 100} {
		for _, n := range []int{20, 1000} {
			rows := mkNullRows(n, nullPct, int64(nullPct*13+n+2))
			opts := OptCompression | OptColumnIndex
			enc, err := Marshal(rows, opts)
			if err != nil {
				t.Fatalf("null%%=%d n=%d marshal: %v", nullPct, n, err)
			}
			var got []nullRow
			if err := Unmarshal(enc, &got); err != nil {
				t.Fatalf("null%%=%d n=%d unmarshal: %v", nullPct, n, err)
			}
			if len(got) != len(rows) {
				t.Fatalf("null%%=%d n=%d: len %d != %d", nullPct, n, len(got), len(rows))
			}
			for i := range rows {
				if !eqNullRow(got[i], rows[i]) {
					t.Fatalf("null%%=%d n=%d row %d mismatch:\n want=%+v\n  got=%+v",
						nullPct, n, i, rows[i], got[i])
				}
			}
		}
	}
}

// colRANSRow is a mixed-column struct used by TestColRANS_RoundTrip.
// It intentionally combines numeric, string, nullable, and float columns
// so that the columnar encoder engages all codec paths under RANS.
type colRANSRow struct {
	ID   int64
	Code uint32
	Name string
	Opt  *int64
	F    float64
}

// mkColRANSRows builds ~200 rows with enum-ish repeated values (to
// trigger QPack / RLE / dict codecs) and roughly 1/3 nil Opt.
func mkColRANSRows(n int, seed int64) []colRANSRow {
	r := rand.New(rand.NewSource(seed))
	names := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	rows := make([]colRANSRow, n)
	for i := range rows {
		rows[i].ID = int64(i)
		rows[i].Code = uint32(r.Intn(8)) // low-cardinality → QPack fires
		rows[i].Name = names[r.Intn(len(names))]
		rows[i].F = float64(r.Intn(100)) / 10.0 // quantized → ALP/Gorilla fires
		if r.Intn(3) != 0 {                     // ~1/3 nil
			v := int64(r.Intn(1000))
			rows[i].Opt = &v
		}
	}
	return rows
}

// TestColRANS_RoundTrip builds a mixed nullable+numeric+string []struct
// and calls roundtripColumnar, which exercises all bundles including
// B6_Balanced_ColIndex and B7_Compression_ColIndex (ColumnIndex + RANS).
func TestColRANS_RoundTrip(t *testing.T) {
	rows := mkColRANSRows(200, 42)
	roundtripColumnar(t, rows)
}
