package qdf

import (
	"reflect"
	"testing"
)

// fbRow is a struct with a codec of its own, standing in for a generated type.
// It implements BOTH Marshaler and Unmarshaler, which is what sends fillDesc
// down its early return with no fields — the single-direction case takes a
// different path and would leave the fields populated.
type fbRow struct {
	ID   int64  `qdf:"id"`
	Name string `qdf:"name"`
}

func (v *fbRow) MarshalQDF(dst []byte) ([]byte, error)      { return dst, nil }
func (v *fbRow) UnmarshalQDF(src []byte) (n int, err error) { return 0, nil }

// The shared descriptor of such a type has no fields, so buildColumnarPlan
// yields nil for it — that is the defect's root, asserted here rather than
// assumed. structuralColumnarPlan must produce a plan anyway, WITHOUT changing
// the shared descriptor.
func TestStructuralColumnarPlanSeesFieldsTheSharedDescriptorHides(t *testing.T) {
	et := reflect.TypeFor[fbRow]()

	shared, err := descOf(et)
	if err != nil {
		t.Fatalf("descOf: %v", err)
	}
	if len(shared.fields) != 0 {
		t.Fatalf("the shared descriptor has %d fields — this type no longer takes the "+
			"early return, and the premise of this test is gone", len(shared.fields))
	}
	if buildColumnarPlan(shared) != nil {
		t.Fatal("the shared descriptor already yields a columnar plan — the defect this " +
			"builds a fallback for does not exist as described")
	}

	plan := structuralColumnarPlan(et)
	if plan == nil {
		t.Fatal("structuralColumnarPlan returned nil for a plain two-field struct")
	}

	// The shared descriptor must be exactly as it was. This is the assertion
	// that keeps the change decode-only: if fields appeared here, the slice
	// ENCODER would start transposing this type and skipping its codec.
	again, err := descOf(et)
	if err != nil {
		t.Fatalf("descOf: %v", err)
	}
	if again != shared {
		t.Fatal("descOf returned a different descriptor — the synthetic build published to typeCache")
	}
	if len(again.fields) != 0 {
		t.Fatalf("the shared descriptor gained %d fields", len(again.fields))
	}
	if buildColumnarPlan(again) != nil {
		t.Fatal("the shared descriptor now yields a columnar plan — encoding would change")
	}
}

// A type that cannot be described structurally must yield nil rather than an
// error or a panic: the caller's contract is to fall back to today's behavior.
func TestStructuralColumnarPlanDeclinesWhatItCannotDescribe(t *testing.T) {
	type undescribable struct {
		Ch chan int `qdf:"ch"`
	}
	if plan := structuralColumnarPlan(reflect.TypeFor[undescribable]()); plan != nil {
		t.Fatal("a struct with a channel field produced a columnar plan")
	}
	if plan := structuralColumnarPlan(reflect.TypeFor[int]()); plan != nil {
		t.Fatal("a non-struct produced a columnar plan")
	}
}
