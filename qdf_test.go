package qdf

import (
	"bytes"
	"math"
	"reflect"
	"testing"
	"time"
)

type Inner struct {
	X int     `qdf:"x"`
	Y float64 `qdf:"y"`
}

type Outer struct {
	Name   string            `qdf:"name"`
	Age    int               `qdf:"age"`
	Active bool              `qdf:"active"`
	Score  float64           `qdf:"score"`
	Tags   []string          `qdf:"tags"`
	Meta   map[string]string `qdf:"meta"`
	Inner  Inner             `qdf:"inner"`
	When   time.Time         `qdf:"when"`
	Buf    []byte            `qdf:"buf"`
	OptPtr *Inner            `qdf:"opt"`
	Counts [3]int32          `qdf:"counts"`
}

func roundTrip[T any](t *testing.T, mode Mode, v T) T {
	t.Helper()
	var b []byte
	var err error
	switch mode {
	case Fast:
		b, err = Marshal(v, OptSpeed)
	case Dense:
		b, err = Marshal(v, OptBalanced)
	}
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out T
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestPrimitives_RoundTrip(t *testing.T) {
	cases := []any{
		true, false,
		int(0), int(1), int(-1), int(-8), int(-9), int(127), int(128),
		int(math.MaxInt32), int(math.MinInt32),
		int64(math.MaxInt64), int64(math.MinInt64),
		uint64(math.MaxUint64),
		float32(3.14), float64(2.71828),
		"", "hello", "a much longer string to exercise str8 path " + string(make([]byte, 300)),
		[]byte("binary"),
		(*int)(nil),
	}
	for _, c := range cases {
		t.Run(reflect.TypeOf(c).String(), func(t *testing.T) {
			b, err := Marshal(c, OptSpeed)
			if err != nil {
				t.Fatalf("encode %T: %v", c, err)
			}
			// Decode into a fresh value of the same type.
			out := reflect.New(reflect.TypeOf(c)).Interface()
			if err := Unmarshal(b, out); err != nil {
				t.Fatalf("decode %T: %v", c, err)
			}
			got := reflect.ValueOf(out).Elem().Interface()
			if !reflect.DeepEqual(got, c) {
				t.Fatalf("round-trip mismatch for %T: got %v want %v", c, got, c)
			}
		})
	}
}

func TestStruct_RoundTrip_Fast(t *testing.T) {
	in := Outer{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Score:  98.6,
		Tags:   []string{"a", "b", "c"},
		Meta:   map[string]string{"k1": "v1", "k2": "v2"},
		Inner:  Inner{X: 7, Y: 1.5},
		When:   time.Unix(1700000000, 0),
		Buf:    []byte{1, 2, 3, 4, 5},
		OptPtr: &Inner{X: 99, Y: -2.0},
		Counts: [3]int32{10, 20, 30},
	}
	out := roundTrip(t, Fast, in)
	if !equalOuter(in, out) {
		t.Fatalf("mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestStruct_RoundTrip_Dense(t *testing.T) {
	in := Outer{
		Name:   "Bob",
		Tags:   []string{"repeat", "repeat", "repeat", "unique"},
		Meta:   map[string]string{"country": "LT", "city": "Vilnius"},
		Inner:  Inner{X: 1, Y: 2},
		When:   time.Unix(1700000000, 0),
		Counts: [3]int32{1, 2, 3},
	}
	out := roundTrip(t, Dense, in)
	if !equalOuter(in, out) {
		t.Fatalf("mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestDense_ReducesSize(t *testing.T) {
	type Row struct {
		Country string `qdf:"country"`
		City    string `qdf:"city"`
		Pop     int    `qdf:"pop"`
	}
	rows := make([]Row, 50)
	for i := range rows {
		rows[i] = Row{Country: "Lithuania", City: "Vilnius", Pop: 580000 + i}
	}
	fast, err := Marshal(rows, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	dense, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("fast=%d dense=%d ratio=%.2f", len(fast), len(dense), float64(len(dense))/float64(len(fast)))
	if len(dense) >= len(fast) {
		t.Fatalf("expected dense < fast: fast=%d dense=%d", len(fast), len(dense))
	}
	// Round-trip the dense form to be sure.
	var got []Row
	if err := Unmarshal(dense, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows, got) {
		t.Fatalf("dense round-trip mismatch")
	}
}

func TestEmptyAndNil(t *testing.T) {
	type S struct {
		A []int          `qdf:"a"`
		B map[string]int `qdf:"b"`
		C *int           `qdf:"c"`
		D string         `qdf:"d"`
	}
	in := S{}
	b, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out S
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.A == nil && in.A != nil {
		t.Fatal("expected nil A")
	}
	if out.D != "" {
		t.Fatal("expected empty D")
	}
}

func equalOuter(a, b Outer) bool {
	if a.Name != b.Name || a.Age != b.Age || a.Active != b.Active || a.Score != b.Score {
		return false
	}
	if !reflect.DeepEqual(a.Tags, b.Tags) {
		return false
	}
	if !reflect.DeepEqual(a.Meta, b.Meta) {
		return false
	}
	if a.Inner != b.Inner {
		return false
	}
	if !a.When.Equal(b.When) {
		return false
	}
	if !bytes.Equal(a.Buf, b.Buf) {
		return false
	}
	if (a.OptPtr == nil) != (b.OptPtr == nil) {
		return false
	}
	if a.OptPtr != nil && *a.OptPtr != *b.OptPtr {
		return false
	}
	if a.Counts != b.Counts {
		return false
	}
	return true
}
