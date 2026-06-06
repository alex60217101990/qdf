package qdf

import (
	"errors"
	"testing"
)

func TestWhere_NativePredicate_Int(t *testing.T) {
	opt := Where("status", func(c int) bool { return c >= 500 })
	if opt.node == nil || opt.node.term == nil {
		t.Fatal("Where did not produce a predicate term")
	}
	if opt.node.term.field != "status" || opt.node.term.want != colKindInt {
		t.Fatalf("term = %+v, want field=status kind=int", opt.node.term)
	}
	got := make([]bool, 0, 4)
	for _, v := range []int64{200, 500, 404, 503} {
		got = append(got, opt.node.term.pI64(v))
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
	if opt.node == nil || opt.node.term == nil {
		t.Fatal("Where did not produce a predicate term")
	}
	if opt.node.term.want != colKindString || opt.node.term.pStr == nil {
		t.Fatalf("term = %+v, want kind=string", opt.node.term)
	}
	if !opt.node.term.pStr("ERROR") || opt.node.term.pStr("INFO") {
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

func TestBitsetOps(t *testing.T) {
	n := 130 // spans 3 words, last partial
	a := newBitset(n)
	b := newBitset(n)
	setBit(a, 1)
	setBit(a, 64)
	setBit(b, 64)
	setBit(b, 129)

	or := append([]uint64(nil), a...)
	bitsetOr(or, b)
	for _, i := range []int{1, 64, 129} {
		if !getBit(or, i) {
			t.Fatalf("bitsetOr: bit %d not set", i)
		}
	}

	andNot := append([]uint64(nil), a...)
	bitsetAndNot(andNot, b) // a &^ b: keeps 1, drops 64
	if !getBit(andNot, 1) || getBit(andNot, 64) {
		t.Fatalf("bitsetAndNot wrong: %064b", andNot[0])
	}

	not := notMask(a, n)
	if getBit(not, 1) || !getBit(not, 0) || !getBit(not, 2) {
		t.Fatalf("notMask wrong at low bits")
	}
	// bits >= n must be clear so popcount is meaningful
	if popcount(not) != n-2 {
		t.Fatalf("notMask popcount = %d, want %d", popcount(not), n-2)
	}
}

func TestExplicitAnd_Equivalent(t *testing.T) {
	type Row struct {
		A int32  `qdf:"a"`
		B string `qdf:"b"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i] = Row{A: int32(i), B: "x"}
		if i%2 == 0 {
			rows[i].B = "y"
		}
	}
	buf, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var flat, explicit []Row
	pA := Where("a", func(v int32) bool { return v >= 10 })
	pB := Where("b", func(s string) bool { return s == "y" })
	if err := Unmarshal(buf, &flat, pA, pB); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(buf, &explicit, And(pA, pB)); err != nil {
		t.Fatal(err)
	}
	if len(flat) != len(explicit) || len(flat) == 0 {
		t.Fatalf("And() not equivalent to flat: %d vs %d", len(flat), len(explicit))
	}
}

func TestSelectInCombinator_Unsupported(t *testing.T) {
	type Row struct {
		A int32 `qdf:"a"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].A = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var out []Row
	err := Unmarshal(buf, &out, And(Where("a", func(v int32) bool { return v > 0 }), Select("a")))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
	// Nested same-op: the Select error must survive simplifyCond's flatten.
	err = Unmarshal(buf, &out, And(And(Where("a", func(v int32) bool { return v > 0 }), Select("a")), Where("a", func(v int32) bool { return v < 5 })))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("nested Select-in-combinator: want ErrUnsupported, got %v", err)
	}
}

// TestNotOfSelect_Unsupported pins that a Not wrapping a non-predicate option
// (a Select, which carries no node) returns ErrUnsupported instead of panicking.
// simplifyCond's condNot branch indexed kids[0] unconditionally, so the err-Not
// (built with nil kids) crashed before firstCondErr could surface the error —
// both directly and through the Not(Not(...)) double-negation unwrap.
func TestNotOfSelect_Unsupported(t *testing.T) {
	type Row struct {
		A int32 `qdf:"a"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].A = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var out []Row

	if err := Unmarshal(buf, &out, Not(Select("a"))); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Not(Select): want ErrUnsupported, got %v", err)
	}
	if err := Unmarshal(buf, &out, Not(Not(Select("a")))); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Not(Not(Select)): want ErrUnsupported, got %v", err)
	}
	// Also reachable nested inside a combinator.
	if err := Unmarshal(buf, &out, And(Where("a", func(v int32) bool { return v > 0 }), Not(Select("a")))); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("And(pred, Not(Select)): want ErrUnsupported, got %v", err)
	}
}

func TestEmptyAndNoPanic(t *testing.T) {
	type Row struct {
		A int32 `qdf:"a"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].A = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var out []Row
	// And() with no predicates must not panic; it matches every row (AND identity).
	if err := Unmarshal(buf, &out, And()); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(rows) {
		t.Fatalf("empty And matched %d rows, want %d", len(out), len(rows))
	}
}

func TestEmptyOrNoRows(t *testing.T) {
	type Row struct {
		A int32 `qdf:"a"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].A = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)
	var out []Row
	// Or() with no predicates is the OR identity (FALSE): no row matches.
	if err := Unmarshal(buf, &out, Or()); err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("empty Or matched %d rows, want 0", len(out))
	}
}

func TestSimplifyEquivalence(t *testing.T) {
	type Row struct {
		A int32 `qdf:"a"`
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i].A = int32(i)
	}
	buf, _ := Marshal(rows, OptBalanced|OptColumnIndex)

	gt10 := func() QueryOption { return Where("a", func(v int32) bool { return v > 10 }) }

	// Not(Not(x)) ≡ x ; And(...) single-child unwraps.
	var a, b []Row
	if err := Unmarshal(buf, &a, gt10()); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(buf, &b, And(Not(Not(gt10())))); err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(a) == 0 {
		t.Fatalf("simplify changed result: %d vs %d", len(a), len(b))
	}
}

func TestSimplifyShape(t *testing.T) {
	leaf := func() *condNode { return &condNode{op: condLeaf, term: &predTerm{field: "a"}} }
	// And(And(a, b), c) -> And(a, b, c)
	tree := &condNode{op: condAnd, kids: []*condNode{
		{op: condAnd, kids: []*condNode{leaf(), leaf()}},
		leaf(),
	}}
	got := simplifyCond(tree)
	if got.op != condAnd || len(got.kids) != 3 {
		t.Fatalf("flatten failed: op=%d kids=%d", got.op, len(got.kids))
	}
	// Not(Not(leaf)) -> leaf
	dn := &condNode{op: condNot, kids: []*condNode{{op: condNot, kids: []*condNode{leaf()}}}}
	if simplifyCond(dn).op != condLeaf {
		t.Fatal("double-not not folded")
	}
	// single-child And -> child
	sc := &condNode{op: condAnd, kids: []*condNode{leaf()}}
	if simplifyCond(sc).op != condLeaf {
		t.Fatal("single-child And not unwrapped")
	}
}
