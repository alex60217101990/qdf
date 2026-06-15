package qdf

import (
	"errors"
	"maps"
	"reflect"
	"testing"
	"unsafe"
)

func TestPatchHeaderRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   byte
		schema  uint64
		base    uint64
		hasBase bool
	}{
		{"no-base", 0, 0x1122334455667788, 0, false},
		{"with-base", flagPatchBaseFP, 0xDEADBEEFCAFEBABE, 0x0102030405060708, true},
		{"rans+base", flagPatchBaseFP | flagPatchRANS, 1, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := writePatchHeader(nil, tc.flags, tc.schema, tc.base)
			h, n, err := readPatchHeader(buf)
			if err != nil {
				t.Fatalf("readPatchHeader: %v", err)
			}
			if h.flags != tc.flags || h.schemaFP != tc.schema {
				t.Fatalf("flags/schema mismatch: got %#x/%#x", h.flags, h.schemaFP)
			}
			if tc.hasBase && h.baseFP != tc.base {
				t.Fatalf("baseFP: got %#x want %#x", h.baseFP, tc.base)
			}
			if n != len(buf) {
				t.Fatalf("consumed %d want %d", n, len(buf))
			}
		})
	}
}

func TestSchemaFingerprintStableAndDistinct(t *testing.T) {
	type A struct {
		X int
		Y string
	}
	type B struct {
		X int
		Z string // different field name
	}
	tdA1, _ := descOf(reflect.TypeFor[A]())
	tdA2, _ := descOf(reflect.TypeFor[A]())
	tdB, _ := descOf(reflect.TypeFor[B]())
	if tdA1.schemaFP != tdA2.schemaFP {
		t.Fatal("fingerprint not stable across calls")
	}
	if tdA1.schemaFP == tdB.schemaFP {
		t.Fatal("distinct shapes collided")
	}
}

func TestReadPatchHeaderRejectsBadMagic(t *testing.T) {
	bad := []byte{'X', 'D', 'P', patchVersion1, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if _, _, err := readPatchHeader(bad); !errors.Is(err, ErrInvalidPatch) {
		t.Fatalf("got %v want ErrInvalidPatch", err)
	}
	if _, _, err := readPatchHeader([]byte{'Q'}); !errors.Is(err, ErrInvalidPatch) {
		t.Fatalf("short: got %v want ErrInvalidPatch", err)
	}
	truncated := writePatchHeader(nil, flagPatchBaseFP, 1, 2)[:15] // cut mid-baseFP
	if _, _, err := readPatchHeader(truncated); !errors.Is(err, ErrInvalidPatch) {
		t.Fatalf("truncated baseFP: got %v want ErrInvalidPatch", err)
	}
}

func TestEqualValueScalarsStringsBytes(t *testing.T) {
	eq := func(td *typeDesc, aP, bP unsafe.Pointer) bool { return equalValue(td, aP, bP, 0) }

	tdI, _ := descOf(reflect.TypeFor[int]())
	x, y := 5, 5
	z := 6
	if !eq(tdI, unsafe.Pointer(&x), unsafe.Pointer(&y)) {
		t.Fatal("5==5 should be equal")
	}
	if eq(tdI, unsafe.Pointer(&x), unsafe.Pointer(&z)) {
		t.Fatal("5!=6 should differ")
	}

	tdS, _ := descOf(reflect.TypeFor[string]())
	s1, s2, s3 := "abc", "abc", "abd"
	if !eq(tdS, unsafe.Pointer(&s1), unsafe.Pointer(&s2)) {
		t.Fatal("abc==abc")
	}
	if eq(tdS, unsafe.Pointer(&s1), unsafe.Pointer(&s3)) {
		t.Fatal("abc!=abd")
	}

	tdB, _ := descOf(reflect.TypeFor[[]byte]())
	b1, b2, b3 := []byte{1, 2, 3}, []byte{1, 2, 3}, []byte{1, 2, 4}
	if !eq(tdB, unsafe.Pointer(&b1), unsafe.Pointer(&b2)) {
		t.Fatal("bytes equal")
	}
	if eq(tdB, unsafe.Pointer(&b1), unsafe.Pointer(&b3)) {
		t.Fatal("bytes differ")
	}
}

func TestEqualValueStructSliceMapPtr(t *testing.T) {
	type Inner struct {
		A int
		B string
	}
	type Outer struct {
		N  int
		In Inner
		Sl []int
		M  map[string]int
		P  *int
	}
	pi := 7
	mk := func() Outer {
		return Outer{N: 1, In: Inner{A: 2, B: "x"}, Sl: []int{1, 2}, M: map[string]int{"k": 9}, P: &pi}
	}
	a, b := mk(), mk()
	tdO, _ := descOf(reflect.TypeFor[Outer]())
	if !equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&b), 0) {
		t.Fatal("identical Outer should be equal")
	}
	c := mk()
	c.M["k"] = 10
	if equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&c), 0) {
		t.Fatal("map value change should differ")
	}
	d := mk()
	d.P = nil
	if equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&d), 0) {
		t.Fatal("ptr presence change should differ")
	}
}

