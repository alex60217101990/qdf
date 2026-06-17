package qdf

import (
	"math"
	"reflect"
	"testing"
	"time"
)

// This file exercises the FULL option matrix (matrixBundles) over deliberately
// complex, mixed-type datasets — every integer/float width, strings, []byte,
// fixed [N]byte arrays, time.Time, embedded + nested structs, nil and non-nil
// pointers, slices, arrays, maps with several key kinds, nested maps, and (for
// the columnar path) a wide []struct with an enum string column, a sequential
// integer column, a float column, a nullable *T column, and a time column.
// Every bundle must encode→decode back to a value identical to the input.

type mxInner struct {
	Code  int16  `qdf:"code"`
	Label string `qdf:"label"`
}

type mxEmbedded struct {
	Flag bool   `qdf:"flag"`
	Note string `qdf:"note"`
}

// mxComplex is a row-major torture struct: it carries a map/slice mix so the
// columnar probe declines it, keeping it on the row-major path under every
// bundle. Floats are chosen so reflect.DeepEqual is exact (no NaN; -0.0 and Inf
// are handled by ==).
type mxComplex struct {
	mxEmbedded // embedded

	I8  int8  `qdf:"i8"`
	I16 int16 `qdf:"i16"`
	I32 int32 `qdf:"i32"`
	I64 int64 `qdf:"i64"`
	I   int   `qdf:"i"`

	U8  uint8  `qdf:"u8"`
	U16 uint16 `qdf:"u16"`
	U32 uint32 `qdf:"u32"`
	U64 uint64 `qdf:"u64"`
	U   uint   `qdf:"u"`

	F32  float32 `qdf:"f32"`
	F64  float64 `qdf:"f64"`
	NegZ float64 `qdf:"negz"` // -0.0 — preserved bit-for-bit outside OptCanonical
	Inf  float64 `qdf:"inf"`  // +Inf

	B     bool     `qdf:"b"`
	S     string   `qdf:"s"`
	Bytes []byte   `qdf:"bytes"`
	Fixed [16]byte `qdf:"fixed"`

	When time.Time `qdf:"when"`

	Ptr    *int     `qdf:"ptr"`    // non-nil
	NilPtr *string  `qdf:"nilptr"` // nil
	Inner  mxInner  `qdf:"inner"`  // nested value
	PInner *mxInner `qdf:"pinner"` // nested pointer

	Ints   []int64   `qdf:"ints"`
	Strs   []string  `qdf:"strs"`
	Floats []float64 `qdf:"floats"`
	Bytes2 [][]byte  `qdf:"bytes2"`
	Arr    [3]int    `qdf:"arr"`
	Inners []mxInner `qdf:"inners"`

	MS    map[string]int   `qdf:"ms"`
	MI    map[int64]string `qdf:"mi"`
	MU    map[uint8]bool   `qdf:"mu"`
	MB    map[bool]string  `qdf:"mb"`
	MNest map[string][]int `qdf:"mnest"`
}

