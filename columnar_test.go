package qdf

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestColumnar_Probe(t *testing.T) {
	td, _ := descOf(reflect.TypeFor[[]colElig]())
	plan := td.colPlan
	if plan == nil {
		t.Fatal("[]colElig must have a colPlan")
	}
	// Compressible: A clustered, B constant, D constant, E repeating.
	good := make([]colElig, 256)
	for i := range good {
		good[i] = colElig{A: 1000 + i%4, B: 7, C: 1.5, D: true, E: "INFO"}
	}
	if !columnarProbe(plan, ptrToSliceData(good), len(good), false, nil) {
		t.Fatal("probe must accept a compressible struct array")
	}
	// Incompressible: every field unique/random-ish.
	bad := make([]colElig, 256)
	for i := range bad {
		bad[i] = colElig{A: i * 2654435761, B: uint32(i * 40503), C: float64(i) * 1.1, D: i%2 == 0, E: "u" + itoa(i)}
	}
	if columnarProbe(plan, ptrToSliceData(bad), len(bad), false, nil) {
		t.Fatal("probe must reject an incompressible struct array")
	}
}

func ptrToSliceData[T any](s []T) unsafe.Pointer {
	return (*sliceHeader)(unsafe.Pointer(&s)).Data
}

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

	// A struct mixing an eligible scalar with a slice field is now HYBRID
	// (the eligible column is transposed, the slice kept as a residual field)
	// instead of disqualifying the whole struct.
	inTd, err := descOf(reflect.TypeFor[colInelig]())
	if err != nil {
		t.Fatal(err)
	}
	hp := buildColumnarPlan(inTd)
	if hp == nil {
		t.Fatal("mixed scalar+slice struct must be hybrid-eligible, got nil")
	}
	if len(hp.cols) != 1 || len(hp.residual) != 1 {
		t.Fatalf("colInelig: want 1 eligible col (A) + 1 residual (B), got cols=%d residual=%d",
			len(hp.cols), len(hp.residual))
	}

	// A struct with NO eligible field is still ineligible (nil → row-major).
	type noElig struct {
		M map[string]int
		S []string
	}
	nTd, err := descOf(reflect.TypeFor[noElig]())
	if err != nil {
		t.Fatal(err)
	}
	if buildColumnarPlan(nTd) != nil {
		t.Fatal("struct with no eligible field must be ineligible (nil plan)")
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

func TestColumnar_RoundTripTyped(t *testing.T) {
	in := make([]colElig, 200)
	for i := range in {
		in[i] = colElig{A: 1000 + i%5, B: uint32(i % 3), C: 2.5, D: i%2 == 0, E: []string{"INFO", "WARN"}[i%2]}
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !containsByte(b, tagColStruct) {
		t.Fatalf("expected tagColStruct on a compressible struct array, got %x...", b[:min(48, len(b))])
	}
	var out []colElig
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestColumnar_FallbackRoundTrip(t *testing.T) {
	in := make([]colElig, 4) // below columnarMinElems → row-major
	for i := range in {
		in[i] = colElig{A: i, E: "x"}
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if containsByte(b, tagColStruct) {
		t.Fatal("below columnarMinElems must not use columnar")
	}
	var out []colElig
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("fallback round-trip mismatch")
	}
}

func TestColumnar_DecodeAny(t *testing.T) {
	in := make([]colElig, 64)
	for i := range in {
		in[i] = colElig{A: 7, B: 1, C: 3.0, D: true, E: "INFO"}
	}
	b, _ := Marshal(in, OptBalanced)
	var v any
	if err := Unmarshal(b, &v); err != nil {
		t.Fatalf("decode-any: %v", err)
	}
	arr, ok := v.([]any)
	if !ok || len(arr) != 64 {
		t.Fatalf("want []any len 64, got %T", v)
	}
	m, ok := arr[0].(map[string]any)
	if !ok || m["e"] != "INFO" {
		t.Fatalf("decode-any element wrong: %#v", arr[0])
	}
}