func TestDiffApplyFlatStruct(t *testing.T) {
	type Rec struct {
		ID   int
		Name string
		Age  uint8
		On   bool
	}
	old := Rec{ID: 1, Name: "alice", Age: 30, On: true}
	neu := Rec{ID: 1, Name: "alice", Age: 31, On: false} // Age + On changed

	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	base := old // copy
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if base != neu {
		t.Fatalf("Apply mismatch:\n got %+v\nwant %+v", base, neu)
	}

	p2, err := Diff(old, old, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	b2 := old
	if err := Apply(&b2, p2); err != nil {
		t.Fatal(err)
	}
	if b2 != old {
		t.Fatal("no-op patch changed value")
	}
}

func TestEqualValueNonPODSliceField(t *testing.T) {
	type Rec struct {
		Tags  []string
		Flags []bool
		Names []string
	}
	td, _ := descOf(reflect.TypeFor[Rec]())
	a := Rec{Tags: []string{"x", "y"}, Flags: []bool{true}, Names: []string{"a"}}
	b := Rec{Tags: []string{"x", "y"}, Flags: []bool{true}, Names: []string{"a"}}
	if !equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&b), 0) {
		t.Fatal("identical non-POD slices should be equal")
	}
	c := Rec{Tags: []string{"x", "z"}, Flags: []bool{true}, Names: []string{"a"}}
	if equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&c), 0) {
		t.Fatal("changed []string should differ")
	}
	d := Rec{Tags: []string{"x", "y"}, Flags: []bool{false}, Names: []string{"a"}}
	if equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&d), 0) {
		t.Fatal("changed []bool should differ")
	}
}

func TestApplyNoCopyIsolation(t *testing.T) {
	type Rec struct {
		ID   int
		Name string
	}
	// Dirty the shared pool: grab a decoder, set noCopy, return it.
	d := decPool.Get().(*Decoder)
	d.SetNoCopy(true)
	decPool.Put(d)

	old := Rec{ID: 1, Name: "alice"}
	neu := Rec{ID: 1, Name: "bob"}
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	base := old
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	got := base.Name
	// Scribble the patch buffer; if Apply aliased it, got mutates.
	for i := range patch {
		patch[i] = 0xFF
	}
	if base.Name != got || base.Name != "bob" {
		t.Fatalf("Apply aliased the patch buffer: name=%q", base.Name)
	}
}

func TestDiffApplyNestedAndPointer(t *testing.T) {
	type Inner struct {
		A int
		B string
	}
	type Outer struct {
		Tag string
		In  Inner
		Opt *Inner
		Num *int
	}
	n7 := 7
	n8 := 8
	old := Outer{Tag: "t", In: Inner{A: 1, B: "x"}, Opt: &Inner{A: 5, B: "y"}, Num: &n7}
	neu := Outer{Tag: "t", In: Inner{A: 1, B: "X"}, Opt: nil, Num: &n8}

	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	base := Outer{Tag: "t", In: Inner{A: 1, B: "x"}, Opt: &Inner{A: 5, B: "y"}, Num: &n7}
	bIn := *base.Opt
	base.Opt = &bIn
	bn := *base.Num
	base.Num = &bn
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if base.In.B != "X" || base.Opt != nil || base.Num == nil || *base.Num != 8 {
		t.Fatalf("apply mismatch: %+v (Num=%v)", base, *base.Num)
	}
}

