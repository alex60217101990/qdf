package qdf

import (
	"reflect"
	"testing"
	"unsafe"
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

func TestKeyToken(t *testing.T) {
	type E struct {
		ID string `qdf:"id,key"`
		N  int
	}
	td, _ := descOf(reflect.TypeFor[E]())
	a := E{ID: "alpha", N: 1}
	b := E{ID: "alpha", N: 2}
	c := E{ID: "beta", N: 1}
	ta := keyToken(td, unsafe.Pointer(&a))
	tb := keyToken(td, unsafe.Pointer(&b))
	tc := keyToken(td, unsafe.Pointer(&c))
	if ta != tb {
		t.Fatal("same key content must yield equal tokens")
	}
	if ta == tc {
		t.Fatal("different keys must yield different tokens")
	}

	type EI struct {
		ID int64 `qdf:"id,key"`
		N  int
	}
	tdi, _ := descOf(reflect.TypeFor[EI]())
	x := EI{ID: 7}
	y := EI{ID: 7}
	z := EI{ID: 8}
	if keyToken(tdi, unsafe.Pointer(&x)) != keyToken(tdi, unsafe.Pointer(&y)) {
		t.Fatal("int64 key 7==7 token")
	}
	if keyToken(tdi, unsafe.Pointer(&x)) == keyToken(tdi, unsafe.Pointer(&z)) {
		t.Fatal("int64 key 7!=8 token")
	}
}

func TestKeyTokenByteArray(t *testing.T) {
	type E struct {
		ID [4]byte `qdf:"id,key"`
		N  int
	}
	td, _ := descOf(reflect.TypeFor[E]())
	a := E{ID: [4]byte{1, 2, 3, 4}}
	b := E{ID: [4]byte{1, 2, 3, 4}}
	c := E{ID: [4]byte{1, 2, 3, 5}}
	if keyToken(td, unsafe.Pointer(&a)) != keyToken(td, unsafe.Pointer(&b)) {
		t.Fatal("[4]byte key equal")
	}
	if keyToken(td, unsafe.Pointer(&a)) == keyToken(td, unsafe.Pointer(&c)) {
		t.Fatal("[4]byte key differ")
	}
}

func TestKeyedPatchUsesKeyedTag(t *testing.T) {
	type E struct {
		ID  string `qdf:"id,key"`
		Val int
	}
	// A pure reorder is the case the keyed differ wins decisively: preserved
	// elements cost nothing, so the keyed patch (a new-order key list) beats the
	// positional alternative (which re-replaces every shifted element). The
	// never-larger picker must therefore keep the tagKeyedSlicePatch body here.
	// (On a one-element value change the positional body is smaller — an index
	// beats a key — so the picker correctly declines the keyed tag; that path is
	// exercised by the round-trip matrix elsewhere.)
	old := []E{{"a", 1}, {"b", 2}, {"c", 3}, {"d", 4}}
	neu := []E{{"d", 4}, {"c", 3}, {"b", 2}, {"a", 1}}
	patch, _ := Diff(old, neu, OptBalanced)
	if !containsByte(patch, tagKeyedSlicePatch) {
		t.Fatal("a reorder must keep the tagKeyedSlicePatch body (keyed beats positional)")
	}
}

func TestKeyTokenAtMatchesKeyToken(t *testing.T) {
	// keyTokenAt over the key value alone must equal keyToken over the element.
	type E struct {
		Pad int
		ID  string `qdf:"id,key"`
	}
	td, _ := descOf(reflect.TypeFor[E]())
	e := E{Pad: 9, ID: "k"}
	full := keyToken(td, unsafe.Pointer(&e))
	kp := unsafe.Add(unsafe.Pointer(&e), td.keyOff)
	at := keyTokenAt(td.keyDesc, kp)
	if full != at {
		t.Fatalf("keyTokenAt %q != keyToken %q", at, full)
	}
}

func TestDiffApplyKeyedReorderAndChange(t *testing.T) {
	type E struct {
		ID   string `qdf:"id,key"`
		Val  int
		Note string
	}
	old := []E{{"a", 1, "x"}, {"b", 2, "y"}, {"c", 3, "z"}}
	neu := []E{{"c", 3, "z"}, {"a", 10, "x"}, {"b", 2, "y"}, {"d", 4, "w"}}
	for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
		patch, err := Diff(old, neu, opts)
		if err != nil {
			t.Fatalf("opts=%v Diff: %v", opts, err)
		}
		base := append([]E(nil), old...)
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opts=%v Apply: %v", opts, err)
		}
		if !reflect.DeepEqual(base, neu) {
			t.Fatalf("opts=%v: got %+v want %+v", opts, base, neu)
		}
	}
}

func TestDiffApplyKeyedValueOnlyNoOrder(t *testing.T) {
	type E struct {
		ID  int64 `qdf:"id,key"`
		Val int
	}
	old := []E{{1, 10}, {2, 20}, {3, 30}}
	neu := []E{{1, 10}, {2, 99}, {3, 30}} // same order, one value changed
	patch, _ := Diff(old, neu, OptBalanced)
	base := append([]E(nil), old...)
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %+v want %+v", base, neu)
	}
}

func TestDiffApplyKeyedDeleteInsert(t *testing.T) {
	type E struct {
		ID  int64 `qdf:"id,key"`
		Val int
	}
	old := []E{{1, 1}, {2, 2}, {3, 3}, {4, 4}}
	neu := []E{{1, 1}, {3, 30}, {5, 5}} // delete 2&4, change 3, add 5, reorder
	patch, _ := Diff(old, neu, OptBalanced)
	base := append([]E(nil), old...)
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %+v want %+v", base, neu)
	}
}
