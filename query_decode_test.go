package qdf

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type qrow struct {
	TS    int64  `qdf:"ts"`
	Level string `qdf:"level"`
	Code  int32  `qdf:"code"`
}

func mkQRows(n int) []qrow {
	out := make([]qrow, n)
	lv := []string{"INFO", "WARN", "ERROR"}
	for i := range out {
		out[i] = qrow{TS: int64(1000 + i), Level: lv[i%3], Code: int32(200 + i%4*100)}
	}
	return out
}

// qrowNullCode mirrors qrow but with a nullable Code, to exercise a
// wire/plan nullability mismatch through the columnar query (scatter) path.
type qrowNullCode struct {
	TS    int64  `qdf:"ts"`
	Level string `qdf:"level"`
	Code  *int32 `qdf:"code"`
}

func mkQRowsNull(n int) []qrowNullCode {
	out := make([]qrowNullCode, n)
	lv := []string{"INFO", "WARN", "ERROR"}
	for i := range out {
		c := int32(200 + i%4*100)
		out[i] = qrowNullCode{TS: int64(1000 + i), Level: lv[i%3], Code: &c}
	}
	return out
}

// safeUnmarshalErr runs Unmarshal and converts a panic into an error so a test
// can assert "rejected cleanly, did not panic".
func safeUnmarshalErr(data []byte, out any, opts ...QueryOption) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic: " + fmt.Sprint(r))
		}
	}()
	return Unmarshal(data, out, opts...)
}

// TestQuery_NullabilityMismatchRejected pins that the columnar query (scatter)
// path rejects a wire/plan nullability mismatch with ErrTypeMismatch instead of
// panicking (reflect.SliceOf(nil)) or corrupting a *T field slot. The full-decode
// path already did this; the query path used a base()-only kind compare.
func TestQuery_NullabilityMismatchRejected(t *testing.T) {
	// wire non-nullable Code → plan nullable *int32.
	enc, err := Marshal(mkQRows(32), OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var out []qrowNullCode
	if err := safeUnmarshalErr(enc, &out, Where("level", func(s string) bool { return s == "ERROR" })); !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("non-nullable→nullable: err=%v, want ErrTypeMismatch (no panic/corruption)", err)
	}

	// wire nullable *int32 → plan non-nullable int32 (the panic direction).
	enc2, err := Marshal(mkQRowsNull(32), OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var out2 []qrow
	if err := safeUnmarshalErr(enc2, &out2, Where("level", func(s string) bool { return s == "ERROR" })); !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("nullable→non-nullable: err=%v, want ErrTypeMismatch (no panic)", err)
	}
}

func TestQuery_ZeroOptionsEqualsPlainUnmarshal(t *testing.T) {
	rows := mkQRows(100)
	enc, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var a, b []qrow
	if err := Unmarshal(enc, &a); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(enc, &b /* no opts */); err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("len %d != %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs", i)
		}
	}
}

func TestQuery_NonColumnarPayloadErrUnsupported(t *testing.T) {
	// A single struct (not a slice) is never columnar.
	enc, err := Marshal(qrow{TS: 1, Level: "INFO", Code: 200}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []qrow
	err = Unmarshal(enc, &out, Where("code", func(c int32) bool { return c >= 300 }))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want wraps ErrUnsupported", err)
	}
}

func TestQuery_SinglePredicateSubset(t *testing.T) {
	rows := mkQRows(300)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrow
	if err := Unmarshal(enc, &got, Where("level", func(s string) bool { return s == "ERROR" })); err != nil {
		t.Fatal(err)
	}
	var want []qrow
	for _, r := range rows {
		if r.Level == "ERROR" {
			want = append(want, r)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d: %+v != %+v", i, got[i], want[i])
		}
	}
}

