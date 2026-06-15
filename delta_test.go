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
