package qdf

import (
	"bytes"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// Completeness tests: encode something, decode it back, prove bit-for-bit
// equality. Run across the three top-level entry points (Marshal,
// MarshalDense, MarshalQPack), every QPack-eligible type, every edge
// case the codecs care about. If any field comes back wrong, fail loud.

type bigPayload struct {
	Name string `qdf:"name"`

	Empty []uint64  `qdf:"empty"`
	OneU  []uint64  `qdf:"oneu"`
	OneI  []int64   `qdf:"onei"`
	OneB  []bool    `qdf:"oneb"`
	OneF  []float64 `qdf:"onef"`

	Bools []bool    `qdf:"bools"`
	Bytes []byte    `qdf:"bytes"`
	U8    []uint8   `qdf:"u8"`
	I8    []int8    `qdf:"i8"`
	U16   []uint16  `qdf:"u16"`
	I16   []int16   `qdf:"i16"`
	U32   []uint32  `qdf:"u32"`
	I32   []int32   `qdf:"i32"`
	U64   []uint64  `qdf:"u64"`
	I64   []int64   `qdf:"i64"`
	Ints  []int     `qdf:"ints"`
	F32   []float32 `qdf:"f32"`
	F64   []float64 `qdf:"f64"`
	Strs  []string  `qdf:"strs"`

	MaxedU64    []uint64  `qdf:"maxu64"`
	MinIntsI64  []int64   `qdf:"miniii"`
	NaNAndInf32 []float32 `qdf:"nan32"`
	NaNAndInf64 []float64 `qdf:"nan64"`
	NegZero     []float64 `qdf:"negzero"`

	Mono   []uint64 `qdf:"mono"`
	Const  []uint64 `qdf:"const"`
	BigRng []int64  `qdf:"bigrng"`

	M map[string]int `qdf:"m"`

	// Nested
	Items []item `qdf:"items"`
}

type item struct {
	ID    uint64    `qdf:"id"`
	Tags  []string  `qdf:"tags"`
	Vec   []float64 `qdf:"vec"`
	Flags []bool    `qdf:"flags"`
}

func makeBigPayload(t *testing.T) bigPayload {
	t.Helper()
	rng := rand.New(rand.NewSource(1234))

	mkU := func(n int) []uint64 {
		s := make([]uint64, n)
		for i := range s {
			s[i] = rng.Uint64()
		}
		return s
	}
	mkI := func(n int) []int64 {
		s := make([]int64, n)
		for i := range s {
			s[i] = int64(rng.Uint64())
		}
		return s
	}

	return bigPayload{
		Name:  "completeness-test",
		Empty: []uint64{},
		OneU:  []uint64{42},
		OneI:  []int64{-7},
		OneB:  []bool{true},
		OneF:  []float64{3.14},
		Bools: []bool{true, false, true, true, false, false, true, false, true},
		Bytes: []byte{0x00, 0xFF, 0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56},
		U8:    []uint8{0, 1, 127, 128, 255},
		I8:    []int8{-128, -1, 0, 1, 127},
		U16:   []uint16{0, 1, 65535},
		I16:   []int16{-32768, 0, 32767},
		U32:   []uint32{0, 1, math.MaxUint32, 12345},
		I32:   []int32{math.MinInt32, -1, 0, 1, math.MaxInt32},
		U64:   mkU(33),
		I64:   mkI(33),
		Ints:  []int{-1 << 30, -1, 0, 1, 1 << 30},
		F32:   []float32{0, -0, 1.5, -2.25, 1e-30, 1e30},
		F64:   []float64{0, -0, 1.5, -2.25, 1e-300, 1e300},
		Strs:  []string{"hello", "world", "hello", "", "long-string-that-might-be-interned"},

		MaxedU64:    []uint64{math.MaxUint64, math.MaxUint64 - 1, 0, 1},
		MinIntsI64:  []int64{math.MinInt64, math.MinInt64 + 1, 0, math.MaxInt64},
		NaNAndInf32: []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 0},
		NaNAndInf64: []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0},
		NegZero:     []float64{0, -0, 0, -0},

		Mono:   makeMonotonicU64(64, 1_700_000_000, 0),
		Const:  []uint64{42, 42, 42, 42, 42, 42, 42, 42},
		BigRng: []int64{math.MinInt64, math.MaxInt64, 0, -1, 1},

		M: map[string]int{"a": 1, "b": 2, "c": 3},

		Items: []item{
			{ID: 100, Tags: []string{"x", "y"}, Vec: []float64{1, 2, 3}, Flags: []bool{true, false}},
			{ID: 200, Tags: []string{}, Vec: nil, Flags: []bool{true, true, true}},
			{ID: 300, Tags: []string{"only"}, Vec: []float64{0.5}, Flags: nil},
		},
	}
}