func TestQuery_MultiPredicateAnd(t *testing.T) {
	rows := mkQRows(300)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrow
	err := Unmarshal(enc, &got,
		Where("level", func(s string) bool { return s == "ERROR" }),
		Where("code", func(c int32) bool { return c >= 400 }))
	if err != nil {
		t.Fatal(err)
	}
	var want []qrow
	for _, r := range rows {
		if r.Level == "ERROR" && r.Code >= 400 {
			want = append(want, r)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d mismatch", i)
		}
	}
}

type qrowSub struct {
	TS int64 `qdf:"ts"`
}

func TestQuery_FilterFieldAbsentFromOutput(t *testing.T) {
	rows := mkQRows(300)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrowSub // no Level column in the output struct
	if err := Unmarshal(enc, &got, Where("level", func(s string) bool { return s == "ERROR" })); err != nil {
		t.Fatal(err)
	}
	var want []int64
	for _, r := range rows {
		if r.Level == "ERROR" {
			want = append(want, r.TS)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i].TS != want[i] {
			t.Fatalf("row %d: ts %d != %d", i, got[i].TS, want[i])
		}
	}
}

func TestQuery_WithAndWithoutIndexIdentical(t *testing.T) {
	rows := mkQRows(257)
	pred := func() QueryOption { return Where("level", func(s string) bool { return s == "WARN" }) }
	encIdx, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	encNoIdx, _ := Marshal(rows, OptBalanced) // no index → decode-and-discard skip path
	var a, b []qrow
	if err := Unmarshal(encIdx, &a, pred()); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(encNoIdx, &b, pred()); err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("len %d != %d (index vs no-index)", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs between index/no-index", i)
		}
	}
}

func TestQuery_NoIndexSkipsNonProjectedColumn(t *testing.T) {
	rows := mkQRows(300)
	enc, _ := Marshal(rows, OptBalanced) // NO OptColumnIndex → skip via decode-and-discard
	var got []qrowSub                    // projects only ts; level is the filter, code is neither
	if err := Unmarshal(enc, &got, Where("level", func(s string) bool { return s == "ERROR" })); err != nil {
		t.Fatal(err)
	}
	var want []int64
	for _, r := range rows {
		if r.Level == "ERROR" {
			want = append(want, r.TS)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i].TS != want[i] {
			t.Fatalf("row %d: ts %d != %d", i, got[i].TS, want[i])
		}
	}
}

func TestQuery_MapOutput(t *testing.T) {
	rows := mkQRows(200)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var out []map[string]any
	err := Unmarshal(enc, &out,
		Where("level", func(s string) bool { return s == "ERROR" }),
		Select("ts", "level"))
	if err != nil {
		t.Fatal(err)
	}
	var want []qrow
	for _, r := range rows {
		if r.Level == "ERROR" {
			want = append(want, r)
		}
	}
	if len(out) != len(want) {
		t.Fatalf("len %d != %d", len(out), len(want))
	}
	for i := range want {
		if out[i]["level"].(string) != "ERROR" {
			t.Fatalf("row %d level = %v", i, out[i]["level"])
		}
		if _, hasCode := out[i]["code"]; hasCode {
			t.Fatalf("row %d: code should be projected out", i)
		}
		if out[i]["ts"].(int64) != want[i].TS {
			t.Fatalf("row %d ts mismatch", i)
		}
	}
}

type qrowOpt struct {
	ID    int64  `qdf:"id"`
	Score *int64 `qdf:"score"` // nullable column
	Level string `qdf:"level"`
}

func mkQRowsOpt(n int) []qrowOpt {
	out := make([]qrowOpt, n)
	for i := range out {
		out[i].ID = int64(i)
		out[i].Level = []string{"INFO", "ERROR"}[i%2]
		if i%3 != 0 { // some rows nil
			v := int64(i * 10)
			out[i].Score = &v
		}
	}
	return out
}

