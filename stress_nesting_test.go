package qdf

import (
	"reflect"
	"runtime/debug"
	"testing"
)

// Pathological-shape stress tests. The reflect codepath walks the
// type recursively for both encode and decode; a deep enough payload
// must not crash, OOM, or stack-overflow.

// 100-deep linked list of pointers. The reflect path traverses the
// chain once on encode and again on decode.
type deepNode struct {
	V    int       `qdf:"v"`
	Next *deepNode `qdf:"next"`
}

func makeDeepNodes(n int) *deepNode {
	if n <= 0 {
		return nil
	}
	head := &deepNode{V: 0}
	cur := head
	for i := 1; i < n; i++ {
		cur.Next = &deepNode{V: i}
		cur = cur.Next
	}
	return head
}

func nodeLen(n *deepNode) int {
	c := 0
	for n != nil {
		c++
		n = n.Next
	}
	return c
}

func TestStress_DeepLinkedList(t *testing.T) {
	for _, depth := range []int{16, 64, 128, 256} {
		t.Run("", func(t *testing.T) {
			in := makeDeepNodes(depth)
			for label, opts := range map[string]Options{"Speed": OptSpeed, "QPack": OptQPack, "Balanced": OptBalanced} {
				buf, err := Marshal(in, opts)
				if err != nil {
					t.Fatalf("%s depth=%d encode: %v", label, depth, err)
				}
				var out *deepNode
				if err := Unmarshal(buf, &out); err != nil {
					t.Fatalf("%s depth=%d decode: %v", label, depth, err)
				}
				if got := nodeLen(out); got != depth {
					t.Fatalf("%s depth=%d: got len %d", label, depth, got)
				}
			}
		})
	}
}

// Wide flat struct. Tests the descriptor cache / field-name dispatch
// on a struct with many fields. Built dynamically via map[string]any
// since Go does not allow declaring 100-field types ergonomically.
func TestStress_WideMap(t *testing.T) {
	for _, width := range []int{32, 100, 512} {
		t.Run("", func(t *testing.T) {
			in := make(map[string]int, width)
			for i := range width {
				k := []byte("k0000")
				k[1] = '0' + byte((i/1000)%10)
				k[2] = '0' + byte((i/100)%10)
				k[3] = '0' + byte((i/10)%10)
				k[4] = '0' + byte(i%10)
				in[string(k)] = i
			}
			for label, opts := range map[string]Options{"Speed": OptSpeed, "Balanced": OptBalanced} {
				buf, err := Marshal(in, opts)
				if err != nil {
					t.Fatalf("%s width=%d encode: %v", label, width, err)
				}
				out := make(map[string]int, width)
				if err := Unmarshal(buf, &out); err != nil {
					t.Fatalf("%s width=%d decode: %v", label, width, err)
				}
				if !reflect.DeepEqual(in, out) {
					t.Fatalf("%s width=%d mismatch", label, width)
				}
			}
		})
	}
}

// Balanced binary tree of structs. Tests both recursion depth and
// branching factor.
type treeNode struct {
	V int       `qdf:"v"`
	L *treeNode `qdf:"l"`
	R *treeNode `qdf:"r"`
}

func makeTree(depth int) *treeNode {
	if depth <= 0 {
		return nil
	}
	return &treeNode{V: depth, L: makeTree(depth - 1), R: makeTree(depth - 1)}
}

func compareTrees(a, b *treeNode) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.V == b.V && compareTrees(a.L, b.L) && compareTrees(a.R, b.R)
}

func TestStress_BinaryTree(t *testing.T) {
	for _, depth := range []int{6, 10, 12} {
		t.Run("", func(t *testing.T) {
			in := makeTree(depth)
			for label, opts := range map[string]Options{"Speed": OptSpeed, "Balanced": OptBalanced} {
				buf, err := Marshal(in, opts)
				if err != nil {
					t.Fatalf("%s depth=%d: %v", label, depth, err)
				}
				var out *treeNode
				if err := Unmarshal(buf, &out); err != nil {
					t.Fatalf("%s depth=%d decode: %v", label, depth, err)
				}
				if !compareTrees(in, out) {
					t.Fatalf("%s depth=%d tree mismatch", label, depth)
				}
			}
		})
	}
}

// Big primitive payloads: 1 MiB string, 1 MiB byte slice, 100k-element
// numeric slices. Catches edge cases in length-prefix selection
// (str8/str16/str32, bin8/bin16/bin32, arr16/arr32) and any per-
// element overhead that scales worse than linearly.
func TestStress_BigPrimitives(t *testing.T) {
	long := make([]byte, 1<<20) // 1 MiB
	for i := range long {
		long[i] = byte(i)
	}
	longString := string(long)

	bigU64 := make([]uint64, 100_000)
	for i := range bigU64 {
		bigU64[i] = uint64(i)
	}
	bigF64 := make([]float64, 100_000)
	for i := range bigF64 {
		bigF64[i] = float64(i) * 0.5
	}

	type bigStruct struct {
		S string    `qdf:"s"`
		B []byte    `qdf:"b"`
		U []uint64  `qdf:"u"`
		F []float64 `qdf:"f"`
	}
	in := bigStruct{S: longString, B: long, U: bigU64, F: bigF64}

	for label, opts := range map[string]Options{"Speed": OptSpeed, "QPack": OptQPack, "Balanced": OptBalanced} {
		buf, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("%s encode: %v", label, err)
		}
		var out bigStruct
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("%s decode: %v", label, err)
		}
		if out.S != in.S {
			t.Fatalf("%s: string mismatch", label)
		}
		if !reflect.DeepEqual(out.B, in.B) {
			t.Fatalf("%s: bytes mismatch", label)
		}
		if !reflect.DeepEqual(out.U, in.U) {
			t.Fatalf("%s: u64 mismatch", label)
		}
		if !reflect.DeepEqual(out.F, in.F) {
			t.Fatalf("%s: f64 mismatch", label)
		}
	}
}

// Make sure the encoder does not exceed Go's default 1 GB max stack on
// any of the recursion-heavy targets above. SetMaxStack is best-effort
// — Go's stack is segmented, but a runaway recursion still hits this
// guard.
func TestStress_StackBound(t *testing.T) {
	prev := debug.SetMaxStack(64 << 20) // 64 MiB
	defer debug.SetMaxStack(prev)
	in := makeDeepNodes(2000)
	buf, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out *deepNode
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if got := nodeLen(out); got != 2000 {
		t.Fatalf("got %d", got)
	}
}
