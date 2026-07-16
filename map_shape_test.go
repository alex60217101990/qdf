package qdf

import (
	"bytes"
	"fmt"
	"testing"
)

// TestDecodeMap_SliceValuesNotAliased pins that a reflect-decoded map whose
// value type is a slice NOT in the fast-path set ([]int8 here) gives each entry
// its own backing array. The decode value holder is reused across entries, and
// reuseOrMakeSlice keeps a cap>=n backing, so without a per-entry reset every
// map value aliased the last slice decoded — silent corruption.
func TestDecodeMap_SliceValuesNotAliased(t *testing.T) {
	in := map[int][]int8{
		1: {1, 2, 3},
		2: {4, 5, 6},
		3: {7, 8, 9},
		4: {10},
		5: {11, 12, 13, 14, 15},
	}
	for _, opts := range []Options{OptSpeed, OptBalanced} {
		b, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("opts=%v marshal: %v", opts, err)
		}
		var out map[int][]int8
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("opts=%v unmarshal: %v", opts, err)
		}
		if !equalMapIntSlice8(in, out) {
			t.Fatalf("opts=%v: map slice values aliased/garbled:\n in =%v\n out=%v", opts, in, out)
		}
	}
}

// TestDecodeMap_ShapeSliceValuesNotAliased is the same hazard through the
// tagMapShape (OptMapShape) decode branch, which uses a pooled value holder.
func TestDecodeMap_ShapeSliceValuesNotAliased(t *testing.T) {
	in := map[string][]int8{"alpha": {1, 2}, "beta": {3, 4, 5}, "gamma": {6}}
	b, err := Marshal(in, OptBalanced|OptMapShape)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string][]int8
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("len %d != %d", len(out), len(in))
	}
	for k, av := range in {
		bv := out[k]
		if len(av) != len(bv) {
			t.Fatalf("key %q len %d != %d", k, len(bv), len(av))
		}
		for i := range av {
			if av[i] != bv[i] {
				t.Fatalf("key %q [%d] = %d, want %d (aliased?)", k, i, bv[i], av[i])
			}
		}
	}
}

func equalMapIntSlice8(a, b map[int][]int8) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
	}
	return true
}

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
	// The production encoders scan st.mapShapes inline (verifying keys); mirror
	// that (setHash, n) lookup here to exercise the registry storage/reset.
	find := func(setHash uint64, n int) (uint32, []string, bool) {
		for i := range st.mapShapes {
			if s := &st.mapShapes[i]; s.setHash == setHash && s.n == n {
				return s.id, s.keys, true
			}
		}
		return 0, nil, false
	}
	if _, _, ok := find(0x1234, 2); ok {
		t.Fatal("empty registry must miss")
	}
	keys := []string{"client", "version"} // canonical (sorted)
	st.mapShapeRegister(0x1234, 2, keys, 7)
	id, got, ok := find(0x1234, 2)
	if !ok || id != 7 {
		t.Fatalf("find = (%d,%v), want (7,true)", id, ok)
	}
	if len(got) != 2 || got[0] != "client" || got[1] != "version" {
		t.Fatalf("find keys = %v", got)
	}
	// Length disambiguates a setHash collision across different set sizes.
	if _, _, ok := find(0x1234, 3); ok {
		t.Fatal("len mismatch must miss")
	}
	// mapShapeRegister takes ownership of the passed slice (no defensive clone):
	// the binding stores the exact slice the caller handed over. Production
	// callers always pass a freshly-made per-declare slice and never reuse it.
	if &got[0] != &keys[0] {
		t.Fatal("mapShapeRegister must take ownership of the passed slice (no clone)")
	}
	st.reset()
	if _, _, ok := find(0x1234, 2); ok {
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

// BenchmarkSkipMapShape measures Skip() on tagMapShape declarations where the
// skipped field's key-set overlaps keys already interned by the non-skip decode
// path.  The fix replaces string(kb) with d.keyCache.Make(kb) in
// decoder_skip.go so overlapping keys are cache hits (zero alloc) instead of
// fresh string copies (one alloc each).
//
// Setup: a sender writes four map-shaped fields; the receiver knows only two of
// them.  Six keys appear in both the known and unknown shapes, so they are
// already interned when Skip() reaches the unknown shape declarations.
func BenchmarkSkipMapShape(b *testing.B) {
	type sender struct {
		Known1 map[string]string `qdf:"k1"` // version, env, region, svc
		Known2 map[string]string `qdf:"k2"` // version, dc, host, env
		Skip1  map[string]string `qdf:"s1"` // version, env, dc, svc  (3 shared with k1/k2)
		Skip2  map[string]string `qdf:"s2"` // region, host, cluster, ns  (2 shared with k1/k2)
	}
	type receiver struct {
		Known1 map[string]string `qdf:"k1"`
		Known2 map[string]string `qdf:"k2"`
	}
	src := sender{
		Known1: map[string]string{"version": "1.0", "env": "prod", "region": "us-east-1", "svc": "api"},
		Known2: map[string]string{"version": "1.0", "dc": "dc1", "host": "h1", "env": "prod"},
		Skip1:  map[string]string{"version": "1.0", "env": "prod", "dc": "dc1", "svc": "api"},
		Skip2:  map[string]string{"region": "us-east-1", "host": "h1", "cluster": "k8s", "ns": "prod"},
	}
	data, err := Marshal(src, OptBalanced|OptMapShape)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		var dst receiver
		if err := Unmarshal(data, &dst); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
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
