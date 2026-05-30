package qdf

import (
	"errors"
	"testing"
)

func TestQueryError_IsAndAs(t *testing.T) {
	qe := &QueryError{Op: "predicate pushdown", Field: "level", Want: colKindInt, Got: colKindString, Err: ErrTypeMismatch}

	if !errors.Is(qe, ErrTypeMismatch) {
		t.Fatal("errors.Is(qe, ErrTypeMismatch) = false, want true (Unwrap chain)")
	}
	var got *QueryError
	if !errors.As(qe, &got) {
		t.Fatal("errors.As(qe, &*QueryError) = false, want true")
	}
	if got.Field != "level" || got.Want != colKindInt || got.Got != colKindString {
		t.Fatalf("As yielded wrong fields: %+v", got)
	}
	if got.Error() == "" {
		t.Fatal("Error() returned empty string")
	}

	nf := &QueryError{Op: "predicate pushdown", Field: "nope", Err: ErrFieldNotFound}
	if !errors.Is(nf, ErrFieldNotFound) {
		t.Fatal("errors.Is(nf, ErrFieldNotFound) = false")
	}
}
