package qdf

import (
	"reflect"
	"testing"
)

type oneCol struct {
	A int `qdf:"a"`
}

type allStr struct {
	X string `qdf:"x"`
	Y string `qdf:"y"`
}

type widthMix struct {
	I8  int8    `qdf:"i8"`
	I16 int16   `qdf:"i16"`
	I32 int32   `qdf:"i32"`
	I64 int64   `qdf:"i64"`
	U8  uint8   `qdf:"u8"`
	U16 uint16  `qdf:"u16"`
	U32 uint32  `qdf:"u32"`
	U64 uint64  `qdf:"u64"`
	F32 float32 `qdf:"f32"`
	F64 float64 `qdf:"f64"`
	B   bool    `qdf:"b"`
	S   string  `qdf:"s"`
	BB  []byte  `qdf:"bb"`
}

func rt[T any](t *testing.T, in []T, wantCol bool) {
	t.Helper()
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := containsByte(b, tagColStruct)
	if got != wantCol {
		t.Fatalf("tagColStruct present=%v want=%v (n=%d)", got, wantCol, len(in))
	}
	var out []T
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(in) == 0 {
		if len(out) != 0 {
			t.Fatalf("empty: out len %d", len(out))
		}
		return
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestReviewSingleColumn(t *testing.T) {
	in := make([]oneCol, 64)
	for i := range in {
		in[i] = oneCol{A: 100 + i%3}
	}
	rt(t, in, true)
}

func TestReviewAllString(t *testing.T) {
	in := make([]allStr, 64)
	for i := range in {
		in[i] = allStr{X: "INFO", Y: []string{"a", "b"}[i%2]}
	}
	rt(t, in, true)
}

func TestReviewExactlyMinElems(t *testing.T) {
	in := make([]oneCol, columnarMinElems) // 16
	for i := range in {
		in[i] = oneCol{A: 5}
	}
	rt(t, in, true) // at the threshold and constant -> columnar
}

func TestReviewBelowMinElems(t *testing.T) {
	in := make([]oneCol, columnarMinElems-1) // 15
	for i := range in {
		in[i] = oneCol{A: 5}
	}
	rt(t, in, false) // below threshold -> row-major
}

func TestReviewEmpty(t *testing.T) {
	rt(t, []oneCol{}, false)
}

func TestReviewWidthMix(t *testing.T) {
	in := make([]widthMix, 64)
	for i := range in {
		in[i] = widthMix{
			I8: int8(-3 + i%2), I16: -300, I32: -70000, I64: -5_000_000_000,
			U8: 200, U16: 60000, U32: 4_000_000_000, U64: 18_000_000_000_000_000_000,
			F32: 1.5, F64: 2.25, B: i%2 == 0, S: "tag", BB: []byte("xy"),
		}
	}
	rt(t, in, true)
}

func TestReviewWidthMixNegativeExtremes(t *testing.T) {
	// exercise sign-extension at min values for each signed width
	in := make([]widthMix, 32)
	for i := range in {
		in[i] = widthMix{
			I8: -128, I16: -32768, I32: -2147483648, I64: -9223372036854775808,
			U8: 255, U16: 65535, U32: 4294967295, U64: 18446744073709551615,
			F32: -3.4e38, F64: 1.7e308, B: true, S: "x", BB: nil,
		}
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []widthMix
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	// nil []byte decodes to nil; normalize for compare
	for i := range out {
		if out[i].BB == nil {
			in[i].BB = nil
		}
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("mismatch:\n in=%+v\nout=%+v", in[0], out[0])
	}
}

func TestReviewDecodeAnyMatchesTyped(t *testing.T) {
	in := make([]widthMix, 40)
	for i := range in {
		in[i] = widthMix{I8: 1, I16: 2, I32: 3, I64: int64(1000 + i%4), U8: 9, U16: 10, U32: 11, U64: 12, F32: 1.5, F64: 2.5, B: true, S: "INFO", BB: []byte("z")}
	}
	b, _ := Marshal(in, OptBalanced)
	if !containsByte(b, tagColStruct) {
		t.Skip("not columnar; skip any-comparison")
	}
	var v any
	if err := Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	arr := v.([]any)
	if len(arr) != len(in) {
		t.Fatalf("any len %d", len(arr))
	}
	m := arr[3].(map[string]any)
	if m["i64"].(int64) != int64(1003) {
		t.Fatalf("i64 any = %v", m["i64"])
	}
	if m["s"].(string) != "INFO" {
		t.Fatalf("s any = %v", m["s"])
	}
	// []byte column decoded as string under any
	if m["bb"].(string) != "z" {
		t.Fatalf("bb any = %v", m["bb"])
	}
}
