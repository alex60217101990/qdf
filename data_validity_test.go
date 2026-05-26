package qdf

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// data_validity_test.go: dedicated suite that verifies marshalled
// data round-trips with semantic AND structural fidelity — order
// preservation, edge-value preservation, bit-exact float behaviour,
// UTF-8 safety, concurrent pool safety, and pool-reuse correctness
// across mixed Options.
//
// Each test runs the round-trip under every encode profile
// (OptSpeed / OptQPack / OptBalanced) unless the assertion is opt-
// specific. The decoder is a single Unmarshal that must accept any
// emit.

// allEncodeOpts returns the option bit-masks we test against in the
// validity suite. Covers the three named bundles plus a "Dense
// without QPack" combo that the QPack-only path bypasses.
func allEncodeOpts() []struct {
	name string
	opts Options
} {
	return []struct {
		name string
		opts Options
	}{
		{"Speed", OptSpeed},
		{"QPack", OptQPack},
		{"Dense", OptDense},
		{"DenseQPack", OptDense | OptQPack},
		{"Balanced", OptBalanced},
	}
}

// runValidity encodes v under each option set, decodes into a fresh
// zero value of the same type, and asserts reflect.DeepEqual.
func runValidity(t *testing.T, v any) {
	t.Helper()
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(v, p.opts)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			outPtr := reflect.New(reflect.TypeOf(v))
			if err := Unmarshal(b, outPtr.Interface()); err != nil {
				t.Fatalf("decode (%d bytes): %v wire=%x", len(b), err, b)
			}
			out := outPtr.Elem().Interface()
			if !reflect.DeepEqual(v, out) {
				t.Fatalf("mismatch\n in=%+v\nout=%+v\nwire=%x", v, out, b)
			}
		})
	}
}

// --- Slice order preservation -----------------------------------

func TestValidity_SliceOrder_PrimitiveTypes(t *testing.T) {
	t.Run("int_slice", func(t *testing.T) {
		runValidity(t, []int{0, 1, -1, 2, -2, 100, -100, 1 << 30, -1 << 30})
	})
	t.Run("uint64_slice_monotonic", func(t *testing.T) {
		s := make([]uint64, 32)
		for i := range s {
			s[i] = uint64(1_700_000_000 + i)
		}
		runValidity(t, s)
	})
	t.Run("uint64_slice_unordered", func(t *testing.T) {
		runValidity(t, []uint64{5, 3, 8, 1, 9, 2, 7, 4, 6})
	})
	t.Run("string_slice_order", func(t *testing.T) {
		runValidity(t, []string{"zeta", "alpha", "mu", "beta", "lambda"})
	})
	t.Run("bool_slice_order", func(t *testing.T) {
		runValidity(t, []bool{true, false, true, true, false, false, true})
	})
	t.Run("float64_slice_order", func(t *testing.T) {
		runValidity(t, []float64{3.14, 2.72, 1.41, 1.62, 0.577, 4.66})
	})
}

func TestValidity_SliceOrder_Nested(t *testing.T) {
	t.Run("slice_of_slices", func(t *testing.T) {
		runValidity(t, [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8}, {}, {9}})
	})
	t.Run("slice_of_structs", func(t *testing.T) {
		type item struct {
			N int    `qdf:"n"`
			S string `qdf:"s"`
		}
		runValidity(t, []item{
			{N: 1, S: "first"},
			{N: 2, S: "second"},
			{N: 3, S: "third"},
		})
	})
}

func TestValidity_SliceOrder_LargeSpansArr32(t *testing.T) {
	// 70 000 elements crosses the arr16 → arr32 boundary at 65535.
	// The encoder must keep order intact across that header change.
	s := make([]int, 70_000)
	for i := range s {
		s[i] = i * 3
	}
	runValidity(t, s)
}

// --- Map nondeterminism + semantic equality ---------------------

func TestValidity_MapSemanticEquality(t *testing.T) {
	// Go map iteration order is randomised; semantic equality is
	// the only contract we have. reflect.DeepEqual handles maps
	// element-wise, so we only need to confirm round-trip.
	t.Run("string_string", func(t *testing.T) {
		m := map[string]string{
			"alpha": "1", "beta": "2", "gamma": "3",
			"delta": "4", "epsilon": "5",
		}
		runValidity(t, m)
	})
	t.Run("string_int", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
		runValidity(t, m)
	})
	t.Run("string_any", func(t *testing.T) {
		// Positive ints round-trip through the any path as uint64
		// (the wire encodes them as positive fixints; decodeAny
		// returns uint64 for those). Use uint64 in the fixture so
		// reflect.DeepEqual matches.
		m := map[string]any{
			"i":  uint64(42),
			"s":  "hello",
			"b":  true,
			"f":  3.14,
			"sl": []any{uint64(1), uint64(2), uint64(3)},
		}
		runValidity(t, m)
	})
	t.Run("nested_map", func(t *testing.T) {
		runValidity(t, map[string]map[string]int{
			"outer1": {"a": 1, "b": 2},
			"outer2": {"x": 9},
		})
	})
}

