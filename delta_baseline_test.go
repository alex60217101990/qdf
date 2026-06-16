package qdf

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestBaselineRegistryEmpty(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	if r == nil {
		t.Fatal("NewBaselineRegistry returned nil")
	}
	if got := r.Len(); got != 0 {
		t.Fatalf("empty registry Len() = %d, want 0", got)
	}
}

func TestDeepCloneFidelity(t *testing.T) {
	td, err := descOf(reflect.TypeFor[dnTop]())
	if err != nil {
		t.Fatal(err)
	}
	for seed := int64(0); seed < 200; seed++ {
		v := gen[dnTop](seed)
		c := deepClone(&v)
		if !reflect.DeepEqual(*c, v) {
			t.Fatalf("seed %d: clone not DeepEqual to original", seed)
		}
		fpV := valueFingerprint(td, unsafe.Pointer(&v))
		fpC := valueFingerprint(td, unsafe.Pointer(c))
		if fpV != fpC {
			t.Fatalf("seed %d: clone fingerprint %x != original %x", seed, fpC, fpV)
		}
	}
}

func TestDeepCloneNilVsEmpty(t *testing.T) {
	type S struct {
		NilSlice   []int
		EmptySlice []int
		NilMap     map[string]int
		EmptyMap   map[string]int
	}
	v := S{EmptySlice: []int{}, EmptyMap: map[string]int{}}
	c := deepClone(&v)
	if c.NilSlice != nil {
		t.Error("nil slice became non-nil")
	}
	if c.EmptySlice == nil {
		t.Error("empty slice became nil")
	}
	if c.NilMap != nil {
		t.Error("nil map became non-nil")
	}
	if c.EmptyMap == nil {
		t.Error("empty map became nil")
	}
}

func TestDeepCloneIndependent(t *testing.T) {
	type S struct {
		Xs []int
		M  map[string]int
	}
	v := S{Xs: []int{1, 2, 3}, M: map[string]int{"a": 1}}
	c := deepClone(&v)
	c.Xs[0] = 99
	c.M["a"] = 99
	if v.Xs[0] != 1 {
		t.Error("mutating clone slice touched original")
	}
	if v.M["a"] != 1 {
		t.Error("mutating clone map touched original")
	}
}
