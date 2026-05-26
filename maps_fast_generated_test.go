package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// maps_fast_generated_test.go covers every (K, V) pair the codegen
// emits. For each pair we exercise:
//
//   - round-trip on a non-empty map keeps every entry intact
//   - nil map → encoded as nil → decodes back to nil
//   - empty map → encodes to a 0-entry map and decodes back as such
//   - dispatch table actually returns the typed pair for the matching
//     reflect.Type
//
// The pair list mirrors internal/mapsgen/main.go. Tests are
// parameter-driven via a table to keep additions cheap.

// dispatchInstalled returns true if the typed-map fast path is
// registered for t. The reflect path falls through to (nil, nil,
// false) otherwise, which is what we exercise here.
func dispatchInstalled(t reflect.Type) bool {
	enc, dec, ok := installMapFastPath(t)
	return ok && enc != nil && dec != nil
}

func mapRoundTrip(t *testing.T, in any) {
	t.Helper()
	for _, opts := range []Options{OptSpeed, OptBalanced} {
		b, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("Marshal(opts=%v): %v", opts, err)
		}
		out := reflect.New(reflect.TypeOf(in)).Interface()
		if err := Unmarshal(b, out); err != nil {
			t.Fatalf("Unmarshal(opts=%v): %v", opts, err)
		}
		got := reflect.ValueOf(out).Elem().Interface()
		if !reflect.DeepEqual(in, got) {
			t.Fatalf("round-trip mismatch (opts=%v):\nwant %#v\ngot  %#v", opts, in, got)
		}
	}
}

func TestMapFast_StringString(t *testing.T) {
	mapRoundTrip(t,map[string]string{"a": "1", "b": "two", "c": ""})
}

func TestMapFast_StringBool(t *testing.T) {
	mapRoundTrip(t,map[string]bool{"on": true, "off": false})
}

func TestMapFast_StringInt8(t *testing.T) {
	mapRoundTrip(t,map[string]int8{"a": -1, "b": 0, "c": 127, "d": -128})
}

func TestMapFast_StringInt16(t *testing.T) {
	mapRoundTrip(t,map[string]int16{"a": -32768, "b": 0, "c": 32767})
}

func TestMapFast_StringInt32(t *testing.T) {
	mapRoundTrip(t,map[string]int32{"a": -2147483648, "b": 0, "c": 2147483647})
}

func TestMapFast_StringInt(t *testing.T) {
	mapRoundTrip(t,map[string]int{"a": -1, "b": 0, "c": 1 << 20})
}

func TestMapFast_StringInt64(t *testing.T) {
	mapRoundTrip(t,map[string]int64{"a": -1 << 40, "b": 0, "c": 1 << 40})
}

func TestMapFast_StringUint8(t *testing.T) {
	mapRoundTrip(t,map[string]uint8{"a": 0, "b": 255})
}

func TestMapFast_StringUint16(t *testing.T) {
	mapRoundTrip(t,map[string]uint16{"a": 0, "b": 65535})
}

func TestMapFast_StringUint32(t *testing.T) {
	mapRoundTrip(t,map[string]uint32{"a": 0, "b": 1 << 31})
}

func TestMapFast_StringUint(t *testing.T) {
	mapRoundTrip(t,map[string]uint{"a": 0, "b": 1 << 20})
}

func TestMapFast_StringUint64(t *testing.T) {
	mapRoundTrip(t,map[string]uint64{"a": 0, "b": 1 << 60})
}

func TestMapFast_StringFloat32(t *testing.T) {
	mapRoundTrip(t,map[string]float32{"a": 0, "b": 3.14, "c": -1.5})
}

func TestMapFast_StringFloat64(t *testing.T) {
	mapRoundTrip(t,map[string]float64{"a": 0, "b": 3.141592653589793, "c": -1.5})
}

func TestMapFast_StringBytes(t *testing.T) {
	in := map[string][]byte{
		"a": []byte("hello"),
		"b": {},
		"c": []byte("world"),
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string][]byte
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got) != len(in) {
		t.Fatalf("len mismatch: want %d got %d", len(in), len(got))
	}
	for k, v := range in {
		if !bytes.Equal(got[k], v) {
			t.Fatalf("key %q: want %q got %q", k, v, got[k])
		}
	}
}

func TestMapFast_StringStringSlice(t *testing.T) {
	mapRoundTrip(t,map[string][]string{
		"tags":   {"red", "blue", "green"},
		"empty":  {},
		"single": {"only"},
	})
}

// map[string]any uses qdf's decodeAny which returns positive ints
// as uint64 regardless of the input's Go type (see decodeAny in
// reflect_encode.go). The test sticks to values that round-trip with
// stable Go types: string, float64, bool, []byte.
func TestMapFast_StringAny(t *testing.T) {
	in := map[string]any{
		"s": "string",
		"f": 3.14,
		"b": true,
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("mismatch:\nwant %#v\ngot  %#v", in, got)
	}
}

func TestMapFast_IntString(t *testing.T) {
	mapRoundTrip(t,map[int]string{1: "one", 2: "two", -5: "neg"})
}

func TestMapFast_IntInt(t *testing.T) {
	mapRoundTrip(t,map[int]int{1: 100, 2: 200, -3: -300})
}

func TestMapFast_IntInt64(t *testing.T) {
	mapRoundTrip(t,map[int]int64{1: 1 << 40, 2: -1 << 40})
}

func TestMapFast_IntAny(t *testing.T) {
	in := map[int]any{1: "a", 2: 3.14, -3: true}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[int]any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("mismatch:\nwant %#v\ngot  %#v", in, got)
	}
}

func TestMapFast_Int64String(t *testing.T) {
	mapRoundTrip(t,map[int64]string{1 << 40: "big", -1 << 40: "neg"})
}