// --- Edge values per primitive type ----------------------------

func TestValidity_NumericEdgeValues(t *testing.T) {
	type edges struct {
		MinInt8   int8   `qdf:"min_i8"`
		MaxInt8   int8   `qdf:"max_i8"`
		MinInt16  int16  `qdf:"min_i16"`
		MaxInt16  int16  `qdf:"max_i16"`
		MinInt32  int32  `qdf:"min_i32"`
		MaxInt32  int32  `qdf:"max_i32"`
		MinInt64  int64  `qdf:"min_i64"`
		MaxInt64  int64  `qdf:"max_i64"`
		MaxUint8  uint8  `qdf:"max_u8"`
		MaxUint16 uint16 `qdf:"max_u16"`
		MaxUint32 uint32 `qdf:"max_u32"`
		MaxUint64 uint64 `qdf:"max_u64"`
		Zero      int    `qdf:"zero"`
		NegOne    int    `qdf:"neg1"`
		Varint1B  uint64 `qdf:"v1"`
		Varint2B  uint64 `qdf:"v2"`
		Varint3B  uint64 `qdf:"v3"`
	}
	runValidity(t, edges{
		MinInt8: math.MinInt8, MaxInt8: math.MaxInt8,
		MinInt16: math.MinInt16, MaxInt16: math.MaxInt16,
		MinInt32: math.MinInt32, MaxInt32: math.MaxInt32,
		MinInt64: math.MinInt64, MaxInt64: math.MaxInt64,
		MaxUint8: math.MaxUint8, MaxUint16: math.MaxUint16,
		MaxUint32: math.MaxUint32, MaxUint64: math.MaxUint64,
		Zero: 0, NegOne: -1,
		Varint1B: 127, Varint2B: 16383, Varint3B: 2_097_151,
	})
}

func TestValidity_FloatBitExact(t *testing.T) {
	// NaN, ±Inf, ±0 round-trip via bit pattern (not float compare).
	// Marshal then Unmarshal then math.Float64bits comparison —
	// reflect.DeepEqual already does this for float fields via the
	// IEEE 754 bit-pattern comparison, so the suite catches a quiet
	// loss of NaN payload or sign-of-zero.
	type vec struct {
		Nan64   float64 `qdf:"n64"`
		PInf64  float64 `qdf:"pi64"`
		NInf64  float64 `qdf:"ni64"`
		NegZero float64 `qdf:"nz"`
		PosZero float64 `qdf:"pz"`
		Subnorm float64 `qdf:"sn"`
		Max64   float64 `qdf:"mx64"`
		Nan32   float32 `qdf:"n32"`
		PInf32  float32 `qdf:"pi32"`
	}
	in := vec{
		Nan64:   math.NaN(),
		PInf64:  math.Inf(+1),
		NInf64:  math.Inf(-1),
		NegZero: math.Copysign(0, -1),
		PosZero: 0,
		Subnorm: math.SmallestNonzeroFloat64,
		Max64:   math.MaxFloat64,
		Nan32:   float32(math.NaN()),
		PInf32:  float32(math.Inf(+1)),
	}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatal(err)
			}
			var out vec
			if err := Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			// Bit-pattern equality — NaN != NaN under float compare.
			if math.Float64bits(in.Nan64) != math.Float64bits(out.Nan64) {
				t.Fatalf("Nan64 lost bit pattern")
			}
			if math.Float64bits(in.NegZero) != math.Float64bits(out.NegZero) {
				t.Fatalf("NegZero lost sign bit")
			}
			if math.Float64bits(in.PosZero) != math.Float64bits(out.PosZero) {
				t.Fatalf("PosZero changed")
			}
			if in.PInf64 != out.PInf64 || in.NInf64 != out.NInf64 {
				t.Fatalf("Inf round-trip lost")
			}
			if in.Subnorm != out.Subnorm || in.Max64 != out.Max64 {
				t.Fatalf("extremal magnitude lost")
			}
			if math.Float32bits(in.Nan32) != math.Float32bits(out.Nan32) {
				t.Fatalf("Nan32 lost bit pattern")
			}
			if in.PInf32 != out.PInf32 {
				t.Fatalf("PInf32 lost")
			}
		})
	}
}

