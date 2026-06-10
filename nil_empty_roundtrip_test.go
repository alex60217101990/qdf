package qdf

import (
	"reflect"
	"testing"
)

// Regression: a nil slice must round-trip as nil, distinct from an empty
// (non-nil) slice — the distinction maps and pointers already keep, and that
// encoding/json keeps as null vs []. encodeSlice* previously emitted a 0-length
// array header for both, so a nil slice decoded as []T{}. A nil slice now emits
// tagNil (via the field-level wrapper) and decodes back to nil.
func TestNilVsEmptySliceRoundTrip(t *testing.T) {
	type S struct {
		A []int            `qdf:"a"`
		B []string         `qdf:"b"`
		C []byte           `qdf:"c"`
		D []float64        `qdf:"d"`
		E [][]int          `qdf:"e"`
		F []map[string]int `qdf:"f"`
	}
	cases := []struct {
		name string
		v    any
	}{
		{"allNil", S{}},
		{"allEmpty", S{A: []int{}, B: []string{}, C: []byte{}, D: []float64{}, E: [][]int{}, F: []map[string]int{}}},
		{"mixed", S{A: nil, B: []string{}, C: []byte{1, 2}, D: nil, E: [][]int{}, F: nil}},
		{"full", S{A: []int{1}, B: []string{"x"}, C: []byte{9}, D: []float64{1.5}, E: [][]int{{1}}, F: []map[string]int{{"k": 1}}}},
		{"topNilSlice", []int(nil)},
		{"topEmptySlice", []int{}},
		{"nestedNilEmptyFull", [][]int{nil, {}, {1, 2}}},
		{"sliceOfStructs", []S{{A: nil}, {A: []int{}}, {A: []int{7}}}},
		{"nilBytes", S{C: nil}},
		{"emptyBytes", S{C: []byte{}}},
	}
	for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression, OptBalanced | OptMapShape, OptBalanced | OptColumnIndex} {
		for _, c := range cases {
			b, err := Marshal(c.v, opt)
			if err != nil {
				t.Fatalf("%s/%v marshal: %v", c.name, opt, err)
			}
			out := reflect.New(reflect.TypeOf(c.v))
			if err := Unmarshal(b, out.Interface()); err != nil {
				t.Fatalf("%s/%v unmarshal: %v", c.name, opt, err)
			}
			if !reflect.DeepEqual(c.v, out.Elem().Interface()) {
				t.Errorf("%s/%v not bit-exact:\n in =%#v\n out=%#v", c.name, opt, c.v, out.Elem().Interface())
			}
		}
	}
}