func TestMapFast_Int64Int64(t *testing.T) {
	mapRoundTrip(t,map[int64]int64{1: 1 << 40, 2: -1 << 40})
}

func TestMapFast_Int64Any(t *testing.T) {
	in := map[int64]any{42: "answer", -7: 3.14, 99: true}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[int64]any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("mismatch:\nwant %#v\ngot  %#v", in, got)
	}
}

func TestMapFast_Uint64String(t *testing.T) {
	mapRoundTrip(t,map[uint64]string{1 << 60: "huge", 0: "zero"})
}

func TestMapFast_Uint64Uint64(t *testing.T) {
	mapRoundTrip(t,map[uint64]uint64{1: 1 << 60, 2: 0})
}

func TestMapFast_Uint64Any(t *testing.T) {
	in := map[uint64]any{1: "a", 2: 3.14, 3: false}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[uint64]any
	if err := Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("mismatch:\nwant %#v\ngot  %#v", in, got)
	}
}

// Nil-map handling: every pair must encode a nil map as tagNil and
// decode it back to a nil map (not an empty map). Cover the full
// pair list to guard against a future generator change that emits a
// `make()` short-circuit before the nil check.
func TestMapFast_NilEncoding(t *testing.T) {
	cases := []any{
		map[string]string(nil),
		map[string]bool(nil),
		map[string]int8(nil),
		map[string]int16(nil),
		map[string]int32(nil),
		map[string]int(nil),
		map[string]int64(nil),
		map[string]uint8(nil),
		map[string]uint16(nil),
		map[string]uint32(nil),
		map[string]uint(nil),
		map[string]uint64(nil),
		map[string]float32(nil),
		map[string]float64(nil),
		map[string][]byte(nil),
		map[string][]string(nil),
		map[string]any(nil),
		map[int]string(nil),
		map[int]int(nil),
		map[int]int64(nil),
		map[int]any(nil),
		map[int64]string(nil),
		map[int64]int64(nil),
		map[int64]any(nil),
		map[uint64]string(nil),
		map[uint64]uint64(nil),
		map[uint64]any(nil),
	}
	for _, in := range cases {
		t.Run(reflect.TypeOf(in).String(), func(t *testing.T) {
			b, err := Marshal(in, OptBalanced)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			out := reflect.New(reflect.TypeOf(in)).Interface()
			if err := Unmarshal(b, out); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			got := reflect.ValueOf(out).Elem()
			if !got.IsNil() {
				t.Fatalf("nil map decoded to non-nil: %v", got.Interface())
			}
		})
	}
}

// Empty-map handling: a 0-entry, non-nil map must round-trip without
// degenerating to nil.
func TestMapFast_EmptyRoundTrip(t *testing.T) {
	cases := []any{
		map[string]string{},
		map[string]int64{},
		map[int]string{},
		map[uint64]any{},
	}
	for _, in := range cases {
		t.Run(reflect.TypeOf(in).String(), func(t *testing.T) {
			b, err := Marshal(in, OptBalanced)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			out := reflect.New(reflect.TypeOf(in)).Interface()
			if err := Unmarshal(b, out); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			got := reflect.ValueOf(out).Elem()
			if got.IsNil() {
				t.Fatalf("empty map decoded as nil")
			}
			if got.Len() != 0 {
				t.Fatalf("empty map decoded with %d entries", got.Len())
			}
		})
	}
}

// Dispatch coverage: every pair listed by the generator must resolve
// to a non-nil enc/dec pair via reflect.Type lookup. Guards against
// the generator silently skipping an entry.
func TestMapFast_DispatchCoverage(t *testing.T) {
	cases := []reflect.Type{
		reflect.TypeFor[map[string]string](),
		reflect.TypeFor[map[string]bool](),
		reflect.TypeFor[map[string]int8](),
		reflect.TypeFor[map[string]int16](),
		reflect.TypeFor[map[string]int32](),
		reflect.TypeFor[map[string]int](),
		reflect.TypeFor[map[string]int64](),
		reflect.TypeFor[map[string]uint8](),
		reflect.TypeFor[map[string]uint16](),
		reflect.TypeFor[map[string]uint32](),
		reflect.TypeFor[map[string]uint](),
		reflect.TypeFor[map[string]uint64](),
		reflect.TypeFor[map[string]float32](),
		reflect.TypeFor[map[string]float64](),
		reflect.TypeFor[map[string][]byte](),
		reflect.TypeFor[map[string][]string](),
		reflect.TypeFor[map[string]any](),
		reflect.TypeFor[map[int]string](),
		reflect.TypeFor[map[int]int](),
		reflect.TypeFor[map[int]int64](),
		reflect.TypeFor[map[int]any](),
		reflect.TypeFor[map[int64]string](),
		reflect.TypeFor[map[int64]int64](),
		reflect.TypeFor[map[int64]any](),
		reflect.TypeFor[map[uint64]string](),
		reflect.TypeFor[map[uint64]uint64](),
		reflect.TypeFor[map[uint64]any](),
	}
	for _, ty := range cases {
		if !dispatchInstalled(ty) {
			t.Errorf("%s: dispatch missing", ty)
		}
	}
}

// Unregistered map types must fall through to the reflect path
// (installMapFastPath returns ok=false). Pick a shape we deliberately
// do NOT generate so the negative case is durable.
func TestMapFast_DispatchFallthrough(t *testing.T) {
	if dispatchInstalled(reflect.TypeFor[map[float64]string]()) {
		t.Error("map[float64]string should NOT have a generated fast path")
	}
	// Reflect path still handles it correctly.
	in := map[float64]string{1.5: "a", 2.5: "b"}
	mapRoundTrip(t,in)
}