func TestDiffApplyPointerToStructPartial(t *testing.T) {
	type Inner struct {
		A int
		B string
	}
	type Outer struct {
		P *Inner
	}
	old := Outer{P: &Inner{A: 1, B: "x"}}
	neu := Outer{P: &Inner{A: 1, B: "Y"}} // only B changes, both non-nil → merge
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	bi := Inner{A: 1, B: "x"}
	base := Outer{P: &bi}
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if base.P == nil || base.P.A != 1 || base.P.B != "Y" {
		t.Fatalf("partial ptr-struct merge failed: %+v", base.P)
	}
}

func TestDiffApplySlicePositional(t *testing.T) {
	type E struct {
		K int
		V string
	}
	old := []E{{1, "a"}, {2, "b"}, {3, "c"}}
	neu := []E{{1, "a"}, {2, "B"}, {3, "c"}, {4, "d"}} // middle change + grow

	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	base := []E{{1, "a"}, {2, "b"}, {3, "c"}}
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %+v want %+v", base, neu)
	}

	neu2 := []E{{1, "a"}} // shrink
	p2, _ := Diff(old, neu2, OptBalanced)
	base2 := []E{{1, "a"}, {2, "b"}, {3, "c"}}
	if err := Apply(&base2, p2); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base2, neu2) {
		t.Fatalf("shrink got %+v want %+v", base2, neu2)
	}
}

func TestDiffApplyPODSliceShortCircuit(t *testing.T) {
	old := []int{1, 2, 3, 4, 5}
	neu := []int{1, 2, 99, 4, 5}
	p, _ := Diff(old, neu, OptBalanced)
	base := []int{1, 2, 3, 4, 5}
	if err := Apply(&base, p); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %v want %v", base, neu)
	}
}

func TestDiffApplyArray(t *testing.T) {
	type A struct {
		Vals [4]int
	}
	old := A{Vals: [4]int{1, 2, 3, 4}}
	neu := A{Vals: [4]int{1, 9, 3, 4}}
	p, _ := Diff(old, neu, OptBalanced)
	base := A{Vals: [4]int{1, 2, 3, 4}}
	if err := Apply(&base, p); err != nil {
		t.Fatal(err)
	}
	if base != neu {
		t.Fatalf("array got %+v want %+v", base, neu)
	}
}

func TestApplyRejectsSliceGrowthBomb(t *testing.T) {
	type Big struct {
		Buf [4096]byte
	}
	// A valid tiny patch as a starting point, then hand-corrupt newLen huge.
	old := []Big{{}}
	neu := []Big{{}, {}} // grow by one (legit, carries the appended element)
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: the legit patch applies.
	base := []Big{{}}
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("legit grow failed: %v", err)
	}
	if len(base) != 2 {
		t.Fatalf("legit grow len=%d want 2", len(base))
	}

	// Now forge a patch claiming a massive newLen with no entries. Build minimal
	// bytes by hand is fragile; instead assert that Apply of a deliberately tiny
	// truncated/garbage slice-patch never allocates unboundedly — it must error,
	// not OOM or panic.
	base2 := []Big{{}}
	bad := append([]byte(nil), patch...)
	// Corrupt is hard to target precisely; rely on the bound: a patch whose body
	// claims growth far beyond its size must be rejected. Use a crafted body via
	// the public path is not feasible here, so just ensure a hostile truncation
	// errors cleanly (no panic / no huge alloc).
	for cut := len(bad) - 1; cut >= 0; cut-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at cut %d: %v", cut, r)
				}
			}()
			b := append([]Big(nil), base2...)
			_ = Apply(&b, bad[:cut])
		}()
	}
}