func TestQuery_NullableColumnNilExcluded(t *testing.T) {
	rows := mkQRowsOpt(120)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrowOpt
	// Predicate on the nullable Score column: only present, >100 match.
	if err := Unmarshal(enc, &got, Where("score", func(v int64) bool { return v > 100 })); err != nil {
		t.Fatal(err)
	}
	var want []qrowOpt
	for _, r := range rows {
		if r.Score != nil && *r.Score > 100 {
			want = append(want, r)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Level != want[i].Level {
			t.Fatalf("row %d scalar mismatch", i)
		}
		if (got[i].Score == nil) != (want[i].Score == nil) {
			t.Fatalf("row %d nil-ness mismatch", i)
		}
		if got[i].Score != nil && *got[i].Score != *want[i].Score {
			t.Fatalf("row %d score %d != %d", i, *got[i].Score, *want[i].Score)
		}
	}
}

func TestQuery_NullableProjectedNotFiltered(t *testing.T) {
	// Nullable column is projected (in output) but not a predicate; filter on Level.
	rows := mkQRowsOpt(90)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrowOpt
	if err := Unmarshal(enc, &got, Where("level", func(s string) bool { return s == "ERROR" })); err != nil {
		t.Fatal(err)
	}
	var want []qrowOpt
	for _, r := range rows {
		if r.Level == "ERROR" {
			want = append(want, r)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Level != want[i].Level {
			t.Fatalf("row %d scalar mismatch", i)
		}
		if (got[i].Score == nil) != (want[i].Score == nil) {
			t.Fatalf("row %d nil-ness mismatch: got %v want %v", i, got[i].Score, want[i].Score)
		}
		if got[i].Score != nil && *got[i].Score != *want[i].Score {
			t.Fatalf("row %d score mismatch", i)
		}
	}
}

func TestQuery_TypeMismatch(t *testing.T) {
	rows := mkQRows(50)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrow
	// level is a string column; an int predicate must be a typed error.
	err := Unmarshal(enc, &got, Where("level", func(v int) bool { return v > 0 }))
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("err = %v, want wraps ErrTypeMismatch", err)
	}
	var qe *QueryError
	if !errors.As(err, &qe) || qe.Field != "level" {
		t.Fatalf("errors.As gave %+v", qe)
	}
}

func TestQuery_FieldNotFound(t *testing.T) {
	rows := mkQRows(50)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var got []qrow
	err := Unmarshal(enc, &got, Where("nonesuch", func(v int) bool { return true }))
	if !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("err = %v, want wraps ErrFieldNotFound", err)
	}
}

func TestQuery_MalformedNoPanic(t *testing.T) {
	rows := mkQRows(40)
	enc, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	for i := range enc {
		m := append([]byte(nil), enc...)
		m[i] ^= 0xFF
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on corrupted byte %d: %v", i, r)
				}
			}()
			var got []qrow
			_ = Unmarshal(m, &got,
				Where("level", func(s string) bool { return s == "ERROR" }),
				Where("code", func(c int32) bool { return c >= 400 }))
			var mp []map[string]any
			_ = Unmarshal(m, &mp, Where("level", func(s string) bool { return s == "INFO" }), Select("ts"))
		}()
	}
}

// TestQuery_NullableMalformedNoPanic byte-flips every position of a payload with
// a nullable (*int64) column and confirms the nullable query paths
// (decodeNullableColumnVals via runQueryColumns, struct and map output) bound
// every buffer index rather than panicking on hostile input.
func TestQuery_NullableMalformedNoPanic(t *testing.T) {
	rows := mkQRowsOpt(40)
	for _, opt := range []Options{OptBalanced, OptBalanced | OptColumnIndex} {
		enc, _ := Marshal(rows, opt)
		for i := range enc {
			m := append([]byte(nil), enc...)
			m[i] ^= 0xFF
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic on corrupted byte %d (opt %d): %v", i, opt, r)
					}
				}()
				var got []qrowOpt
				_ = Unmarshal(m, &got,
					Where("score", func(v int64) bool { return v > 100 }),
					Where("level", func(s string) bool { return s == "ERROR" }))
				var mp []map[string]any
				_ = Unmarshal(m, &mp,
					Where("score", func(v int64) bool { return v > 50 }), Select("id", "score"))
			}()
		}
	}
}

