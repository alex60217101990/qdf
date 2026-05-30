package qdf

import (
	"math/rand"
	"testing"
)

type nullStrRow struct {
	Seq   int64   // sequential → columnar engages
	Label *string // optional enum-ish string
	Note  *string // optional, higher cardinality
}

func mkNullStrRows(n, nullPct int, seed int64) []nullStrRow {
	r := rand.New(rand.NewSource(seed))
	labels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	out := make([]nullStrRow, n)
	for i := range out {
		out[i].Seq = int64(i)
		if r.Intn(100) >= nullPct {
			v := labels[r.Intn(len(labels))]
			out[i].Label = &v
		}
		if r.Intn(100) >= nullPct {
			v := "note-" + labels[r.Intn(len(labels))]
			out[i].Note = &v
		}
	}
	return out
}

func eqNullStrRow(a, b nullStrRow) bool {
	if a.Seq != b.Seq {
		return false
	}
	eq := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return eq(a.Label, b.Label) && eq(a.Note, b.Note)
}

// RED until nullable string columns ship: a struct with optional string
// fields must use the columnar path (presence bitmap + dense string column)
// and come out far smaller than the row-major baseline, while round-tripping
// nils and values exactly.
func TestNullableString_RoundTripAndSmaller(t *testing.T) {
	for _, nullPct := range []int{0, 40, 100} {
		rows := mkNullStrRows(2000, nullPct, int64(nullPct+1))
		col, err := Marshal(rows, OptBalanced&^OptRANS)
		if err != nil {
			t.Fatalf("null%%=%d: %v", nullPct, err)
		}
		var got []nullStrRow
		if err := Unmarshal(col, &got); err != nil {
			t.Fatalf("null%%=%d unmarshal: %v", nullPct, err)
		}
		if len(got) != len(rows) {
			t.Fatalf("len %d != %d", len(got), len(rows))
		}
		for i := range rows {
			if !eqNullStrRow(got[i], rows[i]) {
				t.Fatalf("null%%=%d row %d mismatch: %+v != %+v", nullPct, i, got[i], rows[i])
			}
		}
		// Discriminator: the same data with the string fields made non-optional
		// (value, not pointer) is columnar-eligible today and encodes tightly.
		// The optional version must come within a small factor of it — proving
		// nullable string columns engage instead of falling back to row-major
		// (where it is ~10× larger). OptBalanced-Dense already beats OptSpeed via
		// interning even when row-major, so OptSpeed is NOT a valid baseline here.
		type vrow struct {
			Seq   int64
			Label string
			Note  string
		}
		vr := make([]vrow, len(rows))
		for i := range rows {
			vr[i].Seq = rows[i].Seq
			if rows[i].Label != nil {
				vr[i].Label = *rows[i].Label
			}
			if rows[i].Note != nil {
				vr[i].Note = *rows[i].Note
			}
		}
		ref, err := Marshal(vr, OptBalanced&^OptRANS)
		if err != nil {
			t.Fatal(err)
		}
		if len(col) >= 4*len(ref) {
			t.Fatalf("null%%=%d: nullable-string columnar %d not within 4x of value-string columnar %d (row-major fallback?)", nullPct, len(col), len(ref))
		}
		t.Logf("null%%=%d nullable=%d value-columnar=%d", nullPct, len(col), len(ref))
	}
}

func TestNullableString_Any(t *testing.T) {
	rows := mkNullStrRows(500, 40, 9)
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var a any
	if err := Unmarshal(enc, &a); err != nil {
		t.Fatal(err)
	}
	got, ok := a.([]any)
	if !ok || len(got) != len(rows) {
		t.Fatalf("any shape %T", a)
	}
	for i := range rows {
		m := got[i].(map[string]any)
		if rows[i].Label == nil {
			if m["Label"] != nil {
				t.Fatalf("row %d Label want nil got %v", i, m["Label"])
			}
		} else if m["Label"].(string) != *rows[i].Label {
			t.Fatalf("row %d Label %v != %s", i, m["Label"], *rows[i].Label)
		}
	}
}
