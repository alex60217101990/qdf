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
