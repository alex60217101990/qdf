package qdf

import "testing"

// namedStatus is a named type whose underlying kind is a signed integer — the
// case that silently degraded WhereCmp/WhereRange to a `field == 0` predicate
// before boundKind resolved the underlying kind via reflect.
type namedStatus int32
type namedScore uint16
type namedLabel string

type namedRow struct {
	S   namedStatus
	Sc  namedScore
	L   namedLabel
	Tag string // constant → columnar transpose
}

// TestWhereNamedTypes guards that WhereCmp/WhereRange over NAMED scalar types
// filter on the actual value, never silently on zero.
func TestWhereNamedTypes(t *testing.T) {
	const n = 2048
	rows := make([]namedRow, n)
	for i := range rows {
		s := namedStatus(0)
		if i%7 == 0 {
			s = namedStatus(5)
		}
		rows[i] = namedRow{S: s, Sc: namedScore(i % 300), L: namedLabel("k"), Tag: "c"}
	}
	b, err := Marshal(rows, OptBalanced|OptColumnIndex|OptZoneMap)
	if err != nil {
		t.Fatal(err)
	}

	check := func(name string, q QueryOption, pred func(namedRow) bool) {
		t.Helper()
		want := 0
		for _, r := range rows {
			if pred(r) {
				want++
			}
		}
		var out []namedRow
		if err := Unmarshal(b, &out, q, Select("S", "Sc", "L", "Tag")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(out) != want {
			t.Fatalf("%s: got %d rows, want %d", name, len(out), want)
		}
		for i, r := range out {
			if !pred(r) {
				t.Fatalf("%s: row %d %+v fails predicate", name, i, r)
			}
		}
	}

	// Named signed int — the silent-==0 case. EQ 5 must return the 5s, not the 0s.
	check("named-int-eq", WhereCmp("S", EQ, namedStatus(5)), func(r namedRow) bool { return r.S == 5 })
	check("named-int-ge", WhereCmp("S", GE, namedStatus(5)), func(r namedRow) bool { return r.S >= 5 })
	// Named unsigned + range.
	check("named-uint-range", WhereRange("Sc", namedScore(100), namedScore(200)),
		func(r namedRow) bool { return r.Sc >= 100 && r.Sc <= 200 })
	// Named string.
	check("named-str-eq", WhereCmp("L", EQ, namedLabel("k")), func(r namedRow) bool { return r.L == "k" })
}
