package qdf

import (
	"errors"
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
