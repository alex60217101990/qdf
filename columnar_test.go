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
	eligTd, err := descOf(reflect.TypeOf(colElig{}))
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

	inTd, err := descOf(reflect.TypeOf(colInelig{}))
	if err != nil {
		t.Fatal(err)
	}
	if buildColumnarPlan(inTd) != nil {
		t.Fatal("struct with a slice field must be ineligible")
	}
}
