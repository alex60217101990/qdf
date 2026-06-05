package qdf

import (
	"bytes"
	"fmt"
	"testing"
)

func TestOptMapShape_Bit(t *testing.T) {
	if OptMapShape == 0 {
		t.Fatal("OptMapShape must be a nonzero bit")
	}
	for _, o := range []Options{OptDense, OptQPack, OptShapeIntern, OptPairPred, OptMTF, OptGorillaFloat, OptRANS, OptColumnIndex, OptFSST} {
		if OptMapShape == o {
			t.Fatalf("OptMapShape collides with %v", o)
		}
	}
	if OptBalanced.Has(OptMapShape) {
		t.Fatal("OptMapShape must not be in OptBalanced in v1")
	}
	if OptCompression.Has(OptMapShape) {
		t.Fatal("OptMapShape must not be in OptCompression in v1")
	}
}

func TestEncStateMapShape_Registry(t *testing.T) {
	st := newEncState()
	if _, _, ok := st.mapShapeFindKeys(0x1234, 2); ok {
		t.Fatal("empty registry must miss")
	}
	keys := []string{"client", "version"} // canonical (sorted)
	st.mapShapeRegister(0x1234, 2, keys, 7)
	id, got, ok := st.mapShapeFindKeys(0x1234, 2)
	if !ok || id != 7 {
		t.Fatalf("find = (%d,%v), want (7,true)", id, ok)
	}
	if len(got) != 2 || got[0] != "client" || got[1] != "version" {
		t.Fatalf("findKeys keys = %v", got)
	}
	// Length disambiguates a setHash collision across different set sizes.
	if _, _, ok := st.mapShapeFindKeys(0x1234, 3); ok {
		t.Fatal("len mismatch must miss")
	}
	// Register mutates nothing the caller owns: caller slice change must not leak.
	keys[0] = "MUTATED"
	if _, got, ok := st.mapShapeFindKeys(0x1234, 2); !ok || got[0] != "client" {
		t.Fatal("mapShapeRegister must clone keys")
	}
	st.reset()
	if _, _, ok := st.mapShapeFindKeys(0x1234, 2); ok {
		t.Fatal("reset must clear the registry")
	}
}

func roundTripMapShape[V comparable](t *testing.T, in map[string]V) {
	t.Helper()
	type wrap struct {
		M map[string]V `qdf:"m"`
	}
	b, err := Marshal(wrap{M: in}, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out wrap
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.M) != len(in) {
		t.Fatalf("len = %d, want %d", len(out.M), len(in))
	}
	for k, want := range in {
		if got, ok := out.M[k]; !ok || got != want {
			t.Fatalf("key %q = (%v,%v), want %v", k, got, ok, want)
		}
	}
}

func TestMapShape_RoundTrip_Single(t *testing.T) {
	roundTripMapShape(t, map[string]string{"version": "v3.42.1", "client": "go-client/1.20"})
	roundTripMapShape(t, map[string]int{"a": 1, "b": 2, "c": 3})
	roundTripMapShape(t, map[string]string{})
	roundTripMapShape(t, map[string]string{"only": "one"})
}

func TestMapShape_RoundTrip_SliceRecurring(t *testing.T) {
	type row struct {
		ID   int               `qdf:"id"`
		Tags map[string]string `qdf:"tags"`
	}
	in := make([]row, 50)
	for i := range in {
		in[i] = row{ID: i, Tags: map[string]string{"version": "v3.42.1", "client": "go-client/1.20"}}
	}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out []row
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("len %d != %d", len(out), len(in))
	}
	for i := range in {
		if out[i].ID != in[i].ID || out[i].Tags["version"] != "v3.42.1" || out[i].Tags["client"] != "go-client/1.20" {
			t.Fatalf("row %d mismatch: %+v", i, out[i])
		}
	}
}

func TestMapShape_RoundTrip_VaryingKeysets(t *testing.T) {
	type row struct {
		Tags map[string]mapShapeVal `qdf:"tags"`
	}
	in := []row{
		{Tags: map[string]mapShapeVal{"a": {1, 1}}},
		{Tags: map[string]mapShapeVal{"a": {1, 1}, "b": {2, 2}}},
		{Tags: map[string]mapShapeVal{"c": {3, 3}}},
		{Tags: map[string]mapShapeVal{"a": {1, 1}, "b": {2, 2}}},
		{Tags: nil},
	}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out []row
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if len(out[i].Tags) != len(in[i].Tags) {
			t.Fatalf("row %d len %d != %d", i, len(out[i].Tags), len(in[i].Tags))
		}
		for k, vv := range in[i].Tags {
			if out[i].Tags[k] != vv {
				t.Fatalf("row %d key %q mismatch", i, k)
			}
		}
	}
}

