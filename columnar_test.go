package qdf

import (
	"reflect"
	"testing"
)

type colElig struct {
	A int     `qdf:"a"`
	B uint32  `qdf:"b"`
	C float64 `qdf:"c"`
	D bool    `qdf:"d"`
	E string  `qdf:"e"`
}

type colInelig struct {
	A int
	B []string // nested slice field → ineligible
}

func TestColumnar_Eligibility(t *testing.T) {
	eligTd, err := descOf(reflect.TypeFor[colElig]())
	if err != nil {
		t.Fatal(err)
	}
	plan := buildColumnarPlan(eligTd)
	if plan == nil {
		t.Fatal("flat scalar+string struct must be columnar-eligible")
	}
	if len(plan.cols) != 5 {
		t.Fatalf("want 5 columns, got %d", len(plan.cols))
	}
	if plan.cols[0].kind != colKindInt || plan.cols[4].kind != colKindString {
		t.Fatalf("kind classification wrong: %+v", plan.cols)
	}

	inTd, err := descOf(reflect.TypeFor[colInelig]())
	if err != nil {
		t.Fatal(err)
	}
	if buildColumnarPlan(inTd) != nil {
		t.Fatal("struct with a slice field must be ineligible")
	}
}

func TestColumnar_ShapeTable(t *testing.T) {
	st := newEncState()
	kinds := []colKind{colKindInt, colKindString}
	id1 := st.colShapeDeclare([]string{"a", "b"}, kinds)
	if id1 != 1 {
		t.Fatalf("first columnar shape id = %d, want 1", id1)
	}
	if got := st.colShapeFor([]string{"a", "b"}, kinds); got != 1 {
		t.Fatalf("reuse lookup = %d, want 1", got)
	}
	if got := st.colShapeFor([]string{"x"}, []colKind{colKindInt}); got != 0 {
		t.Fatal("unknown shape must return 0")
	}

	d := newDecState()
	sh := d.colShapeDeclareDec([]string{"a", "b"}, kinds)
	if sh == nil || len(sh.names) != 2 || sh.kinds[1] != colKindString {
		t.Fatalf("decoder columnar shape wrong: %+v", sh)
	}
	if d.colShapeLookup(1) == nil {
		t.Fatal("decoder lookup by id=1 must hit")
	}
}