// --- UTF-8 string safety ---------------------------------------

func TestValidity_UTF8(t *testing.T) {
	// Multi-byte runes across the BMP and supplementary planes plus
	// a deliberately mixed-script line that exercises 1, 2, 3, and
	// 4-byte UTF-8 sequences in the same blob.
	corpus := []string{
		"hello",
		"привет",                 // Cyrillic, 2-byte each
		"こんにちは",                  // Japanese, 3-byte each
		"السلام عليكم",           // Arabic, 2-byte
		"🌍🚀🎉",                    // emoji, 4-byte each
		"​  hidden‮spaces",       // zero-width + LRO
		strings.Repeat("Ω", 200), // 2-byte repeated
	}
	for _, s := range corpus {
		if !utf8.ValidString(s) {
			t.Fatalf("corpus invariant: input %q is not valid UTF-8", s)
		}
	}
	runValidity(t, corpus)
}

func TestValidity_UTF8_RawBytesViaBin(t *testing.T) {
	// []byte path encodes via bin tags; arbitrary bytes (not just
	// valid UTF-8) must round-trip byte-for-byte.
	raw := make([]byte, 0, 512)
	for i := range 256 {
		raw = append(raw, byte(i))
	}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(raw, p.opts)
			if err != nil {
				t.Fatal(err)
			}
			var out []byte
			if err := Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, out) {
				t.Fatalf("raw bytes diverged\n in=%x\nout=%x", raw, out)
			}
		})
	}
}

// --- Time / pointer / embedded types ---------------------------

func TestValidity_TimeNonUTC(t *testing.T) {
	// time.Time encodes as Unix-nanos; the location is dropped on
	// the wire. The round-trip therefore preserves the instant but
	// returns a time in the local zone — compare via Time.Equal
	// (instant equality) rather than reflect.DeepEqual.
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}
	in := time.Date(2026, 5, 26, 12, 34, 56, 789_000_000, loc)
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatal(err)
			}
			var out time.Time
			if err := Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			if !in.Equal(out) {
				t.Fatalf("instant diverged: in=%v out=%v", in, out)
			}
		})
	}
}

func TestValidity_PointerFields(t *testing.T) {
	type leaf struct {
		V int `qdf:"v"`
	}
	type holder struct {
		Direct *leaf   `qdf:"direct"`
		Nil    *leaf   `qdf:"nil"`
		Slice  []*leaf `qdf:"slice"`
	}
	runValidity(t, holder{
		Direct: &leaf{V: 42},
		Nil:    nil,
		Slice:  []*leaf{{V: 1}, {V: 2}, nil, {V: 4}},
	})
}

func TestValidity_EmbeddedStruct(t *testing.T) {
	type base struct {
		ID int `qdf:"id"`
	}
	type composed struct {
		base
		Name string `qdf:"name"`
	}
	runValidity(t, composed{base: base{ID: 7}, Name: "embedded"})
}

// --- Pool reuse across mixed Options ---------------------------

// TestValidity_PoolReuseMixedOpts hammers the shared encPool with a
// sequence of calls that alternate Options. The encoder must apply
// the new opts on each acquire and not leak the previous run's
// intern table / opts / mode into the next caller.
func TestValidity_PoolReuseMixedOpts(t *testing.T) {
	v := struct {
		A int      `qdf:"a"`
		B string   `qdf:"b"`
		C []string `qdf:"c"`
	}{42, "hello", []string{"foo", "bar", "foo"}}

	sequence := []Options{
		OptSpeed,
		OptBalanced,
		OptSpeed,
		OptQPack,
		OptBalanced,
		OptSpeed,
		OptBalanced,
		OptDense | OptShapeIntern,
	}
	for i, opts := range sequence {
		b, err := Marshal(v, opts)
		if err != nil {
			t.Fatalf("[%d] encode: %v", i, err)
		}
		var out struct {
			A int      `qdf:"a"`
			B string   `qdf:"b"`
			C []string `qdf:"c"`
		}
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if !reflect.DeepEqual(v, out) {
			t.Fatalf("[%d] round-trip mismatch under opts=%05b\n in=%+v\nout=%+v",
				i, opts, v, out)
		}
	}
}

// --- Concurrent encode / decode --------------------------------