func TestQueryOrNot(t *testing.T) {
	type Ev struct {
		Level string `qdf:"level"`
		Code  int32  `qdf:"code"`
	}
	rows := make([]Ev, 60)
	for i := range rows {
		switch i % 3 {
		case 0:
			rows[i] = Ev{Level: "ERROR", Code: int32(i)}
		case 1:
			rows[i] = Ev{Level: "WARN", Code: int32(i)}
		default:
			rows[i] = Ev{Level: "INFO", Code: int32(i)}
		}
	}
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}

	// (ERROR OR (WARN AND code>=30))
	var got []Ev
	err = Unmarshal(buf, &got,
		Or(
			Where("level", func(s string) bool { return s == "ERROR" }),
			And(
				Where("level", func(s string) bool { return s == "WARN" }),
				Where("code", func(c int32) bool { return c >= 30 }),
			),
		))
	if err != nil {
		t.Fatal(err)
	}
	var want []Ev
	for _, r := range rows {
		if r.Level == "ERROR" || (r.Level == "WARN" && r.Code >= 30) {
			want = append(want, r)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Or/And mismatch:\n got %v\nwant %v", got, want)
	}

	// NOT(level == "INFO")
	var notInfo []Ev
	err = Unmarshal(buf, &notInfo, Not(Where("level", func(s string) bool { return s == "INFO" })))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range notInfo {
		if r.Level == "INFO" {
			t.Fatalf("NOT kept an INFO row: %v", r)
		}
	}
	cnt := 0
	for _, r := range rows {
		if r.Level != "INFO" {
			cnt++
		}
	}
	if len(notInfo) != cnt {
		t.Fatalf("NOT count = %d, want %d", len(notInfo), cnt)
	}
}

