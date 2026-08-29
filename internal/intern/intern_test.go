package intern

import (
	"strconv"
	"testing"
	"unsafe"
)

func TestCache_Hit(t *testing.T) {
	var c Cache
	a := c.Make([]byte("hello"))
	b := c.Make([]byte("hello"))
	if a != b {
		t.Fatalf("string equality mismatch: %q vs %q", a, b)
	}
	// Same string data pointer = same allocation reused.
	if pointerOf(a) != pointerOf(b) {
		t.Fatal("expected pointer reuse on cache hit")
	}
}

func TestCache_Miss_CopiesBytes(t *testing.T) {
	var c Cache
	src := []byte("mutate me")
	s := c.Make(src)
	// Mutate the source — the cached string must not change (defensive copy).
	src[0] = 'X'
	if s != "mutate me" {
		t.Fatalf("cache string was mutated: %q", s)
	}
}

func TestCache_DistinctKeys(t *testing.T) {
	var c Cache
	got := map[string]bool{}
	for i := range 100 {
		s := c.Make([]byte("k" + strconv.Itoa(i)))
		got[s] = true
	}
	if len(got) != 100 {
		t.Fatalf("expected 100 distinct strings, got %d", len(got))
	}
}

func TestCache_CollisionOverwrites(t *testing.T) {
	// We can't easily force a collision with a randomized seed; instead
	// just sanity-check that the API behaves correctly across many distinct
	// inputs. Functional correctness is preserved regardless of collision.
	var c Cache
	for i := range 10_000 {
		k := []byte("collision-test-" + strconv.Itoa(i))
		s := c.Make(k)
		if s != string(k) {
			t.Fatalf("string mismatch at iter %d: %q vs %q", i, s, string(k))
		}
	}
}

func pointerOf(s string) uintptr {
	return uintptr(unsafe.Pointer(unsafe.StringData(s)))
}