// TestValidity_ConcurrentEncodeDecode runs many goroutines that
// encode and decode concurrently through the shared pools. The race
// detector flags any non-atomic state. Correctness is verified by
// round-trip equality per call; the test fails if any goroutine's
// decoded value diverges from its input.
func TestValidity_ConcurrentEncodeDecode(t *testing.T) {
	const goroutines = 16
	const iters = 200
	type item struct {
		I int    `qdf:"i"`
		N string `qdf:"n"`
	}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for g := range goroutines {
				g := g
				go func() {
					defer wg.Done()
					for i := range iters {
						in := item{I: g*iters + i, N: fmt.Sprintf("g%d-i%d", g, i)}
						b, err := Marshal(in, p.opts)
						if err != nil {
							t.Errorf("encode: %v", err)
							return
						}
						var out item
						if err := Unmarshal(b, &out); err != nil {
							t.Errorf("decode: %v wire=%x", err, b)
							return
						}
						if in != out {
							t.Errorf("mismatch: %+v vs %+v", in, out)
							return
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}

// --- Wire self-description (cross-opts decode through one Unmarshal) ---

// TestValidity_AnyOptsDecodesIntoAnyType pins the wire-self-description
// contract: every opts-emitted byte stream decodes correctly through
// the single Unmarshal entry point, regardless of which option mix
// the encoder chose. This is the "decoder doesn't have to know"
// guarantee documented in the README.
func TestValidity_AnyOptsDecodesIntoAnyType(t *testing.T) {
	type sample struct {
		IDs    []uint64  `qdf:"ids"`
		Tags   []string  `qdf:"tags"`
		Vec    []float64 `qdf:"vec"`
		Bools  []bool    `qdf:"bools"`
		Bytes  []byte    `qdf:"bytes"`
		Status int       `qdf:"status"`
	}
	in := sample{
		IDs:    []uint64{1_700_000_000, 1_700_000_001, 1_700_000_002, 1_700_000_003},
		Tags:   []string{"prod", "prod", "stage", "prod"},
		Vec:    []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Bools:  []bool{true, false, true, false},
		Bytes:  []byte("hello, world"),
		Status: 200,
	}
	encoded := map[Options][]byte{}
	for _, p := range allEncodeOpts() {
		b, err := Marshal(in, p.opts)
		if err != nil {
			t.Fatalf("%s encode: %v", p.name, err)
		}
		encoded[p.opts] = b
	}
	for opts, b := range encoded {
		var out sample
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("decode opts=%05b: %v wire=%x", opts, err, b)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("opts=%05b mismatch", opts)
		}
	}
}

// --- Encoder.Reset opts hygiene -------------------------------

// TestValidity_EncoderResetClearsOpts confirms Reset() returns the
// encoder to OptSpeed so a manually-driven *Encoder cannot leak its
// previous configuration into a fresh encode.
func TestValidity_EncoderResetClearsOpts(t *testing.T) {
	enc := NewEncoderWith(OptBalanced)
	if !enc.opts.Has(OptBalanced) {
		t.Fatal("encoder did not pick up OptBalanced")
	}
	enc.WriteString("seed")
	enc.Reset()
	if enc.opts != OptSpeed {
		t.Fatalf("Reset did not clear opts: %b", enc.opts)
	}
	if enc.mode != Fast {
		t.Fatalf("Reset did not switch to Fast: %v", enc.mode)
	}
	if enc.qpack {
		t.Fatalf("Reset did not clear qpack")
	}
}

// --- Order preservation in deep nesting -----------------------

// TestValidity_NestedEmbedding verifies the encoding/json-style
// flattening of anonymous embedded structs works recursively (an
// embedded type that itself embeds another).
func TestValidity_NestedEmbedding(t *testing.T) {
	type leaf struct {
		L int `qdf:"l"`
	}
	type mid struct {
		leaf
		M int `qdf:"m"`
	}
	type top struct {
		mid
		T int `qdf:"t"`
	}
	in := top{mid: mid{leaf: leaf{L: 1}, M: 2}, T: 3}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatal(err)
			}
			var out top
			if err := Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			if out != in {
				t.Fatalf("nested-embedding round-trip lost data:\n in=%+v\nout=%+v", in, out)
			}
		})
	}
}

