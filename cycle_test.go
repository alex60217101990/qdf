package qdf

import (
	"runtime/debug"
	"testing"
)

// Cycle detection. A pointer cycle through value-typed fields (e.g.
// `type T struct { Next T }`) is rejected at descriptor-build time so
// the encoder cannot recurse forever. A cycle through pointer-typed
// fields is structurally allowed by Go's type system but the
// encoder must not blow the stack at runtime — the decoder rebuilds
// the cycle on demand only if the encoder emitted it, and the test
// suite captures the current behaviour either way (encode-error or
// encode-success-without-OOM).

// cyclicPtr is the legal form: two structs that point at each other
// via pointers. The encoder walks until it hits a nil pointer.
type cyclicPtr struct {
	V    int        `qdf:"v"`
	Next *cyclicPtr `qdf:"next"`
}

// Pointer cycle detection: the encoder bumps a depth counter on each
// pointer dereference and returns ErrCycleDetected once depth exceeds
// the encoder's maxDepth (default 10000). Cheaper than a per-pointer
// set and catches both genuine *T->*T cycles and pathologically deep
// payloads.
func TestCycle_PointerCycleDetected(t *testing.T) {
	a := &cyclicPtr{V: 1}
	b := &cyclicPtr{V: 2}
	a.Next = b
	b.Next = a // genuine cycle

	_, err := Marshal(a)
	if err == nil {
		t.Fatal("expected ErrCycleDetected on pointer cycle")
	}
	if err.Error() != ErrCycleDetected.Error() {
		t.Fatalf("got %v, want %v", err, ErrCycleDetected)
	}
}

// Value-typed cycle through a pointer chain that is finite in length
// but very deep. Must succeed.
func TestCycle_DeepFinitePointerChain(t *testing.T) {
	prev := debug.SetMaxStack(128 << 20)
	defer debug.SetMaxStack(prev)
	const depth = 5000
	head := &cyclicPtr{V: 0}
	cur := head
	for i := 1; i < depth; i++ {
		cur.Next = &cyclicPtr{V: i}
		cur = cur.Next
	}
	buf, err := Marshal(head)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out *cyclicPtr
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := 0
	for n := out; n != nil; n = n.Next {
		got++
	}
	if got != depth {
		t.Fatalf("depth=%d got=%d", depth, got)
	}
}

// Value-typed cycle through a non-pointer field (e.g. `type T struct { Next T }`)
// must be rejected at descriptor-build time. This is a compile-time
// type error in Go actually (a struct cannot contain a value-typed
// instance of itself), so the test below uses the codec's runtime
// cycle handling through a slice instead: `type T struct { Children []T }`
// is legal at the type level but the recursive descriptor build must
// terminate.
type recurStruct struct {
	V        int           `qdf:"v"`
	Children []recurStruct `qdf:"children"`
}

func TestCycle_RecursiveStructLegal(t *testing.T) {
	in := recurStruct{
		V: 1,
		Children: []recurStruct{
			{V: 2, Children: []recurStruct{
				{V: 3, Children: []recurStruct{}},
			}},
			{V: 4, Children: []recurStruct{}},
		},
	}
	buf, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out recurStruct
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.V != 1 || len(out.Children) != 2 {
		t.Fatalf("top: %+v", out)
	}
	if out.Children[0].V != 2 || out.Children[0].Children[0].V != 3 {
		t.Fatalf("nested: %+v", out)
	}
}