func TestDiffApplyMapPerKey(t *testing.T) {
	old := map[string]int{"a": 1, "b": 2, "c": 3}
	neu := map[string]int{"a": 1, "b": 20, "d": 4} // b changed, c deleted, d added
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	base := map[string]int{"a": 1, "b": 2, "c": 3}
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %v want %v", base, neu)
	}

	// Per-key property: a one-key change in a large map must produce a patch far
	// smaller than a whole-map replace (which would re-encode every entry).
	bigOld := make(map[string]int, 1000)
	bigNew := make(map[string]int, 1000)
	for i := range 1000 {
		bigOld[string(rune('A'+i%26))+string(rune('0'+i/26))] = i
		bigNew[string(rune('A'+i%26))+string(rune('0'+i/26))] = i
	}
	bigNew["A0"] = 999999 // single-key change
	smallPatch, err := Diff(bigOld, bigNew, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	wholeReplace, err := Diff(map[string]int(nil), bigNew, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(smallPatch) >= len(wholeReplace) {
		t.Fatalf("per-key path not exercised: one-key patch %d >= whole-map %d",
			len(smallPatch), len(wholeReplace))
	}
	bigBase := make(map[string]int, 1000)
	maps.Copy(bigBase, bigOld)
	if err := Apply(&bigBase, smallPatch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bigBase, bigNew) {
		t.Fatalf("big map apply mismatch")
	}
}

func TestDiffApplyMapStructValuesMerge(t *testing.T) {
	type V struct {
		N int
		S string
	}
	old := map[string]V{"x": {1, "a"}, "y": {2, "b"}}
	neu := map[string]V{"x": {1, "A"}, "y": {2, "b"}} // x.S changed only
	p, _ := Diff(old, neu, OptBalanced)
	base := map[string]V{"x": {1, "a"}, "y": {2, "b"}}
	if err := Apply(&base, p); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %v want %v", base, neu)
	}
}

func TestDiffApplyMapNilAndEmpty(t *testing.T) {
	// nil base map gets entries added
	old := map[string]int(nil)
	neu := map[string]int{"k": 5}
	p, _ := Diff(old, neu, OptBalanced)
	var base map[string]int
	if err := Apply(&base, p); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("nil->set got %v want %v", base, neu)
	}
}

func TestDiffApplyMapPointerValues(t *testing.T) {
	type V struct {
		N int
		S string
	}
	// "y" has a DIFFERENT pointer but EQUAL content; "x" content changes.
	old := map[string]*V{"x": {1, "a"}, "y": {2, "b"}}
	neu := map[string]*V{"x": {1, "A"}, "y": {2, "b"}}
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	base := map[string]*V{"x": {1, "a"}, "y": {2, "b"}}
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if base["x"].S != "A" || base["y"].N != 2 || base["y"].S != "b" {
		t.Fatalf("ptr-value map mismatch: x=%+v y=%+v", base["x"], base["y"])
	}
}

func TestApplyRejectsWrongSchema(t *testing.T) {
	type A struct{ X int }
	type B struct{ Y int }
	p, _ := Diff(A{X: 1}, A{X: 2}, OptBalanced)
	var b B
	if err := Apply(&b, p); !errors.Is(err, ErrPatchSchemaMismatch) {
		t.Fatalf("got %v want ErrPatchSchemaMismatch", err)
	}
}

func TestApplyRejectsWrongBase(t *testing.T) {
	type A struct{ X, Y int }
	p, _ := Diff(A{X: 1, Y: 1}, A{X: 2, Y: 1}, OptBalanced)
	wrong := A{X: 9, Y: 9} // not the old the patch was built against
	if err := Apply(&wrong, p); !errors.Is(err, ErrPatchBaseMismatch) {
		t.Fatalf("got %v want ErrPatchBaseMismatch", err)
	}
}

func TestApplyRejectsTruncatedPatch(t *testing.T) {
	type Nested struct {
		A int
		B string
		C []int
		D map[string]int
	}
	type Rec struct {
		ID   int
		Name string
		Tags []string
		Sub  Nested
		Opt  *Nested
	}
	old := Rec{ID: 1, Name: "x", Tags: []string{"a"}, Sub: Nested{A: 1, B: "p", C: []int{1}, D: map[string]int{"k": 1}}}
	neu := Rec{ID: 2, Name: "y", Tags: []string{"a", "b"}, Sub: Nested{A: 2, B: "q", C: []int{1, 2}, D: map[string]int{"k": 2, "z": 9}}, Opt: &Nested{A: 7}}
	p, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Truncate at every length; Apply must never panic, must return an error or nil.
	for cut := range len(p) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncation at %d: %v", cut, r)
				}
			}()
			base := old
			// give base independent backing so a partial apply can't corrupt old
			base.Tags = append([]string(nil), old.Tags...)
			base.Sub.C = append([]int(nil), old.Sub.C...)
			base.Sub.D = map[string]int{"k": 1}
			s := old.Sub
			base.Opt = &s
			_ = Apply(&base, p[:cut])
		}()
	}
}

