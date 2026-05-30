package qdf

import (
	"math/rand"
	"testing"
)

type nullRow struct {
	Seq int64    // sequential → columnar engages
	A   *int64   // optional
	B   *uint32  // optional, narrow
	C   *float64 // optional
	D   *bool    // optional
}

func mkNullRows(n int, nullPct int, seed int64) []nullRow {
	r := rand.New(rand.NewSource(seed))
	out := make([]nullRow, n)
	for i := range out {
		out[i].Seq = int64(i)
		if r.Intn(100) >= nullPct {
			v := int64(r.Intn(10000) - 5000)
			out[i].A = &v
		}
		if r.Intn(100) >= nullPct {
			v := uint32(r.Intn(1 << 20))
			out[i].B = &v
		}
		if r.Intn(100) >= nullPct {
			v := float64(r.Intn(1000)) / 7
			out[i].C = &v
		}
		if r.Intn(100) >= nullPct {
			v := r.Intn(2) == 0
			out[i].D = &v
		}
	}
	return out
}

func eqNullRow(a, b nullRow) bool {
	if a.Seq != b.Seq {
		return false
	}
	eqI := func(x, y *int64) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	eqU := func(x, y *uint32) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	eqF := func(x, y *float64) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	eqB := func(x, y *bool) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return eqI(a.A, b.A) && eqU(a.B, b.B) && eqF(a.C, b.C) && eqB(a.D, b.D)
}

func TestNullable_RoundTrip(t *testing.T) {
	for _, nullPct := range []int{0, 30, 95, 100} {
		for _, n := range []int{20, 1000} {
			rows := mkNullRows(n, nullPct, int64(nullPct*7+n))
			enc, err := Marshal(rows, OptBalanced&^OptRANS)
			if err != nil {
				t.Fatalf("null%%=%d n=%d: %v", nullPct, n, err)
			}
			var got []nullRow
			if err := Unmarshal(enc, &got); err != nil {
				t.Fatalf("null%%=%d n=%d unmarshal: %v", nullPct, n, err)
			}
			if len(got) != len(rows) {
				t.Fatalf("len %d != %d", len(got), len(rows))
			}
			for i := range rows {
				if !eqNullRow(got[i], rows[i]) {
					t.Fatalf("null%%=%d n=%d row %d mismatch: %+v != %+v", nullPct, n, i, got[i], rows[i])
				}
			}
		}
	}
}

func TestNullable_RoundTripAny(t *testing.T) {
	rows := mkNullRows(500, 40, 99)
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var gotAny any
	if err := Unmarshal(enc, &gotAny); err != nil {
		t.Fatal(err)
	}
	got, ok := gotAny.([]any)
	if !ok || len(got) != len(rows) {
		t.Fatalf("any shape: %T", gotAny)
	}
	for i := range rows {
		m := got[i].(map[string]any)
		if rows[i].A == nil {
			if m["A"] != nil {
				t.Fatalf("row %d A: want nil got %v", i, m["A"])
			}
		} else if m["A"].(int64) != *rows[i].A {
			t.Fatalf("row %d A: %v != %d", i, m["A"], *rows[i].A)
		}
		if rows[i].D == nil {
			if m["D"] != nil {
				t.Fatalf("row %d D: want nil got %v", i, m["D"])
			}
		} else if m["D"].(bool) != *rows[i].D {
			t.Fatalf("row %d D mismatch", i)
		}
	}
}

func TestNullable_SmallerThanRowMajor(t *testing.T) {
	rows := mkNullRows(2000, 30, 7)
	col, err := Marshal(rows, OptBalanced&^OptRANS) // columnar + null-mask
	if err != nil {
		t.Fatal(err)
	}
	row, err := Marshal(rows, OptSpeed) // no codecs → row-major baseline
	if err != nil {
		t.Fatal(err)
	}
	// Unlocking the columnar codecs for a struct with optional fields must
	// beat the row-major baseline. (Random high-entropy values cap the
	// absolute ratio, but columnar still wins via the presence bitmap and
	// FOR-packed present columns.)
	if len(col) >= len(row) {
		t.Fatalf("columnar null-mask %d not smaller than row-major %d", len(col), len(row))
	}
	t.Logf("n=%d  columnar=%d  row-major(OptSpeed)=%d  (%.1f%% smaller)",
		len(rows), len(col), len(row), 100*(1-float64(len(col))/float64(len(row))))
}
