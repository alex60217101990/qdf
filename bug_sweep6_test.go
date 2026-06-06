package qdf

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
)

// TestEncode_InterfaceCycle_NoStackOverflow pins the depth guard on the dynamic
// (interface) encode path. encodePtr guards static *T cycles, but a cycle routed
// through an any-typed field re-entered the reflect machinery via encodeIface
// without bumping the depth counter → unbounded recursion → fatal stack
// overflow. It must now return ErrCycleDetected.
func TestEncode_InterfaceCycle_NoStackOverflow(t *testing.T) {
	type ifaceNode struct {
		V    int `qdf:"v"`
		Next any `qdf:"next"`
	}
	a := &ifaceNode{V: 1}
	a.Next = a // self-cycle through the any field
	if _, err := Marshal(a, OptBalanced); !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("want ErrCycleDetected, got %v", err)
	}
	// Mutual cycle A -> B -> A through any fields.
	b := &ifaceNode{V: 2}
	a.Next = b
	b.Next = a
	if _, err := Marshal(a, OptBalanced); !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("mutual: want ErrCycleDetected, got %v", err)
	}
}

// TestDecodeAny_PackedSlices pins that a numeric/bool slice carried by an
// any-typed field round-trips under the QPack-enabled modes. decodeAny had no
// case for the tagPack* tags and returned ErrBadTag, so any interface{} value
// holding such a slice failed to decode under the default OptBalanced.
func TestDecodeAny_PackedSlices(t *testing.T) {
	type Box struct {
		V any `qdf:"v"`
	}
	// Incompressible int32/uint32 stay raw-LE at their native width (kind
	// Int32/Uint32) rather than widening — decodeAny must handle those kinds too.
	i32 := make([]int32, 50)
	u32 := make([]uint32, 50)
	x := uint32(0x9e3779b9)
	for i := range i32 {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		i32[i] = int32(x)
		u32[i] = x
	}
	cases := []any{
		[]int64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109},
		[]uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		[]float64{1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5, 8.5, 9.5, 10.5},
		[]float32{1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5, 8.5, 9.5, 10.5},
		[]bool{true, false, true, false, true, false, true, false, true, false},
		i32,
		u32,
	}
	for _, opts := range []Options{OptQPack, OptBalanced, OptCompression} {
		for _, v := range cases {
			b, err := Marshal(Box{V: v}, opts)
			if err != nil {
				t.Fatalf("opts=%d %T marshal: %v", opts, v, err)
			}
			var out Box
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("opts=%d %T unmarshal: %v", opts, v, err)
			}
			if !reflect.DeepEqual(v, out.V) {
				t.Fatalf("opts=%d %T: decoded != original\n in =%#v\n out=%#v (%T)", opts, v, v, out.V, out.V)
			}
		}
	}

	// Float edge cases (NaN / ±Inf / -0) must round-trip BIT-exactly through the
	// any path. reflect.DeepEqual mishandles NaN, so compare via Float*bits.
	negZero64 := math.Copysign(0, -1)
	negZero32 := float32(math.Copysign(0, -1))
	f64 := []float64{math.NaN(), math.Inf(1), math.Inf(-1), negZero64, 1.5, 2.5, 3.5, 4.5}
	f32 := []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), negZero32, 1.5, 2.5, 3.5, 4.5}
	bitsEq64 := func(a, b []float64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
				return false
			}
		}
		return true
	}
	for _, opts := range []Options{OptQPack, OptBalanced, OptCompression} {
		b, _ := Marshal(Box{V: f64}, opts)
		var o64 Box
		if err := Unmarshal(b, &o64); err != nil {
			t.Fatalf("opts=%d f64 edge unmarshal: %v", opts, err)
		}
		if !bitsEq64(f64, o64.V.([]float64)) {
			t.Fatalf("opts=%d f64 edge bits NOT preserved: %#v", opts, o64.V)
		}
		b, _ = Marshal(Box{V: f32}, opts)
		var o32 Box
		if err := Unmarshal(b, &o32); err != nil {
			t.Fatalf("opts=%d f32 edge unmarshal: %v", opts, err)
		}
		if !f32bitsEqual(f32, o32.V.([]float32)) {
			t.Fatalf("opts=%d f32 edge bits NOT preserved: %#v", opts, o32.V)
		}
	}
}