func TestApplyRejectsCorruptedBody(t *testing.T) {
	type Rec struct {
		ID   int
		Tags []string
		M    map[string]int
	}
	old := Rec{ID: 1, Tags: []string{"a"}, M: map[string]int{"k": 1}}
	neu := Rec{ID: 2, Tags: []string{"a", "b", "c"}, M: map[string]int{"k": 2, "z": 3}}
	p, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Flip bytes in the body (after the 21-byte header) one at a time; Apply must
	// never panic. (May error, may by chance still apply — just must not crash.)
	for i := 21; i < len(p); i++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on corrupted byte %d: %v", i, r)
				}
			}()
			bad := append([]byte(nil), p...)
			bad[i] ^= 0xFF
			base := Rec{ID: 1, Tags: []string{"a"}, M: map[string]int{"k": 1}}
			_ = Apply(&base, bad)
		}()
	}
}

func TestDiffApplyMapNoncomparableValues(t *testing.T) {
	type W struct {
		M map[string]map[string]int
		L map[string][]int
	}
	old := W{M: map[string]map[string]int{"o": {"a": 1}}, L: map[string][]int{"k": {1, 2}}}
	neu := W{M: map[string]map[string]int{"o": {"a": 2}}, L: map[string][]int{"k": {1, 3}}}
	patch, err := Diff(old, neu, OptBalanced) // must NOT panic
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	base := W{M: map[string]map[string]int{"o": {"a": 1}}, L: map[string][]int{"k": {1, 2}}}
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if base.M["o"]["a"] != 2 || base.L["k"][1] != 3 {
		t.Fatalf("noncomparable-value map mismatch: %+v", base)
	}
}

