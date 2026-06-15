package qdf

import (
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
	if schemaFingerprint(tdA1) != schemaFingerprint(tdA2) {
		t.Fatal("fingerprint not stable across calls")
	}
	if schemaFingerprint(tdA1) == schemaFingerprint(tdB) {
		t.Fatal("distinct shapes collided")
	}
}

func TestReadPatchHeaderRejectsBadMagic(t *testing.T) {
	bad := []byte{'X', 'D', 'P', patchVersion1, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if _, _, err := readPatchHeader(bad); err != ErrInvalidPatch {
		t.Fatalf("got %v want ErrInvalidPatch", err)
	}
	if _, _, err := readPatchHeader([]byte{'Q'}); err != ErrInvalidPatch {
		t.Fatalf("short: got %v want ErrInvalidPatch", err)
	}
	truncated := writePatchHeader(nil, flagPatchBaseFP, 1, 2)[:15] // cut mid-baseFP
	if _, _, err := readPatchHeader(truncated); err != ErrInvalidPatch {
		t.Fatalf("truncated baseFP: got %v want ErrInvalidPatch", err)
	}
}

func TestEqualValueScalarsStringsBytes(t *testing.T) {
	eq := func(td *typeDesc, a, b unsafe.Pointer) bool { return equalValue(td, a, b) }

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
	if !equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&b)) {
		t.Fatal("identical Outer should be equal")
	}
	c := mk()
	c.M["k"] = 10
	if equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&c)) {
		t.Fatal("map value change should differ")
	}
	d := mk()
	d.P = nil
	if equalValue(tdO, unsafe.Pointer(&a), unsafe.Pointer(&d)) {
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
	if !equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&b)) {
		t.Fatal("identical non-POD slices should be equal")
	}
	c := Rec{Tags: []string{"x", "z"}, Flags: []bool{true}, Names: []string{"a"}}
	if equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&c)) {
		t.Fatal("changed []string should differ")
	}
	d := Rec{Tags: []string{"x", "y"}, Flags: []bool{false}, Names: []string{"a"}}
	if equalValue(td, unsafe.Pointer(&a), unsafe.Pointer(&d)) {
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