// deepEqualFloatAware compares two values for equality, allowing NaN
// patterns to match NaN. Plain reflect.DeepEqual fails on NaN.
func deepEqualFloatAware(t *testing.T, a, b any, path string) bool {
	t.Helper()
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if va.Kind() != vb.Kind() {
		t.Errorf("%s: kind %s vs %s", path, va.Kind(), vb.Kind())
		return false
	}
	switch va.Kind() {
	case reflect.Float32:
		x, y := va.Interface().(float32), vb.Interface().(float32)
		if math.IsNaN(float64(x)) && math.IsNaN(float64(y)) {
			return true
		}
		if math.Float32bits(x) != math.Float32bits(y) {
			t.Errorf("%s: %v vs %v", path, x, y)
			return false
		}
		return true
	case reflect.Float64:
		x, y := va.Interface().(float64), vb.Interface().(float64)
		if math.IsNaN(x) && math.IsNaN(y) {
			return true
		}
		if math.Float64bits(x) != math.Float64bits(y) {
			t.Errorf("%s: %v (%x) vs %v (%x)", path, x, math.Float64bits(x), y, math.Float64bits(y))
			return false
		}
		return true
	case reflect.Slice, reflect.Array:
		if va.Len() != vb.Len() {
			t.Errorf("%s: len %d vs %d", path, va.Len(), vb.Len())
			return false
		}
		ok := true
		for i := range va.Len() {
			if !deepEqualFloatAware(t, va.Index(i).Interface(), vb.Index(i).Interface(), pathIdx(path, i)) {
				ok = false
			}
		}
		return ok
	case reflect.Struct:
		ok := true
		for i := range va.NumField() {
			if !va.Type().Field(i).IsExported() {
				continue
			}
			if !deepEqualFloatAware(t, va.Field(i).Interface(), vb.Field(i).Interface(),
				path+"."+va.Type().Field(i).Name) {
				ok = false
			}
		}
		return ok
	case reflect.Map:
		if va.Len() != vb.Len() {
			t.Errorf("%s: map len %d vs %d", path, va.Len(), vb.Len())
			return false
		}
		ok := true
		for _, k := range va.MapKeys() {
			vbv := vb.MapIndex(k)
			if !vbv.IsValid() {
				t.Errorf("%s: missing key %v", path, k.Interface())
				ok = false
				continue
			}
			if !deepEqualFloatAware(t, va.MapIndex(k).Interface(), vbv.Interface(),
				path+"["+sprintKey(k)+"]") {
				ok = false
			}
		}
		return ok
	case reflect.String:
		if va.String() != vb.String() {
			t.Errorf("%s: %q vs %q", path, va.String(), vb.String())
			return false
		}
		return true
	default:
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s: %v vs %v", path, a, b)
			return false
		}
		return true
	}
}

func pathIdx(p string, i int) string {
	return p + "[" + itoaPlain(i) + "]"
}

func sprintKey(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	}
	return v.Type().String()
}

func itoaPlain(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestCompleteness_AllModes(t *testing.T) {
	in := makeBigPayload(t)
	modes := []struct {
		name    string
		marshal func(any) ([]byte, error)
	}{
		{"Marshal", Marshal},
		{"MarshalQPack", MarshalQPack},
		{"MarshalDense", MarshalDense},
	}
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			buf, err := m.marshal(in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out bigPayload
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			deepEqualFloatAware(t, in, out, m.name)
		})
	}
}

func TestCompleteness_CrossModeDecode(t *testing.T) {
	in := makeBigPayload(t)
	// Encode in QPack/Dense, decode by the same Unmarshal (only one decoder
	// exists). Then encode in legacy Marshal and verify decoder still
	// accepts it (forward-compat in the other direction).
	for _, enc := range []func(any) ([]byte, error){Marshal, MarshalQPack, MarshalDense} {
		buf, err := enc(in)
		if err != nil {
			t.Fatal(err)
		}
		var out bigPayload
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("decode %d-byte buf: %v", len(buf), err)
		}
		deepEqualFloatAware(t, in, out, "cross")
	}
}

func TestCompleteness_StreamingDense(t *testing.T) {
	items := []bigPayload{
		makeBigPayload(t),
		makeBigPayload(t),
		makeBigPayload(t),
	}
	items[1].Name = "msg2"
	items[2].Name = "msg3"

	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(&w)
	defer dec.Close()
	for i, want := range items {
		var got bigPayload
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("decode msg %d: %v", i, err)
		}
		deepEqualFloatAware(t, want, got, "stream["+itoaPlain(i)+"]")
	}
}

func TestCompleteness_FuzzRandomStructsQPack(t *testing.T) {
	// Random data, never the same shape twice. Hammer all three encoders.
	rng := rand.New(rand.NewSource(99))
	for trial := range 50 {
		type rec struct {
			A []uint64  `qdf:"a"`
			B []int64   `qdf:"b"`
			C []float64 `qdf:"c"`
			D []bool    `qdf:"d"`
			E string    `qdf:"e"`
		}
		n := rng.Intn(256)
		in := rec{
			A: make([]uint64, n),
			B: make([]int64, n),
			C: make([]float64, n),
			D: make([]bool, n),
			E: "trial-" + itoaPlain(trial),
		}
		for i := range n {
			in.A[i] = rng.Uint64() >> uint(rng.Intn(60))
			in.B[i] = int64(rng.Uint64()) >> uint(rng.Intn(60))
			in.C[i] = rng.Float64() - 0.5
			in.D[i] = rng.Intn(2) == 0
		}
		for _, enc := range []func(any) ([]byte, error){Marshal, MarshalQPack, MarshalDense} {
			buf, err := enc(in)
			if err != nil {
				t.Fatalf("trial %d marshal: %v", trial, err)
			}
			var out rec
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("trial %d unmarshal (%d bytes): %v", trial, len(buf), err)
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("trial %d mismatch with encoder", trial)
			}
		}
	}
}
