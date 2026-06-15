package qdf

import (
	"reflect"
	"testing"
)

func TestKeyTagParsed(t *testing.T) {
	type Entity struct {
		ID string `qdf:"id,key"`
		X  float64
	}
	td, err := descOf(reflect.TypeFor[Entity]())
	if err != nil {
		t.Fatal(err)
	}
	if !td.keyed {
		t.Fatal("Entity should be keyed (ID has ,key)")
	}
	if td.keyOff != reflect.TypeFor[Entity]().Field(0).Offset {
		t.Fatalf("keyOff=%d want %d", td.keyOff, reflect.TypeFor[Entity]().Field(0).Offset)
	}
	if td.keyDesc == nil || td.keyDesc.kind != reflect.String {
		t.Fatalf("keyDesc wrong: %+v", td.keyDesc)
	}
}

func TestKeyTagUntaggedNotKeyed(t *testing.T) {
	type Plain struct {
		ID string
		X  float64
	}
	td, _ := descOf(reflect.TypeFor[Plain]())
	if td.keyed {
		t.Fatal("Plain has no ,key — must not be keyed")
	}
}

func TestKeyTagNonComparableRejected(t *testing.T) {
	type Bad struct {
		K []int `qdf:"k,key"` // slice key is not comparable
	}
	if _, err := descOf(reflect.TypeFor[Bad]()); err == nil {
		t.Fatal("a non-comparable key field must be a build error")
	}
}

func TestKeyTagDoubleRejected(t *testing.T) {
	type Two struct {
		A int `qdf:"a,key"`
		B int `qdf:"b,key"`
	}
	if _, err := descOf(reflect.TypeFor[Two]()); err == nil {
		t.Fatal("two ,key fields must be a build error")
	}
}