func TestQueryMultiLeafSameColumn(t *testing.T) {
	type Row struct {
		Code int32 `qdf:"code"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].Code = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	// code < 5 OR code >= 35 — same column, two leaves, decoded once.
	var got []Row
	if err := Unmarshal(buf, &got,
		Or(
			Where("code", func(c int32) bool { return c < 5 }),
			Where("code", func(c int32) bool { return c >= 35 }),
		)); err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("multi-leaf same column = %d rows, want 10", len(got))
	}
}

func TestQueryNotCompound(t *testing.T) {
	type Ev struct {
		Level string `qdf:"level"`
		Code  int32  `qdf:"code"`
	}
	rows := make([]Ev, 60)
	for i := range rows {
		switch i % 3 {
		case 0:
			rows[i] = Ev{Level: "ERROR", Code: int32(i)}
		case 1:
			rows[i] = Ev{Level: "WARN", Code: int32(i)}
		default:
			rows[i] = Ev{Level: "INFO", Code: int32(i)}
		}
	}
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	// NOT(level == "ERROR" OR code < 10) == DeMorgan: level != "ERROR" AND code >= 10
	var got []Ev
	if err := Unmarshal(buf, &got, Not(Or(
		Where("level", func(s string) bool { return s == "ERROR" }),
		Where("code", func(c int32) bool { return c < 10 }),
	))); err != nil {
		t.Fatal(err)
	}
	var want []Ev
	for _, r := range rows {
		if r.Level != "ERROR" && r.Code >= 10 {
			want = append(want, r)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Not(Or) mismatch: got %d rows, want %d", len(got), len(want))
	}
}

func TestQueryOrNotMapParity(t *testing.T) {
	type Ev struct {
		Level string `qdf:"level"`
		Code  int32  `qdf:"code"`
	}
	rows := make([]Ev, 60)
	for i := range rows {
		if i%2 == 0 {
			rows[i] = Ev{Level: "ERROR", Code: int32(i)}
		} else {
			rows[i] = Ev{Level: "INFO", Code: int32(i)}
		}
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)

	q := Or(
		Where("level", func(s string) bool { return s == "ERROR" }),
		Where("code", func(c int32) bool { return c >= 50 }),
	)
	var typed []Ev
	if err := Unmarshal(buf, &typed, q, Select("level", "code")); err != nil {
		t.Fatal(err)
	}
	var asMap []map[string]any
	if err := Unmarshal(buf, &asMap, q, Select("level", "code")); err != nil {
		t.Fatal(err)
	}
	if len(typed) != len(asMap) || len(typed) == 0 {
		t.Fatalf("map/typed row count differs: %d vs %d", len(typed), len(asMap))
	}
	for i := range typed {
		if asMap[i]["level"].(string) != typed[i].Level {
			t.Fatalf("row %d level mismatch", i)
		}
	}
}

func TestQueryNullableThreeValued(t *testing.T) {
	type Row struct {
		P *int32 `qdf:"p"`
	}
	rows := make([]Row, 30)
	for i := range rows {
		if i%2 == 0 {
			v := int32(i)
			rows[i].P = &v
		} // odd rows: nil
	}
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	// Where(p>=0): nil rows excluded.
	var pos []Row
	if err := Unmarshal(buf, &pos, Where("p", func(v int32) bool { return v >= 0 })); err != nil {
		t.Fatal(err)
	}
	for _, r := range pos {
		if r.P == nil {
			t.Fatal("Where kept a nil row")
		}
	}
	// Not(Where(p>=0)): three-valued — nil is UNKNOWN, still excluded.
	var notPos []Row
	if err := Unmarshal(buf, &notPos, Not(Where("p", func(v int32) bool { return v >= 0 }))); err != nil {
		t.Fatal(err)
	}
	for _, r := range notPos {
		if r.P == nil {
			t.Fatal("Not(Where) resurrected a nil row — violates 3VL")
		}
	}
	// p>=0 is true for all present rows, so NOT excludes everything.
	if len(notPos) != 0 {
		t.Fatalf("Not(p>=0) kept %d rows, want 0", len(notPos))
	}

	// Mixed: Not(p>10) must KEEP present-but-false rows (p<=10) while still
	// excluding nil (UNKNOWN) — proves Not over a nullable leaf isn't empty for
	// the wrong reason.
	var notHigh []Row
	if err := Unmarshal(buf, &notHigh, Not(Where("p", func(v int32) bool { return v > 10 }))); err != nil {
		t.Fatal(err)
	}
	var want int
	for _, r := range rows {
		if r.P != nil && *r.P <= 10 { // present AND p<=10
			want++
		}
	}
	if len(notHigh) != want || want == 0 {
		t.Fatalf("Not(p>10) kept %d rows, want %d (present-and-false only)", len(notHigh), want)
	}
	for _, r := range notHigh {
		if r.P == nil || *r.P > 10 {
			t.Fatalf("Not(p>10) kept a wrong row: %v", r.P)
		}
	}
}

// FuzzQueryOrNotByteFlip byte-flips every position of a well-formed payload and
// confirms the OR/NOT predicate-tree decode path never panics or races on
// corrupted input. Returning an error is fine; crashing is not.
func FuzzQueryOrNotByteFlip(f *testing.F) {
	type Row struct {
		A int32  `qdf:"a"`
		B string `qdf:"b"`
		P *int32 `qdf:"p"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i] = Row{A: int32(i), B: "x"}
		if i%2 == 0 {
			v := int32(i)
			rows[i].P = &v
		}
	}
	// Seed with index variant (primary path).
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(buf)
	// Seed with no-index variant (skip/decode-and-discard path).
	bufNoIdx, err := Marshal(rows, OptBalanced)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(bufNoIdx)
	f.Fuzz(func(t *testing.T, data []byte) {
		var out []Row
		// Must never panic, never race, regardless of corruption.
		_ = Unmarshal(data, &out,
			Or(
				Where("a", func(v int32) bool { return v >= 10 }),
				Not(Where("b", func(s string) bool { return s == "x" })),
			),
			Where("p", func(v int32) bool { return v >= 0 }),
		)
	})
}