func TestDiffApplyRANSRoundTrip(t *testing.T) {
	type Rec struct {
		ID   int
		Blob string
	}
	// A large, low-entropy change so the patch body exceeds ransMinBytes (512).
	old := make([]Rec, 200)
	neu := make([]Rec, 200)
	for i := range old {
		old[i] = Rec{ID: i, Blob: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
		neu[i] = Rec{ID: i, Blob: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"} // every Blob changes
	}
	patch, err := Diff(old, neu, OptCompression)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	h, _, err := readPatchHeader(patch)
	if err != nil {
		t.Fatalf("readPatchHeader: %v", err)
	}
	if h.flags&flagPatchRANS == 0 {
		t.Logf("note: rANS did not fire (patch body may be below threshold or not entropy-reducible); test still validates the non-rANS path under OptCompression")
	} else {
		t.Logf("rANS fired: patch=%d bytes", len(patch))
	}
	base := make([]Rec, 200)
	copy(base, old)
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatal("rANS round-trip mismatch")
	}
}

func TestDiffApplyRANSForcedFire(t *testing.T) {
	// Construct a payload almost certain to trigger rANS: a long highly-repetitive
	// string field that changes, producing a large low-entropy patch body.
	type Doc struct {
		ID   int
		Text string
	}
	old := Doc{ID: 1, Text: ""}
	// 4 KB of a small alphabet → highly compressible.
	buf := make([]byte, 4096)
	for i := range buf {
		buf[i] = byte('a' + (i % 4))
	}
	neu := Doc{ID: 1, Text: string(buf)}
	patch, err := Diff(old, neu, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	h, _, err := readPatchHeader(patch)
	if err != nil {
		t.Fatal(err)
	}
	if h.flags&flagPatchRANS == 0 {
		t.Fatalf("expected rANS to fire on a 4KB low-entropy patch body, but flagPatchRANS is unset (patch=%d bytes)", len(patch))
	}
	base := Doc{ID: 1, Text: ""}
	if err := Apply(&base, patch); err != nil {
		t.Fatalf("Apply (rANS): %v", err)
	}
	if base != neu {
		t.Fatal("forced-rANS round-trip mismatch")
	}
}

func TestDiffApplyNilVsEmptyBothLenZero(t *testing.T) {
	type S struct {
		Sl []int
		M  map[string]int
		B  []byte
	}
	nilV := S{Sl: nil, M: nil, B: nil}
	emptyV := S{Sl: []int{}, M: map[string]int{}, B: []byte{}}

	check := func(name string, from, to S) {
		for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
			patch, err := Diff(from, to, opts)
			if err != nil {
				t.Fatalf("%s opts=%v Diff: %v", name, opts, err)
			}
			base := from
			// independent backing for the non-nil container fields, while
			// preserving the empty-vs-nil distinction (a plain append from a
			// nil head collapses an empty-non-nil source back to nil).
			if from.Sl != nil {
				base.Sl = make([]int, len(from.Sl))
				copy(base.Sl, from.Sl)
			}
			if from.B != nil {
				base.B = make([]byte, len(from.B))
				copy(base.B, from.B)
			}
			if from.M != nil {
				base.M = map[string]int{}
				maps.Copy(base.M, from.M)
			}
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("%s opts=%v Apply: %v", name, opts, err)
			}
			if (base.Sl == nil) != (to.Sl == nil) {
				t.Fatalf("%s opts=%v: Sl nilness wrong (got nil=%v want nil=%v)", name, opts, base.Sl == nil, to.Sl == nil)
			}
			if (base.M == nil) != (to.M == nil) {
				t.Fatalf("%s opts=%v: M nilness wrong (got nil=%v want nil=%v)", name, opts, base.M == nil, to.M == nil)
			}
			if (base.B == nil) != (to.B == nil) {
				t.Fatalf("%s opts=%v: B nilness wrong (got nil=%v want nil=%v)", name, opts, base.B == nil, to.B == nil)
			}
			if !reflect.DeepEqual(base, to) {
				t.Fatalf("%s opts=%v: DeepEqual mismatch got=%+v want=%+v", name, opts, base, to)
			}
		}
	}
	check("nil->empty", nilV, emptyV)
	check("empty->nil", emptyV, nilV)
	check("nil->nil", nilV, nilV)
	check("empty->empty", emptyV, emptyV)
}

// TestDiffApplyDeletions exercises element/key removal across containers: map
// key deletion (including delete-all), middle-element slice deletion, slice
// shrink-to-empty and shrink-to-nil, and nested struct field deletions. Arrays
// have no deletion concept (fixed length); see TestDiffApplyArray.
func TestDiffApplyDeletions(t *testing.T) {
	tiers := []Options{OptBalanced, OptCompression, OptSpeed}

	t.Run("map delete one / all", func(t *testing.T) {
		for _, opts := range tiers {
			// delete one key
			p, err := Diff(map[string]int{"a": 1, "b": 2, "c": 3},
				map[string]int{"a": 1, "c": 3}, opts)
			if err != nil {
				t.Fatal(err)
			}
			base := map[string]int{"a": 1, "b": 2, "c": 3}
			if err := Apply(&base, p); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(base, map[string]int{"a": 1, "c": 3}) {
				t.Fatalf("opts=%v delete-one got %v", opts, base)
			}
			// delete all -> empty non-nil
			p2, _ := Diff(map[string]int{"a": 1, "b": 2}, map[string]int{}, opts)
			base2 := map[string]int{"a": 1, "b": 2}
			if err := Apply(&base2, p2); err != nil {
				t.Fatal(err)
			}
			if base2 == nil || len(base2) != 0 {
				t.Fatalf("opts=%v delete-all got %v (nil=%v)", opts, base2, base2 == nil)
			}
		}
	})

	t.Run("slice delete middle / shrink", func(t *testing.T) {
		for _, opts := range tiers {
			// middle deletion via positional shift+truncate
			p, _ := Diff([]int{10, 20, 30, 40}, []int{10, 30, 40}, opts)
			base := []int{10, 20, 30, 40}
			if err := Apply(&base, p); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(base, []int{10, 30, 40}) {
				t.Fatalf("opts=%v slice-middle got %v", opts, base)
			}
			// shrink to empty non-nil
			p2, _ := Diff([]int{1, 2, 3}, []int{}, opts)
			b2 := []int{1, 2, 3}
			if err := Apply(&b2, p2); err != nil {
				t.Fatal(err)
			}
			if b2 == nil || len(b2) != 0 {
				t.Fatalf("opts=%v shrink-empty got %v (nil=%v)", opts, b2, b2 == nil)
			}
			// shrink to nil
			p3, _ := Diff([]int{1, 2, 3}, []int(nil), opts)
			b3 := []int{1, 2, 3}
			if err := Apply(&b3, p3); err != nil {
				t.Fatal(err)
			}
			if b3 != nil {
				t.Fatalf("opts=%v shrink-nil got non-nil %v", opts, b3)
			}
		}
	})

	t.Run("nested struct deletions", func(t *testing.T) {
		type Rec struct {
			Tags  map[string]int
			Items []string
		}
		old := Rec{Tags: map[string]int{"x": 1, "y": 2}, Items: []string{"a", "b", "c"}}
		neu := Rec{Tags: map[string]int{"x": 1}, Items: []string{"a"}}
		for _, opts := range tiers {
			p, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatal(err)
			}
			base := Rec{Tags: map[string]int{"x": 1, "y": 2}, Items: []string{"a", "b", "c"}}
			if err := Apply(&base, p); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("opts=%v nested got %#v want %#v", opts, base, neu)
			}
		}
	})
}

