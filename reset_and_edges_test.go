package qdf

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// Encoder/Decoder state-reset coverage: confirm Reset clears every
// piece of state the encoder relies on so a pool-recycled instance
// cannot leak prior-call data into the next call. Pointer-edge and
// misc-shape tests live alongside since they share the same risk
// category: state hidden behind a pool/cache.

// --- Encoder.Reset coverage -----------------------------------------

func TestReset_EncoderBufferTruncated(t *testing.T) {
	e := NewEncoder(Dense)
	e.WriteString("first-payload-1234567")
	if len(e.buf) == 0 {
		t.Fatal("buf empty after write")
	}
	e.Reset()
	if len(e.buf) != 0 {
		t.Fatalf("buf not truncated: len=%d", len(e.buf))
	}
}

func TestReset_EncoderInternTableCleared(t *testing.T) {
	e := NewEncoder(Dense)
	e.WriteString("region-eu-west-1")
	if int(e.state.internLoad) == 0 {
		t.Fatal("intern table empty after WriteString")
	}
	e.Reset()
	if int(e.state.internLoad) != 0 {
		t.Fatalf("intern table not cleared: %d entries", int(e.state.internLoad))
	}
}

func TestReset_EncoderMarkov0Cleared(t *testing.T) {
	e := NewEncoder(Dense)
	e.WriteString("token-aaa-bbb-ccc")
	if e.state.lastID == lruInvalidID {
		t.Fatal("lastID not set after WriteString")
	}
	e.Reset()
	if e.state.lastID != lruInvalidID {
		t.Fatal("lastID not cleared by Reset")
	}
}

func TestReset_EncoderDepthCleared(t *testing.T) {
	// Force the encoder into a state where depth has incremented,
	// then Reset and confirm depth is back to 0.
	type box struct {
		Inner *struct {
			V int `qdf:"v"`
		} `qdf:"inner"`
	}
	e := NewEncoder(Fast)
	e.depth = 5 // simulate mid-recursion
	e.Reset()
	if e.depth != 0 {
		t.Fatalf("depth not reset: %d", e.depth)
	}
	_ = box{}
}

func TestReset_EncoderHeaderFlagCleared(t *testing.T) {
	e := NewEncoder(Fast)
	e.WriteBool(true)
	if !e.headerOut {
		t.Fatal("headerOut should be true after first write")
	}
	e.Reset()
	if e.headerOut {
		t.Fatal("headerOut should be false after Reset")
	}
}

func TestReset_PooledEncoderNoLeak(t *testing.T) {
	// Encode through Marshal twice with different payloads. Confirm
	// the pooled encoder does not retain bytes from call 1 in call 2.
	a, err := Marshal(map[string]int{"alpha": 1}, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Marshal(map[string]int{"beta": 2}, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a, b) {
		t.Fatalf("two distinct payloads produced same wire: %x", a)
	}
	if string(a[5:]) == string(b[5:]) {
		t.Fatalf("pool leaked bytes between calls")
	}
}

// --- Pointer edges --------------------------------------------------

func TestPtrEdges_PtrToPtr(t *testing.T) {
	v := 42
	pv := &v
	ppv := &pv
	type holder struct {
		P **int `qdf:"p"`
	}
	in := holder{P: ppv}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out holder
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.P == nil || *out.P == nil || **out.P != 42 {
		t.Fatalf("ptr-to-ptr round-trip failed: %+v", out)
	}
}

func TestPtrEdges_SliceOfPtr(t *testing.T) {
	a, b := 1, 2
	type holder struct {
		PS []*int `qdf:"ps"`
	}
	in := holder{PS: []*int{&a, nil, &b, nil}}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out holder
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.PS) != 4 {
		t.Fatalf("len: %d", len(out.PS))
	}
	if out.PS[0] == nil || *out.PS[0] != 1 {
		t.Fatalf("[0]: %v", out.PS[0])
	}
	if out.PS[1] != nil {
		t.Fatalf("[1] should be nil: %v", *out.PS[1])
	}
	if out.PS[2] == nil || *out.PS[2] != 2 {
		t.Fatalf("[2]: %v", out.PS[2])
	}
	if out.PS[3] != nil {
		t.Fatalf("[3] should be nil")
	}
}

func TestPtrEdges_MapStringPtr(t *testing.T) {
	v := 42
	type holder struct {
		M map[string]*int `qdf:"m"`
	}
	in := holder{M: map[string]*int{"k": &v, "nil": nil}}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out holder
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.M["k"] == nil || *out.M["k"] != 42 {
		t.Fatalf("k: %v", out.M["k"])
	}
	if out.M["nil"] != nil {
		t.Fatalf("nil entry came back non-nil")
	}
}

// --- Misc edges -----------------------------------------------------

func TestMisc_MapWithIntKey(t *testing.T) {
	// Non-string map keys. msgpack supports them; verify qdf either
	// supports or rejects cleanly.
	in := map[int]string{1: "one", 2: "two", 3: "three"}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Skipf("non-string map keys not supported: %v", err)
	}
	var out map[int]string
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("got %v", out)
	}
}

func TestMisc_TimeEdges(t *testing.T) {
	// qdf encodes time.Time as int64 nanoseconds since the Unix
	// epoch. Documented range: roughly 1677-09-21 to 2262-04-11.
	// Values outside that range cannot round-trip — caller's
	// responsibility, like the stdlib time/Unix functions. Test
	// pins the supported range.
	cases := []time.Time{
		time.Unix(0, 0).UTC(),
		time.Unix(1, 0).UTC(),
		time.Unix(-1, 0).UTC(),
		time.Unix(1<<30, 0).UTC(),  // year 2004
		time.Unix(1<<31, 0).UTC(),  // year 2038, the famous one
		time.Unix(-1<<30, 0).UTC(), // year 1935
		time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1700, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, in := range cases {
		buf, err := Marshal(in, OptSpeed)
		if err != nil {
			t.Errorf("encode %v: %v", in, err)
			continue
		}
		var out time.Time
		if err := Unmarshal(buf, &out); err != nil {
			t.Errorf("decode %v: %v", in, err)
			continue
		}
		if !in.Equal(out) {
			t.Errorf("got %v want %v", out, in)
		}
	}
}

func TestMisc_EmptyStructRoot(t *testing.T) {
	type empty struct{}
	in := empty{}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out empty
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
}

func TestMisc_UnicodeKeys(t *testing.T) {
	// Multi-byte UTF-8 keys inside a Dense intern table.
	in := map[string]int{
		"привет": 1,
		"こんにちは":  2,
		"السلام": 3,
		"hello":  4,
	}
	for label, opts := range map[string]Options{
		"Speed":    OptSpeed,
		"Balanced": OptBalanced,
	} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		var out map[string]int
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("%s decode: %v", label, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("%s: %v vs %v", label, in, out)
		}
	}
}

func TestMisc_AnonymousStructType(t *testing.T) {
	in := struct {
		A int    `qdf:"a"`
		B string `qdf:"b"`
	}{A: 7, B: "anon"}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		A int    `qdf:"a"`
		B string `qdf:"b"`
	}
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 7 || out.B != "anon" {
		t.Fatalf("got %+v", out)
	}
}

func TestMisc_VeryLongFieldName(t *testing.T) {
	// Field name longer than fixstr (>31). Forces str8/str16 path.
	longName := strings.Repeat("k", 200)
	in := map[string]int{longName: 42}
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]int
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out[longName] != 42 {
		t.Fatalf("got %v", out)
	}
}