func newMxComplex() mxComplex {
	n := 42
	return mxComplex{
		mxEmbedded: mxEmbedded{Flag: true, Note: "embedded"},
		I8:         -128, I16: -32768, I32: -2147483648, I64: -9223372036854775808, I: -1000000,
		U8: 255, U16: 65535, U32: 4294967295, U64: 18446744073709551615, U: 1000000,
		F32:    math.MaxFloat32,
		F64:    3.141592653589793,
		NegZ:   math.Copysign(0, -1),
		Inf:    math.Inf(1),
		B:      true,
		S:      "the quick brown fox jumps over the lazy dog",
		Bytes:  []byte{0, 1, 2, 254, 255, 128},
		Fixed:  [16]byte{0xde, 0xad, 0xbe, 0xef, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		When:   time.Date(2026, 6, 17, 10, 30, 0, 123456789, time.UTC),
		Ptr:    &n,
		Inner:  mxInner{Code: 7, Label: "inner"},
		PInner: &mxInner{Code: 9, Label: "pinner"},
		Ints:   []int64{1, 2, 3, -4, 5, 1 << 40},
		Strs:   []string{"alpha", "beta", "gamma", ""},
		Floats: []float64{1.5, -2.25, 0, math.Inf(-1)},
		Bytes2: [][]byte{{1, 2}, {}, {255}},
		Arr:    [3]int{10, 20, 30},
		Inners: []mxInner{{1, "a"}, {2, "b"}, {3, "c"}},
		MS:     map[string]int{"x": 1, "y": 2, "z": 3},
		MI:     map[int64]string{10: "ten", 20: "twenty"},
		MU:     map[uint8]bool{1: true, 2: false, 255: true},
		MB:     map[bool]string{true: "t", false: "f"},
		MNest:  map[string][]int{"a": {1, 2}, "b": {3}},
	}
}

func TestMatrix_ComplexRowMajor(t *testing.T) {
	roundtripBundles(t, newMxComplex())
}

// mxEvent is a pure-columnar row: no maps or nested slices, so the columnar
// probe accepts a []mxEvent and every bundle (incl. ColIndex / FSST) takes the
// transpose path. It mixes a sequential int column, two enum string columns, a
// float column, a bool column, a fixed-byte column, a time column, and a
// nullable *int column.
type mxEvent struct {
	TS      int64     `qdf:"ts"`      // sequential → Delta+FOR
	Seq     uint32    `qdf:"seq"`     // sequential
	Level   string    `qdf:"level"`   // enum → dictionary
	Service string    `qdf:"service"` // enum → dictionary / FSST
	Lat     float64   `qdf:"lat"`     // float column (Gorilla under Compression)
	Ok      bool      `qdf:"ok"`      // bool bit-pack
	Count   int32     `qdf:"count"`   // FOR
	Tag     [8]byte   `qdf:"tag"`     // fixed-byte column
	When    time.Time `qdf:"when"`    // time sec/nsec sub-columns
	Opt     *int      `qdf:"opt"`     // nullable column (presence bitmap + dense)
}

func newMxEvents() []mxEvent {
	levels := []string{"INFO", "WARN", "ERROR"}
	services := []string{"api", "auth", "billing", "cache"}
	base := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	rows := make([]mxEvent, 40)
	for i := range rows {
		var opt *int
		if i%3 == 0 {
			v := i * 7
			opt = &v // some rows present, others nil
		}
		rows[i] = mxEvent{
			TS:      int64(1_700_000_000 + i),
			Seq:     uint32(i),
			Level:   levels[i%len(levels)],
			Service: services[(i*3)%len(services)],
			Lat:     float64(i) * 0.5,
			Ok:      i%2 == 0,
			Count:   int32(i * i),
			Tag:     [8]byte{byte(i), 0xAA, 0xBB, byte(i + 1), 0, 0, 0, byte(i + 2)},
			When:    base.Add(time.Duration(i) * time.Second),
			Opt:     opt,
		}
	}
	return rows
}

func TestMatrix_ComplexColumnar(t *testing.T) {
	roundtripColumnar(t, newMxEvents())
}

// TestMatrix_ComplexColumnarSmall covers the sub-columnar-threshold path: the
// same rich row type but only a few rows, so the columnar probe declines and the
// rows decode row-major under every bundle. Catches row-major/columnar
// divergence for the same type.
func TestMatrix_ComplexColumnarSmall(t *testing.T) {
	roundtripColumnar(t, newMxEvents()[:3])
}

// TestMatrix_ComplexDelta runs the structural-delta feature (Diff/Apply) over
// the full option matrix on the complex columnar dataset: a change scattered
// across several columns (float, string, int, and the nullable *int — both a
// new value and a clear-to-nil) is diffed under each bundle and applied back
// onto a copy of the base, which must reconstruct the updated value exactly.
func TestMatrix_ComplexDelta(t *testing.T) {
	old := newMxEvents()

	updated := append([]mxEvent(nil), old...)
	updated[5].Lat = 999.5      // float column
	updated[10].Level = "ERROR" // enum string column
	updated[20].Count = -1      // int column
	updated[30].Ok = !updated[30].Ok
	n := 123
	updated[0].Opt = &n  // nullable: set a value where there was one
	updated[3].Opt = nil // nullable: clear a previously-present value
	m := 7
	updated[4].Opt = &m // nullable: set where it was nil

	for _, b := range matrixBundles() {
		t.Run(b.name, func(t *testing.T) {
			patch, err := Diff(old, updated, b.opts)
			if err != nil {
				t.Fatalf("diff %s: %v", b.name, err)
			}
			base := append([]mxEvent(nil), old...)
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("apply %s: %v", b.name, err)
			}
			if !reflect.DeepEqual(updated, base) {
				t.Fatalf("%s delta roundtrip mismatch", b.name)
			}
		})
	}
}