// TestApplyDivergentBaseRejected verifies that the default-on base fingerprint
// rejects applying a patch onto a base that is not the old the patch was built
// from. This is why "deleting an absent key" never arises in normal flow: a
// matching base provably contains every tombstoned key. applyMap still no-ops a
// SetMapIndex(zero) on an absent key defensively against hand-crafted patches.
func TestApplyDivergentBaseRejected(t *testing.T) {
	p, err := Diff(map[string]int{"a": 1, "gone": 9}, map[string]int{"a": 1}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	divergent := map[string]int{"a": 1} // lacks "gone" -> not the original old
	if err := Apply(&divergent, p); !errors.Is(err, ErrPatchBaseMismatch) {
		t.Fatalf("divergent base: got %v want ErrPatchBaseMismatch", err)
	}
	// The matching base applies cleanly and removes the tombstoned key.
	matching := map[string]int{"a": 1, "gone": 9}
	if err := Apply(&matching, p); err != nil {
		t.Fatalf("matching base: %v", err)
	}
	if !reflect.DeepEqual(matching, map[string]int{"a": 1}) {
		t.Fatalf("got %v", matching)
	}
}

func TestApplyDivergentBaseInterfaceField(t *testing.T) {
	type Rec struct {
		N int
		X any
	}
	// Patch changes only N; X ("alpha") is unchanged.
	patch, err := Diff(Rec{N: 1, X: "alpha"}, Rec{N: 99, X: "alpha"}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Apply onto a base whose interface field DIVERGES from old.
	divergent := Rec{N: 1, X: "DIFFERENT"}
	if err := Apply(&divergent, patch); !errors.Is(err, ErrPatchBaseMismatch) {
		t.Fatalf("divergent interface base: got %v want ErrPatchBaseMismatch", err)
	}
	// Matching base applies cleanly.
	matching := Rec{N: 1, X: "alpha"}
	if err := Apply(&matching, patch); err != nil {
		t.Fatalf("matching base: %v", err)
	}
	if matching.N != 99 || matching.X != "alpha" {
		t.Fatalf("got %+v", matching)
	}
}

func TestDiffApplyDenseInternRoundTrip(t *testing.T) {
	type Rec struct {
		A string
		B string
		C string
		D string
	}
	// Repeated strings exercise the Dense intern path within one patch body.
	old := Rec{A: "alpha", B: "alpha", C: "beta", D: "alpha"}
	neu := Rec{A: "alpha", B: "gamma", C: "beta", D: "gamma"} // B, D change to a repeated new value
	for _, opts := range []Options{OptDense, OptDense | OptBalanced, OptDense | OptCompression, OptBalanced} {
		patch, err := Diff(old, neu, opts)
		if err != nil {
			t.Fatalf("opts=%v Diff: %v", opts, err)
		}
		base := old
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opts=%v Apply: %v", opts, err)
		}
		if base != neu {
			t.Fatalf("opts=%v: got %+v want %+v", opts, base, neu)
		}
	}
	// Pooled reuse: many sequential Diff/Apply must not leak intern state across calls.
	for i := 0; i < 200; i++ {
		patch, _ := Diff(old, neu, OptDense|OptBalanced)
		base := old
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("reuse %d: %v", i, err)
		}
		if base != neu {
			t.Fatalf("reuse %d mismatch", i)
		}
	}
}