// TestDecodeAny_HostileConstantSliceBounded pins that the QPack-slice support in
// decodeAny cannot be driven into a multi-GB allocation by a tiny hostile header.
// A constant-value codec (bitsPer == 0) carries an empty body, so its element
// count is bounded only by qpackMaxStandaloneCount; decoding such a slice through
// the schemaless any path (the fuzzer's map[string]any / []any target) must reject
// an over-cap count instead of allocating count*8 bytes. (This guards the OOM the
// decodeAny QPack support newly exposed to the any-decode path.)
func TestDecodeAny_HostileConstantSliceBounded(t *testing.T) {
	// QDF header + map8{1} + key "x" + value = tagPackFor, kind=uint64, bits=0,
	// min=0, n = qpackMaxStandaloneCount+1 (empty body — a ~16-byte header).
	buf := append(mkHeader(), tagMap8, 0x01)
	buf = append(buf, tagFixstr|0x01, 'x')
	buf = append(buf, tagPackFor, qpackKindUint64, 0x00, 0x00)
	buf = appendUvarint(buf, uint64(qpackMaxStandaloneCount)+1)
	var m map[string]any
	if err := Unmarshal(buf, &m); err == nil {
		t.Fatal("decodeAny accepted an over-cap constant slice (multi-GB make hazard)")
	}
	// A small constant slice still decodes fine through the any path.
	ok := append(mkHeader(), tagMap8, 0x01)
	ok = append(ok, tagFixstr|0x01, 'y')
	ok = append(ok, tagPackFor, qpackKindUint64, 0x00, 0x07) // min=7
	ok = appendUvarint(ok, 4)
	var m2 map[string]any
	if err := Unmarshal(ok, &m2); err != nil {
		t.Fatalf("small constant slice in any wrongly rejected: %v", err)
	}
	if got, _ := m2["y"].([]uint64); len(got) != 4 || got[0] != 7 {
		t.Fatalf("small constant slice in any decoded wrong: %#v", m2["y"])
	}
}

// TestOOM_DictHugeCountBoundedBeforeMake pins that a dictionary-coded integer
// slice with bitsPer > 0 bounds its element count against the remaining buffer
// BEFORE allocating. The reader previously ran make([]T, n) ahead of that check,
// so a ~tiny header (count=2 distinct values, then a huge n with no body) drove
// a multi-GB allocation — found via the any decode path the fuzzer reaches.
func TestOOM_DictHugeCountBoundedBeforeMake(t *testing.T) {
	for _, kind := range []byte{qpackKindInt64, qpackKindUint64} {
		buf := append(mkHeader(), tagPackDict, kind)
		buf = appendUvarint(buf, 2)     // 2 distinct values ⇒ bitsForDistinct(2)=1
		buf = appendUvarint(buf, 0)     // table[0] (zigzag/raw 0)
		buf = appendUvarint(buf, 2)     // table[1]
		buf = appendUvarint(buf, 1<<30) // n = 1G elements, but no index body follows
		var outI []int64
		var outU []uint64
		var err error
		if kind == qpackKindInt64 {
			err = Unmarshal(buf, &outI)
		} else {
			err = Unmarshal(buf, &outU)
		}
		if err == nil {
			t.Fatalf("kind=%#x: huge dict count accepted (multi-GB make hazard)", kind)
		}
	}
}

// TestSkip_ColumnarUnknownField pins that an unknown []struct field encoded in
// the columnar (tagColStruct) form is skipped correctly during schema-evolution
// decode. Skip had no tagColStruct case and returned ErrBadTag, aborting the
// whole decode and losing every trailing field.
func TestSkip_ColumnarUnknownField(t *testing.T) {
	type item struct {
		A int32  `qdf:"a"`
		B string `qdf:"b"`
	}
	type full struct {
		Items []item `qdf:"items"`
		Tail  string `qdf:"tail"`
	}
	in := full{Tail: "after-columnar"}
	// 1000 distinct rows so the columnar probe actually picks the columnar form
	// (tagColStruct) under both OptBalanced and OptCompression — verified below.
	for i := 0; i < 1000; i++ {
		in.Items = append(in.Items, item{A: int32(i), B: fmt.Sprintf("s%d", i)})
	}
	// A target that drops Items → the Items value routes through Skip(tagColStruct).
	type partial struct {
		Tail string `qdf:"tail"`
	}
	for _, opts := range []Options{OptBalanced, OptCompression} {
		b, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		// Guard against a silently-vacuous test: the field MUST be columnar.
		hasCol := false
		for _, x := range b {
			if x == tagColStruct {
				hasCol = true
				break
			}
		}
		if !hasCol {
			t.Fatalf("opts=%d: Items did not encode columnar — test would not exercise Skip(tagColStruct)", opts)
		}
		var out partial
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("opts=%d unmarshal (columnar skip): %v", opts, err)
		}
		if out.Tail != in.Tail {
			t.Fatalf("opts=%d: Tail=%q want %q — columnar skip desynced the stream", opts, out.Tail, in.Tail)
		}
		// Control: decoding into the FULL struct must reproduce the columnar data
		// identically (proves the payload Skip walked over is itself sound, and
		// that skipping it consumed exactly as many bytes as decoding it).
		var fullOut full
		if err := Unmarshal(b, &fullOut); err != nil {
			t.Fatalf("opts=%d full unmarshal: %v", opts, err)
		}
		if !reflect.DeepEqual(in, fullOut) {
			t.Fatalf("opts=%d: full round-trip mismatch (columnar data not identical)", opts)
		}
	}
}