// TestValidity_DecodeMalformed runs a small malformed-input matrix
// through Unmarshal and asserts none of them panic. The decoder is
// the public attack surface for untrusted bytes; the contract is
// "returns an error, never panics".
func TestValidity_DecodeMalformed(t *testing.T) {
	corpus := [][]byte{
		nil,
		{},
		{'Q'},
		{'Q', 'D', 'F'},
		{'Q', 'D', 'F', 0x99, 0x00},       // bad version
		{'X', 'X', 'X', 0x01, 0x00},       // bad magic
		{'Q', 'D', 'F', 0x01, 0x00, 0xFF}, // unknown tag
		{'Q', 'D', 'F', 0x01, 0x00, 0xCE, 0x99, 0x99, 0x99}, // truncated str16
		{'Q', 'D', 'F', 0x01, 0x01, 0xE1, 0x05},             // state-ref to unknown id
	}
	for i, bad := range corpus {
		t.Run(fmt.Sprintf("corpus_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Unmarshal panicked on malformed input: %v", r)
				}
			}()
			var out any
			err := Unmarshal(bad, &out)
			if err == nil {
				// Some short buffers may decode to nil legally; assert
				// at least that we did not crash.
				return
			}
			_ = err
		})
	}
}

// TestValidity_ConcurrentMixedOpts exercises the shared encPool
// under simultaneous goroutines that each use a different Options
// value. Verifies the pool's applyOpts-on-acquire contract holds
// even when multiple opts coexist in flight.
func TestValidity_ConcurrentMixedOpts(t *testing.T) {
	type item struct {
		I int    `qdf:"i"`
		N string `qdf:"n"`
	}
	matrix := []Options{OptSpeed, OptQPack, OptDense, OptBalanced, OptDense | OptShapeIntern}
	const iters = 100
	var wg sync.WaitGroup
	wg.Add(len(matrix))
	for gIdx, opts := range matrix {
		go func() {
			defer wg.Done()
			for i := range iters {
				in := item{I: gIdx*iters + i, N: fmt.Sprintf("g%d-i%d", gIdx, i)}
				b, err := Marshal(in, opts)
				if err != nil {
					t.Errorf("encode opts=%b: %v", opts, err)
					return
				}
				var out item
				if err := Unmarshal(b, &out); err != nil {
					t.Errorf("decode opts=%b: %v", opts, err)
					return
				}
				if in != out {
					t.Errorf("opts=%b mismatch: %+v vs %+v", opts, in, out)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// TestValidity_AppendAccumulates verifies repeated AppendMarshal
// calls into the same buffer accumulate without losing or
// overwriting earlier bytes. The encoder must respect the existing
// dst content and append to it.
func TestValidity_AppendAccumulates(t *testing.T) {
	dst := []byte{0xAA, 0xBB, 0xCC} // sentinel prefix
	pre := len(dst)
	var err error
	for i := range 5 {
		dst, err = AppendMarshal(dst, i, OptSpeed)
		if err != nil {
			t.Fatalf("append [%d]: %v", i, err)
		}
	}
	// Prefix preserved.
	if !bytes.Equal(dst[:pre], []byte{0xAA, 0xBB, 0xCC}) {
		t.Fatalf("prefix corrupted: %x", dst[:pre])
	}
	// Each suffix appended a full envelope (header + value). Total
	// length > pre and the buffer is non-empty.
	if len(dst) <= pre {
		t.Fatalf("nothing appended: len=%d pre=%d", len(dst), pre)
	}
}

// TestValidity_DeepArrayOfArraysOrder builds a depth-3 nested
// integer array with distinct numbers at every position and confirms
// every (i, j, k) tuple round-trips to its original value. Catches
// any off-by-one in the slice fast-path / reflect path that could
// shuffle elements.
func TestValidity_DeepArrayOfArraysOrder(t *testing.T) {
	const D = 6
	in := make([][][]int, D)
	want := 0
	for i := range in {
		in[i] = make([][]int, D)
		for j := range in[i] {
			in[i][j] = make([]int, D)
			for k := range in[i][j] {
				in[i][j][k] = want
				want++
			}
		}
	}
	for _, p := range allEncodeOpts() {
		t.Run(p.name, func(t *testing.T) {
			b, err := Marshal(in, p.opts)
			if err != nil {
				t.Fatal(err)
			}
			var out [][][]int
			if err := Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			seen := 0
			for i := range out {
				for j := range out[i] {
					for k := range out[i][j] {
						if out[i][j][k] != seen {
							t.Fatalf("[%d][%d][%d]=%d want %d", i, j, k, out[i][j][k], seen)
						}
						seen++
					}
				}
			}
			if seen != D*D*D {
				t.Fatalf("decoded %d elements, expected %d", seen, D*D*D)
			}
		})
	}
}