// mapShapeVal is a struct value type with NO generated map fast path, so a
// map[string]mapShapeVal exercises the reflect encodeMap -> encodeStringMapShaped
// path (the fast paths in maps_fast_generated.go cover only scalar/string
// values and are wired in a later phase).
type mapShapeVal struct {
	X int `qdf:"x"`
	Y int `qdf:"y"`
}

func TestMapShape_Deterministic(t *testing.T) {
	type wrap struct {
		M map[string]mapShapeVal `qdf:"m"`
	}
	v := wrap{M: map[string]mapShapeVal{"z": {1, 1}, "a": {2, 2}, "m": {3, 3}}}
	b1, _ := Marshal(v, OptBalanced|OptMapShape)
	b2, _ := Marshal(v, OptBalanced|OptMapShape)
	if !bytes.Equal(b1, b2) {
		t.Fatal("encode not deterministic under OptMapShape")
	}
}

// TestMapShape_ReflectShapeFires proves the shape path is actually engaged on
// the reflect map encoder: a slice of rows with a recurring key-set must encode
// strictly smaller with OptMapShape than without (keys emitted once), and still
// round-trip.
func TestMapShape_ReflectShapeFires(t *testing.T) {
	type row struct {
		Tags map[string]mapShapeVal `qdf:"tags"`
	}
	in := make([]row, 100)
	for i := range in {
		in[i] = row{Tags: map[string]mapShapeVal{
			"alpha": {i, i + 1}, "bravo": {i + 2, i + 3}, "charlie": {i + 4, i + 5},
		}}
	}
	plain, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	shaped, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	if len(shaped) >= len(plain) {
		t.Fatalf("shape did not engage: shaped=%d plain=%d", len(shaped), len(plain))
	}
	var out []row
	if err := Unmarshal(shaped, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("len %d != %d", len(out), len(in))
	}
	for i := range in {
		for k, vv := range in[i].Tags {
			if out[i].Tags[k] != vv {
				t.Fatalf("row %d key %q = %+v want %+v", i, k, out[i].Tags[k], vv)
			}
		}
	}
}

// TestMapShape_CrossMode verifies that flag-off and flag-on encodings both
// decode to the same value. (Byte-stability of the flag-off path is NOT
// asserted: plain map encoding ranges the Go runtime map in non-deterministic
// order — a pre-existing property. The OptMapShape path, by contrast, is
// deterministic; see TestMapShape_Deterministic.)
func TestMapShape_CrossMode(t *testing.T) {
	type row struct {
		ID   int               `qdf:"id"`
		Tags map[string]string `qdf:"tags"`
	}
	in := []row{{ID: 1, Tags: map[string]string{"a": "1", "b": "2"}}}
	off, _ := Marshal(in, OptBalanced)
	onb, _ := Marshal(in, OptBalanced|OptMapShape)
	var a, b []row
	if err := Unmarshal(off, &a); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(onb, &b); err != nil {
		t.Fatal(err)
	}
	if a[0].Tags["a"] != b[0].Tags["a"] || a[0].Tags["b"] != b[0].Tags["b"] || a[0].ID != b[0].ID {
		t.Fatal("cross-mode decode mismatch")
	}
}

func TestMapShape_CollisionStress(t *testing.T) {
	type row struct {
		Tags map[string]mapShapeVal `qdf:"tags"`
	}
	in := make([]row, 200)
	for i := range in {
		in[i] = row{Tags: map[string]mapShapeVal{
			fmt.Sprintf("k%d", i):   {1, 1},
			fmt.Sprintf("k%d", i+1): {2, 2},
		}}
	}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out []row
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	for i := range in {
		for k, vv := range in[i].Tags {
			if out[i].Tags[k] != vv {
				t.Fatalf("row %d key %q corrupted", i, k)
			}
		}
	}
}

func benchMapShapeCorpus(n int) []struct {
	ID   int               `qdf:"id"`
	Svc  string            `qdf:"svc"`
	Tags map[string]string `qdf:"tags"`
} {
	type row struct {
		ID   int               `qdf:"id"`
		Svc  string            `qdf:"svc"`
		Tags map[string]string `qdf:"tags"`
	}
	_ = row{}
	out := make([]struct {
		ID   int               `qdf:"id"`
		Svc  string            `qdf:"svc"`
		Tags map[string]string `qdf:"tags"`
	}, n)
	svcs := []string{"ingest", "auth", "billing", "api-gateway"}
	for i := range out {
		out[i].ID = i
		out[i].Svc = svcs[i%len(svcs)]
		out[i].Tags = map[string]string{"version": "v3.42.1", "client": "go-client/1.20"}
	}
	return out
}

