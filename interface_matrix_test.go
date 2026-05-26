package qdf

import (
	"math"
	"reflect"
	"testing"
)

// interface{} (any) field handling. The reflect path encodes the
// dynamic type of an interface field, the decoder rebuilds it through
// decodeAny. Pin the supported dynamic-type set + edge cases so a
// later refactor cannot drop a type silently.

type ifaceHolder struct {
	V any `qdf:"v"`
}

func TestInterface_DynamicTypes(t *testing.T) {
	cases := []struct {
		name string
		v    any
	}{
		{"nil", nil},
		{"bool_true", true},
		{"bool_false", false},
		{"int_zero", 0},
		{"int_neg", -7},
		{"int_max", int64(math.MaxInt64)},
		{"int_min", int64(math.MinInt64)},
		{"uint_max", uint64(math.MaxUint64)},
		{"float32", float32(1.5)},
		{"float64", float64(-2.25)},
		{"float_inf", math.Inf(1)},
		{"float_nan", math.NaN()},
		{"empty_string", ""},
		{"short_string", "hi"},
		{"long_string", "this-is-a-longer-string-past-fixstr-boundary"},
		{"bytes", []byte{1, 2, 3, 0xFF}},
		{"empty_bytes", []byte{}},
		{"string_slice", []any{"a", "b", "c"}},
		{"mixed_slice", []any{1, "two", 3.0, true}},
		{"nested_map", map[string]any{"k": "v", "n": 42}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := ifaceHolder{V: c.v}
			buf, err := Marshal(in, OptSpeed)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			var out ifaceHolder
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !ifaceEqual(in.V, out.V) {
				t.Fatalf("got %#v want %#v", out.V, in.V)
			}
		})
	}
}

// ifaceEqual compares two `any` values with three accommodations
// beyond reflect.DeepEqual:
//   - NaN matches NaN bit-for-bit.
//   - Integer-typed Go values (int, int64, uint64, etc.) compare by
//     numeric value across their native widths because decodeAny
//     reduces every signed/unsigned variant to uint64 or int64.
//   - Nested slices/maps recurse through this function.
func ifaceEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(av) && math.IsNaN(bv) {
			return true
		}
		return math.Float64bits(av) == math.Float64bits(bv)
	case float32:
		bv, ok := b.(float32)
		if !ok {
			return false
		}
		if math.IsNaN(float64(av)) && math.IsNaN(float64(bv)) {
			return true
		}
		return math.Float32bits(av) == math.Float32bits(bv)
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !ifaceEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvv, ok := bv[k]
			if !ok {
				return false
			}
			if !ifaceEqual(v, bvv) {
				return false
			}
		}
		return true
	}
	// Cross-width integer comparison: int / int8..64 / uint / uint8..64
	// all compare by numeric value.
	aInt, aIsInt, aIsUint, aU := normaliseInt(a)
	bInt, bIsInt, bIsUint, bU := normaliseInt(b)
	if (aIsInt || aIsUint) && (bIsInt || bIsUint) {
		if aIsInt && bIsInt {
			return aInt == bInt
		}
		if aIsUint && bIsUint {
			return aU == bU
		}
		if aIsInt && bIsUint {
			return aInt >= 0 && uint64(aInt) == bU
		}
		return bInt >= 0 && uint64(bInt) == aU
	}
	return reflect.DeepEqual(a, b)
}

func normaliseInt(v any) (signed int64, isInt bool, isUint bool, unsigned uint64) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true, false, 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0, false, true, rv.Uint()
	}
	return
}

func TestInterface_SliceOfAny(t *testing.T) {
	in := []any{1, "hello", 3.14, true, nil, []byte{1, 2, 3}}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out []any
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatalf("len %d vs %d", len(in), len(out))
	}
	for i := range in {
		if !ifaceEqual(in[i], out[i]) {
			t.Fatalf("[%d] %#v vs %#v", i, in[i], out[i])
		}
	}
}

func TestInterface_MapStringAny(t *testing.T) {
	in := map[string]any{
		"id":    42,
		"name":  "alice",
		"score": 3.14,
		"on":    true,
		"tags":  []any{"a", "b"},
	}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatalf("len %d vs %d", len(in), len(out))
	}
	for k, v := range in {
		if !ifaceEqual(v, out[k]) {
			t.Fatalf("[%s] %#v vs %#v", k, v, out[k])
		}
	}
}
