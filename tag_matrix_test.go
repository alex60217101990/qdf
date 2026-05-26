package qdf

import (
	"reflect"
	"testing"
)

// Struct-tag matrix and named-type variant matrix. The reflect path
// must treat named-type aliases the same as their underlying primitive
// type, fall through correctly to the json-tag fallback when no qdf
// tag is present, and obey the qdf:"-" skip directive.

// Named-type aliases. Each underlying type gets a wrapper struct so
// the field name is identical across the matrix and only the type
// shape changes.
type namedInt int
type namedInt64 int64
type namedUint uint
type namedUint64 uint64
type namedFloat32 float32
type namedFloat64 float64
type namedString string
type namedBool bool
type namedBytes []byte

type namedHolder struct {
	I   namedInt     `qdf:"i"`
	I64 namedInt64   `qdf:"i64"`
	U   namedUint    `qdf:"u"`
	U64 namedUint64  `qdf:"u64"`
	F32 namedFloat32 `qdf:"f32"`
	F64 namedFloat64 `qdf:"f64"`
	S   namedString  `qdf:"s"`
	B   namedBool    `qdf:"b"`
	BS  namedBytes   `qdf:"bs"`
}

func TestTagMatrix_NamedTypes(t *testing.T) {
	in := namedHolder{
		I:   1,
		I64: -2,
		U:   3,
		U64: 4,
		F32: 5.5,
		F64: -6.25,
		S:   "named-string",
		B:   true,
		BS:  namedBytes{0x01, 0x02, 0x03},
	}
	for label, opts := range map[string]Options{"Speed": OptSpeed, "QPack": OptQPack, "Balanced": OptBalanced} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		var out namedHolder
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("%s decode: %v", label, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("%s: %+v != %+v", label, in, out)
		}
	}
}

// Tag fallback chain: qdf tag wins, then json tag, then field name.
type tagFallback struct {
	A int `qdf:"a"`
	B int `json:"bee"`
	C int // no tag — Go field name "C" lower-cased? Stays "C".
}

func TestTagMatrix_FallbackChain(t *testing.T) {
	in := tagFallback{A: 1, B: 2, C: 3}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out tagFallback
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 1 || out.B != 2 || out.C != 3 {
		t.Fatalf("got %+v want %+v", out, in)
	}
	// Confirm via a map-shaped decode that the keys match expectations.
	var m map[string]int
	if err := Unmarshal(buf, &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != 1 {
		t.Fatalf("a key missing: %v", m)
	}
	if m["bee"] != 2 {
		t.Fatalf("json fallback failed: %v", m)
	}
	if m["C"] != 3 {
		t.Fatalf("field-name fallback failed: %v", m)
	}
}

// qdf:"-" skip directive. The skipped field must never appear on the
// wire, and the decoder must leave the destination field at its zero
// value even if the source was non-zero.
type tagSkip struct {
	Keep int `qdf:"keep"`
	Skip int `qdf:"-"`
}

func TestTagMatrix_SkipDirective(t *testing.T) {
	in := tagSkip{Keep: 1, Skip: 999}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out tagSkip
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.Keep != in.Keep {
		t.Fatalf("Keep lost: %+v", out)
	}
	if out.Skip != 0 {
		t.Fatalf("Skip leaked through wire: %+v", out)
	}
	// Confirm the wire really does not carry the skipped field name.
	var m map[string]any
	if err := Unmarshal(buf, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["-"]; ok {
		t.Fatalf("literal '-' key present: %v", m)
	}
	if _, ok := m["Skip"]; ok {
		t.Fatalf("Skip field present on wire: %v", m)
	}
}

// Documented behaviour: qdf does NOT flatten anonymous embedded
// struct fields the way encoding/json does. The embedded type
// becomes a single nested object under its type name. Callers who
// need flattening should declare fields explicitly. Test pins the
// behaviour so it cannot drift silently.
type embeddedInner struct {
	A int `qdf:"a"`
	B int `qdf:"b"`
}
type embeddedOuter struct {
	embeddedInner
	C int `qdf:"c"`
}

func TestTagMatrix_EmbeddedStruct_NotFlattened(t *testing.T) {
	in := embeddedOuter{embeddedInner: embeddedInner{A: 1, B: 2}, C: 3}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	// Decode through map[string]any to inspect the actual wire keys.
	var m map[string]any
	if err := Unmarshal(buf, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["a"]; ok {
		t.Fatalf("unexpectedly flattened: inner field 'a' surfaced in outer map: %v", m)
	}
	if _, ok := m["c"]; !ok {
		t.Fatalf("outer field 'c' missing: %v", m)
	}
}

// Pointer fields: nil pointer round-trips to nil, non-nil to the same
// value. The encoder writes WriteNil for the former.
type ptrHolder struct {
	P    *int    `qdf:"p"`
	Nilp *string `qdf:"nilp"`
}

func TestTagMatrix_PointerFields(t *testing.T) {
	v := 7
	in := ptrHolder{P: &v, Nilp: nil}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out ptrHolder
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.P == nil || *out.P != 7 {
		t.Fatalf("non-nil pointer lost: %+v", out)
	}
	if out.Nilp != nil {
		t.Fatalf("nil pointer became non-nil: %+v", out)
	}
}

// Fixed-size array vs slice. Arrays have a known length at compile
// time, slices do not. Both must round-trip independently of each
// other.
type arrayHolder struct {
	A [4]int     `qdf:"a"`
	S []int      `qdf:"s"`
	F [3]float64 `qdf:"f"`
}

func TestTagMatrix_FixedArrayAndSlice(t *testing.T) {
	in := arrayHolder{
		A: [4]int{1, 2, 3, 4},
		S: []int{5, 6, 7},
		F: [3]float64{0.5, 1.5, 2.5},
	}
	for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatal(err)
		}
		var out arrayHolder
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("array/slice mismatch: %+v vs %+v", in, out)
		}
	}
}