func BenchmarkMapShape_Telemetry(b *testing.B) {
	in := benchMapShapeCorpus(1000)
	b.Run("off", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in, OptBalanced)
		}
	})
	b.Run("on", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = Marshal(in, OptBalanced|OptMapShape)
		}
	})
}

func TestMapShape_Malformed(t *testing.T) {
	type wrap struct {
		M map[string]string `qdf:"m"`
	}
	in := wrap{M: map[string]string{"a": "1", "b": "2", "c": "3"}}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	// Every truncation must error, never panic.
	for cut := range b {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncated input (cut=%d): %v", cut, r)
				}
			}()
			var out wrap
			_ = Unmarshal(b[:cut], &out)
		}()
	}
}

// TestMapShape_Nested exercises mapHolderCache re-entrancy: a reflect map whose
// value is itself a reflect map (both take the generic path) must round-trip —
// the inner acquire sees the cache busy and uses a local holder pair.
func TestMapShape_Nested(t *testing.T) {
	type wrap struct {
		M map[string]map[string]mapShapeVal `qdf:"m"`
	}
	in := wrap{M: map[string]map[string]mapShapeVal{
		"outer1": {"a": {1, 2}, "b": {3, 4}},
		"outer2": {"a": {5, 6}, "b": {7, 8}},
	}}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out wrap
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.M) != len(in.M) {
		t.Fatalf("outer len %d != %d", len(out.M), len(in.M))
	}
	for ok, im := range in.M {
		om := out.M[ok]
		if len(om) != len(im) {
			t.Fatalf("inner %q len %d != %d", ok, len(om), len(im))
		}
		for ik, iv := range im {
			if om[ik] != iv {
				t.Fatalf("[%q][%q] = %+v want %+v", ok, ik, om[ik], iv)
			}
		}
	}
}

// TestMapShape_PointerValues: map[string]*struct value type through the reflect
// path (nil + non-nil pointers) round-trips under OptMapShape.
func TestMapShape_PointerValues(t *testing.T) {
	type row struct {
		M map[string]*mapShapeVal `qdf:"m"`
	}
	in := []row{
		{M: map[string]*mapShapeVal{"x": {1, 1}, "y": {2, 2}}},
		{M: map[string]*mapShapeVal{"x": {3, 3}, "y": nil}},
		{M: map[string]*mapShapeVal{"x": {4, 4}, "y": {5, 5}}},
	}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out []row
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	for i := range in {
		for k, want := range in[i].M {
			got := out[i].M[k]
			switch {
			case want == nil && got != nil:
				t.Fatalf("row %d key %q: want nil, got %+v", i, k, got)
			case want != nil && (got == nil || *got != *want):
				t.Fatalf("row %d key %q mismatch", i, k)
			}
		}
	}
}

// TestMapShape_RecursiveType: a self-recursive map type (type T map[string]T)
// must round-trip — the directly-recursive same-type case stresses the cache
// busy-guard hardest.
func TestMapShape_RecursiveType(t *testing.T) {
	type rec map[string]any
	in := rec{"a": rec{"b": rec{"c": "leaf"}}, "x": "y"}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out rec
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["x"] != "y" {
		t.Fatalf("x=%v", out["x"])
	}
}

// TestMapShape_TopLevelMap: a map encoded as the ROOT value (not a struct
// field) must still emit the stream header before tagMapShape. Regression for
// a "bad magic" decode failure when the shape path bypassed WriteMapHeader.
func TestMapShape_TopLevelMap(t *testing.T) {
	cases := []any{
		map[string]int{"a": 1, "b": 2, "c": 3},
		map[string]string{"x": "1", "y": "2"},
		map[string]mapShapeVal{"p": {1, 2}, "q": {3, 4}}, // reflect path
	}
	for _, in := range cases {
		b, err := Marshal(in, OptBalanced|OptMapShape)
		if err != nil {
			t.Fatalf("marshal %T: %v", in, err)
		}
		switch v := in.(type) {
		case map[string]int:
			var out map[string]int
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("unmarshal %T: %v", in, err)
			}
			for k, w := range v {
				if out[k] != w {
					t.Fatalf("%T[%q]=%d want %d", in, k, out[k], w)
				}
			}
		case map[string]string:
			var out map[string]string
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("unmarshal %T: %v", in, err)
			}
			for k, w := range v {
				if out[k] != w {
					t.Fatalf("mismatch %q", k)
				}
			}
		case map[string]mapShapeVal:
			var out map[string]mapShapeVal
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("unmarshal %T: %v", in, err)
			}
			for k, w := range v {
				if out[k] != w {
					t.Fatalf("mismatch %q", k)
				}
			}
		}
	}
}
