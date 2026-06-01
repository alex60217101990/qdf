package qdf

import (
	"fmt"
	"reflect"
	"testing"
)

// TestCompression_TypeMatrix roundtrips a struct containing every non-string
// type class (ints, uints, floats, bools, map, pointer) through the three
// heavy codec bundles that were previously only size-tested. A decode bug in
// OptCompression / OptRANS on these types would pass CI before this test.
func TestCompression_TypeMatrix(t *testing.T) {
	type mixed struct {
		Ints    []int64        `qdf:"ints"`
		Uints   []uint64       `qdf:"uints"`
		Floats  []float64      `qdf:"floats"`
		Bools   []bool         `qdf:"bools"`
		Strs    []string       `qdf:"strs"`
		NestedM map[string]int `qdf:"nested_m"`
		Ptr     *int64         `qdf:"ptr"`
	}
	n := int64(42)
	v := mixed{
		Ints:    []int64{1, 1, 1, 2, 3, 3, 1000000, -1000000},
		Uints:   []uint64{0, 0, 0, 1 << 40},
		Floats:  []float64{0.1, 0.1, 3.14159, 0.1},
		Bools:   []bool{true, true, false, true},
		Strs:    []string{"a", "a", "b", "a", "c"},
		NestedM: map[string]int{"x": 1, "y": 2},
		Ptr:     &n,
	}
	for _, opts := range []Options{OptCompression, OptDense | OptRANS, OptQPack | OptGorillaFloat} {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			data, err := Marshal(v, opts)
			if err != nil {
				t.Fatalf("opts=%d marshal: %v", opts, err)
			}
			var out mixed
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("opts=%d unmarshal: %v", opts, err)
			}
			if !reflect.DeepEqual(v, out) {
				t.Fatalf("opts=%d roundtrip mismatch:\n in=%#v\nout=%#v", opts, v, out)
			}
		})
	}
}