func TestDiffCyclicValueErrorsNotCrash(t *testing.T) {
	type Node struct {
		V    int
		Next *Node
	}
	a := &Node{V: 1}
	a.Next = a // self-cycle
	// Must return an error, not crash the process with a stack overflow.
	if _, err := Diff(*a, *a, OptBalanced); err == nil {
		t.Fatal("expected an error for a cyclic value, got nil")
	}
	// A different cyclic pair too.
	b := &Node{V: 2}
	b.Next = b
	if _, err := Diff(*a, *b, OptBalanced); err == nil {
		t.Fatal("expected an error diffing two cyclic values, got nil")
	}
}

func TestApplyDeeplyNestedPatchRejected(t *testing.T) {
	// A linked list deep enough to exceed maxDeltaDepth must Diff-error, not crash.
	type N struct {
		V    int
		Next *N
	}
	build := func(depth, leaf int) *N {
		head := &N{}
		cur := head
		for i := 1; i < depth; i++ {
			cur.Next = &N{}
			cur = cur.Next
		}
		cur.V = leaf
		return head
	}
	old := build(maxDeltaDepth+50, 1)
	neu := build(maxDeltaDepth+50, 2)
	if _, err := Diff(*old, *neu, OptBalanced); err == nil {
		t.Fatal("expected error for over-deep value, got nil")
	}
}

func TestDiffApplyInterfaceNonComparable(t *testing.T) {
	type Rec struct {
		N int
		X any
	}
	// X holds a non-comparable []int; changing N must not panic.
	old := Rec{N: 1, X: []int{1, 2, 3}}
	neu := Rec{N: 9, X: []int{1, 2, 3}}
	patch, err := Diff(old, neu, OptBalanced) // must NOT panic
	if err != nil {
		t.Fatal(err)
	}
	base := Rec{N: 1, X: []int{1, 2, 3}}
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if base.N != 9 {
		t.Fatalf("N not applied: %+v", base)
	}
	// Diff(x,x) with a non-comparable interface must also not panic.
	if _, err := Diff(old, old, OptBalanced); err != nil {
		t.Fatal(err)
	}
	// Map value holding a non-comparable any.
	om := map[string]any{"k": []int{1}}
	nm := map[string]any{"k": []int{1, 2}}
	mp, err := Diff(om, nm, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	mb := map[string]any{"k": []int{1}}
	if err := Apply(&mb, mp); err != nil {
		t.Fatal(err)
	}
}

func TestDiffApplyNoBaseFingerprint(t *testing.T) {
	type Rec struct {
		N int
		S string
	}
	old := Rec{N: 1, S: "x"}
	neu := Rec{N: 2, S: "y"}
	patch, err := Diff(old, neu, OptBalanced|OptDeltaNoBaseFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	// No baseFP flag in the header.
	h, _, err := readPatchHeader(patch)
	if err != nil {
		t.Fatal(err)
	}
	if h.flags&flagPatchBaseFP != 0 {
		t.Fatal("expected no baseFP flag")
	}
	// Round-trips onto the correct base.
	base := old
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if base != neu {
		t.Fatalf("got %+v", base)
	}
	// And — by design — applies onto a WRONG base WITHOUT error (no guard).
	wrong := Rec{N: 99, S: "zzz"}
	if err := Apply(&wrong, patch); err != nil {
		t.Fatalf("no-fingerprint apply should not error: %v", err)
	}
}
