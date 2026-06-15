package qdf

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

// mcNC is a Marshaler/Unmarshaler struct with a NON-COMPARABLE field.
type mcNC struct{ Vals []int32 }

func (m mcNC) MarshalQDF(dst []byte) ([]byte, error) {
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(m.Vals)))
	for _, v := range m.Vals {
		dst = binary.LittleEndian.AppendUint32(dst, uint32(v))
	}
	return dst, nil
}
func (m *mcNC) UnmarshalQDF(src []byte) (int, error) {
	if len(src) < 4 {
		return 0, ErrShortBuffer
	}
	n := binary.LittleEndian.Uint32(src[:4])
	i := 4
	m.Vals = make([]int32, n)
	for k := range int(n) {
		if i+4 > len(src) {
			return 0, ErrShortBuffer
		}
		m.Vals[k] = int32(binary.LittleEndian.Uint32(src[i:]))
		i += 4
	}
	return i, nil
}

func TestDiffMarshalerNonComparableNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Diff panicked on marshaler-with-noncomparable: %v", r)
		}
	}()
	old := mcNC{Vals: []int32{1, 2, 3}}
	neu := mcNC{Vals: []int32{1, 2, 9}}
	if _, err := Diff(old, neu, OptBalanced); err != nil {
		t.Fatalf("Diff err: %v", err)
	}
}

func TestFingerprintNotBlindToTimeField(t *testing.T) {
	type Rec struct {
		T time.Time
		N int
	}
	a := Rec{T: time.Unix(1000, 0), N: 5}
	b := Rec{T: time.Unix(9999, 0), N: 5} // differs ONLY in time
	td, _ := descOf(reflect.TypeFor[Rec]())
	fa := valueFingerprint(td, unsafe.Pointer(&a))
	fb := valueFingerprint(td, unsafe.Pointer(&b))
	if fa == fb {
		t.Fatal("two values differing only in a time.Time field have the SAME fingerprint (blind spot)")
	}
}

func TestSchemaFPDistinguishesMapKeyAndArrayLen(t *testing.T) {
	tdI, _ := descOf(reflect.TypeFor[map[int64]int64]())
	tdS, _ := descOf(reflect.TypeFor[map[string]int64]())
	if tdI.schemaFP == tdS.schemaFP {
		t.Fatal("map[int64]V and map[string]V collide in schemaFP")
	}
	tdA3, _ := descOf(reflect.TypeFor[[3]int64]())
	tdA4, _ := descOf(reflect.TypeFor[[4]int64]())
	if tdA3.schemaFP == tdA4.schemaFP {
		t.Fatal("[3]int64 and [4]int64 collide in schemaFP")
	}
}
