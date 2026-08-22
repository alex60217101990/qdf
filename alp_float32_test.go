package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

type alp32row struct {
	V []float32 `qdf:"v"`
}

// TestALPFloat32Fires: a quantized float32 column compresses via ALP under
// OptCompression (well below raw 4 bytes/val) and round-trips bit-exactly.
func TestALPFloat32Fires(t *testing.T) {
	n := 2000
	v := make([]float32, n)
	for i := range v {
		// float32-nearest-to-decimal (e.g. a sensor/price value stored as float32):
		// v = round-to-f32(k/10). round(v*10) == k and f32(k/10) == v, so it packs
		// as an integer mantissa with no exceptions. (Computing it as f32*f32, e.g.
		// float32(i)*0.1, would NOT match the f64 reconstruction and falls to the
		// exception path — that is float-arithmetic noise, not clean decimal data.)
		v[i] = float32(float64(2050+i) / 10)
	}
	in := alp32row{V: v}
	// OptBalanced|OptGorillaFloat enables ALP but NOT the whole-body rANS pass, so
	// the wire reflects the column codec directly: ALP must fire (Gorilla declines
	// on decimal data) and the body collapses to ~ceil(log2 range) bits/value.
	b, err := Marshal(in, OptBalanced|OptGorillaFloat)
	if err != nil {
		t.Fatal(err)
	}
	raw := 2 + uvarintLen(uint64(n)) + n*4
	hasALP := false
	for i := 0; i+1 < len(b); i++ {
		if b[i] == tagPackALP && b[i+1] == qpackKindFloat32 {
			hasALP = true
			break
		}
	}
	if !hasALP {
		t.Fatalf("ALP float32 did not fire (no tagPackALP/kindFloat32); wire=%d raw=%d", len(b), raw)
	}
	if len(b)*2 >= raw { // ALP ~11 bits/val vs raw 32 → well under half
		t.Fatalf("ALP float32 wire %d not << raw %d", len(b), raw)
	}
	var out alp32row
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("round-trip mismatch")
	}
	t.Logf("wire=%d raw=%d (%.1f bits/val)", len(b), raw, float64(len(b)*8)/float64(n))
}

// TestALPFloat32Exceptions: NaN / ±Inf / -0.0 / irrational values round-trip
// bit-exactly via the exception list.
func TestALPFloat32Exceptions(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for iter := range 200 {
		n := 16 + r.Intn(300)
		v := make([]float32, n)
		for i := range v {
			switch r.Intn(8) {
			case 0:
				v[i] = float32(math.NaN())
			case 1:
				v[i] = float32(math.Inf(1))
			case 2:
				v[i] = float32(math.Inf(-1))
			case 3:
				v[i] = float32(math.Copysign(0, -1)) // -0.0
			case 4:
				v[i] = math.Float32frombits(r.Uint32()) // arbitrary bits
			default:
				v[i] = float32(r.Intn(100000)) * 0.01 // decimal
			}
		}
		in := alp32row{V: v}
		for _, opt := range []Options{OptBalanced, OptCompression, OptBalanced | OptGorillaFloat} {
			b, err := Marshal(in, opt)
			if err != nil {
				t.Fatalf("iter %d opt %d: %v", iter, opt, err)
			}
			var out alp32row
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("iter %d opt %d: %v", iter, opt, err)
			}
			for i := range v {
				if math.Float32bits(out.V[i]) != math.Float32bits(v[i]) {
					t.Fatalf("iter %d opt %d [%d]: bits %x != %x", iter, opt, i, math.Float32bits(out.V[i]), math.Float32bits(v[i]))
				}
			}
		}
	}
}
