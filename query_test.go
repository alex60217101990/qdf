package qdf

import "testing"

func TestWhere_NativePredicate_Int(t *testing.T) {
	opt := Where("status", func(c int) bool { return c >= 500 })
	if opt.term == nil {
		t.Fatal("Where did not produce a predicate term")
	}
	if opt.term.field != "status" || opt.term.want != colKindInt {
		t.Fatalf("term = %+v, want field=status kind=int", opt.term)
	}
	got := make([]bool, 0, 4)
	for _, v := range []int64{200, 500, 404, 503} {
		got = append(got, opt.term.pI64(v))
	}
	want := []bool{false, true, false, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pI64 %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestWhere_NativePredicate_String(t *testing.T) {
	opt := Where("level", func(s string) bool { return s == "ERROR" })
	if opt.term.want != colKindString || opt.term.pStr == nil {
		t.Fatalf("term = %+v, want kind=string", opt.term)
	}
	if !opt.term.pStr("ERROR") || opt.term.pStr("INFO") {
		t.Fatal("pStr mismatch")
	}
}

func TestSelect_Option(t *testing.T) {
	opt := Select("a", "b")
	if len(opt.selectFields) != 2 || opt.selectFields[0] != "a" || opt.selectFields[1] != "b" {
		t.Fatalf("select fields = %v", opt.selectFields)
	}
}

func TestBitset_AndPopcountMatched(t *testing.T) {
	a := newBitset(10)
	b := newBitset(10)
	for _, i := range []int{1, 3, 5, 7} {
		setBit(a, i)
	}
	for _, i := range []int{3, 5, 9} {
		setBit(b, i)
	}
	bitsetAnd(a, b) // a &= b  => {3,5}
	if pc := popcount(a); pc != 2 {
		t.Fatalf("popcount = %d, want 2", pc)
	}
	got := matchedIndices(a, 10, nil)
	want := []int{3, 5}
	if len(got) != len(want) || got[0] != 3 || got[1] != 5 {
		t.Fatalf("matched = %v, want %v", got, want)
	}
}
