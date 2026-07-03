package qdf

import (
	"reflect"
	"strings"
	"testing"
)

type bpDoc struct {
	ID   int64 `qdf:"id"`
	Name Str   `qdf:"name"`
	Blob Bytes `qdf:"blob"`
	At   Time  `qdf:"at"`
	OK   bool  `qdf:"ok"`
}

type bpNested struct {
	Inner bpDoc
	N     [4]int32
}

type bpBadString struct {
	Name string `qdf:"name"`
}

type bpBadNested struct {
	In struct{ M map[string]int }
}

func TestBatchPlanValid(t *testing.T) {
	p, err := batchPlanOf(reflect.TypeFor[bpDoc]())
	if err != nil {
		t.Fatalf("bpDoc: %v", err)
	}
	if len(p.fields) != 5 {
		t.Fatalf("fields = %d, want 5", len(p.fields))
	}
	if p.fields[1].kind != bfStr || p.fields[1].name != "name" {
		t.Fatalf("field[1] = %+v", p.fields[1])
	}
	// v1 SCOPE DECISION: nested named structs are validated pointer-free but
	// rejected with a clear v1-fallback error (see batch_desc.go doc comment).
	_, err = batchPlanOf(reflect.TypeFor[bpNested]())
	if err == nil || !strings.Contains(err.Error(), "Inner") ||
		(!strings.Contains(err.Error(), "v1") && !strings.Contains(err.Error(), "fallback")) {
		t.Fatalf("want v1-fallback error naming field Inner, got %v", err)
	}
}

func TestBatchPlanRejects(t *testing.T) {
	_, err := batchPlanOf(reflect.TypeFor[bpBadString]())
	if err == nil || !strings.Contains(err.Error(), "Name") ||
		!strings.Contains(err.Error(), "qdf.Str") {
		t.Fatalf("want error naming field Name suggesting qdf.Str, got %v", err)
	}
	if _, err := batchPlanOf(reflect.TypeFor[bpBadNested]()); err == nil ||
		!strings.Contains(err.Error(), "In.M") {
		t.Fatalf("want nested path In.M, got %v", err)
	}
	type badTime struct {
		T2 struct{ Sec int64 }
		TT any
	}
	if _, err := batchPlanOf(reflect.TypeFor[badTime]()); err == nil {
		t.Fatal("interface field must be rejected")
	}
}

func TestBatchPlanCached(t *testing.T) {
	a, _ := batchPlanOf(reflect.TypeFor[bpDoc]())
	b, _ := batchPlanOf(reflect.TypeFor[bpDoc]())
	if a != b {
		t.Fatal("plan not cached")
	}
}

type bpEmbSkipInner struct {
	X int64 `qdf:"x"`
}

type bpEmbSkip struct {
	bpEmbSkipInner `qdf:"-"`
	Y              int64 `qdf:"y"`
}

// TestBatchPlanEmbeddedSkipTag: a `qdf:"-"` tag on an anonymous embedded
// field opts the whole nested block out, exactly as the encoder's
// appendStructFields does — without the skip the plan's field set diverges
// from the wire (spurious X the encoder never emits).
func TestBatchPlanEmbeddedSkipTag(t *testing.T) {
	p, err := batchPlanOf(reflect.TypeFor[bpEmbSkip]())
	if err != nil {
		t.Fatalf("bpEmbSkip: %v", err)
	}
	if len(p.fields) != 1 || p.fields[0].name != "y" {
		t.Fatalf("fields = %+v, want exactly [y]", p.fields)
	}
}
