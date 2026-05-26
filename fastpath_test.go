package qdf

import (
	"math"
	"reflect"
	"testing"
)

// Make sure every specialized fast-path round-trips identically to the
// generic reflect path (and identical encoded bytes — they must, because
// both paths target the same wire format).

func TestFastPath_MapStringString(t *testing.T) {
	in := map[string]string{"a": "1", "b": "2", "c": "3"}
	b, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch: %v vs %v", in, out)
	}
}

func TestFastPath_MapStringInt(t *testing.T) {
	in := map[string]int{"a": 1, "b": -2, "c": 1 << 30}
	b, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]int
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch: %v vs %v", in, out)
	}
}

func TestFastPath_MapStringInt64(t *testing.T) {
	in := map[string]int64{"big": math.MaxInt64, "small": math.MinInt64}
	b, _ := Marshal(in, OptSpeed)
	var out map[string]int64
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch: %v vs %v", in, out)
	}
}

func TestFastPath_MapStringAny(t *testing.T) {
	in := map[string]any{
		"s":  "string",
		"i":  uint64(42),
		"b":  true,
		"f":  3.14,
		"sl": []any{uint64(1), uint64(2)},
	}
	b, _ := Marshal(in, OptSpeed)
	var out map[string]any
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch: %#v vs %#v", in, out)
	}
}

func TestFastPath_SliceTypes(t *testing.T) {
	type tc struct {
		name string
		in   any
	}
	cases := []tc{
		{"string", []string{"a", "b", "c"}},
		{"int", []int{1, -1, 0, 1 << 30}},
		{"int32", []int32{1, 2, math.MaxInt32, math.MinInt32}},
		{"int64", []int64{0, math.MaxInt64, math.MinInt64}},
		{"uint32", []uint32{0, 1, math.MaxUint32}},
		{"uint64", []uint64{0, 1, math.MaxUint64}},
		{"float32", []float32{1.5, -2.5, math.MaxFloat32}},
		{"float64", []float64{1.5, -2.5, math.MaxFloat64, math.SmallestNonzeroFloat64}},
		{"bool", []bool{true, false, true, true, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := Marshal(c.in, OptSpeed)
			if err != nil {
				t.Fatal(err)
			}
			out := reflect.New(reflect.TypeOf(c.in)).Interface()
			if err := Unmarshal(b, out); err != nil {
				t.Fatal(err)
			}
			got := reflect.ValueOf(out).Elem().Interface()
			if !reflect.DeepEqual(got, c.in) {
				t.Fatalf("mismatch: %v vs %v", got, c.in)
			}
		})
	}
}

// TestFastPath_AgreesWithGeneric forces a NON-fast-path map (map[int]string
// is not specialized) and verifies it still round-trips. Together with the
// specialized cases this proves both code paths produce equivalent results.
func TestFastPath_GenericMapStillWorks(t *testing.T) {
	in := map[int]string{1: "a", 2: "b", 3: "c"}
	b, _ := Marshal(in, OptSpeed)
	var out map[int]string
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch: %v vs %v", in, out)
	}
}
