package qdf

import (
	"math/rand"
	"reflect"
	"testing"
)

// --- Task 6: duplicate-key positional fallback ---

func TestKeyedDuplicateKeysFallBackPositional(t *testing.T) {
	type E struct {
		ID  string `qdf:"id,key"`
		Val int
	}
	old := []E{{"a", 1}, {"a", 2}, {"b", 3}} // duplicate key "a"
	neu := []E{{"a", 9}, {"a", 2}, {"b", 3}}
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// dup keys → positional patch, not keyed
	if !containsByte(patch, tagSlicePatch) {
		t.Fatal("duplicate keys must fall back to the positional tagSlicePatch")
	}
	base := append([]E(nil), old...)
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %+v want %+v", base, neu)
	}
}

// --- Task 7: nil / empty / presence + nested keyed slices ---

func TestKeyedNilAndEmpty(t *testing.T) {
	type E struct {
		ID string `qdf:"id,key"`
		V  int
	}
	type Rec struct{ Items []E }
	cases := []struct {
		name     string
		old, neu Rec
	}{
		{"nil->filled", Rec{nil}, Rec{[]E{{"a", 1}}}},
		{"filled->nil", Rec{[]E{{"a", 1}}}, Rec{nil}},
		{"empty->filled", Rec{[]E{}}, Rec{[]E{{"a", 1}}}},
		{"filled->empty", Rec{[]E{{"a", 1}}}, Rec{[]E{}}},
		{"nil->empty", Rec{nil}, Rec{[]E{}}},
		{"empty->nil", Rec{[]E{}}, Rec{nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := Diff(tc.old, tc.neu, OptBalanced)
			if err != nil {
				t.Fatal(err)
			}
			base := Rec{}
			if tc.old.Items != nil {
				// Faithfully mirror old's nilness: make() keeps an empty slice
				// non-nil (append-from-nil would collapse empty->nil and trip baseFP).
				base.Items = make([]E, len(tc.old.Items))
				copy(base.Items, tc.old.Items)
			}
			if err := Apply(&base, patch); err != nil {
				t.Fatal(err)
			}
			if (base.Items == nil) != (tc.neu.Items == nil) {
				t.Fatalf("nilness wrong: got nil=%v want nil=%v", base.Items == nil, tc.neu.Items == nil)
			}
			if !reflect.DeepEqual(base, tc.neu) {
				t.Fatalf("got %+v want %+v", base, tc.neu)
			}
		})
	}
}

func TestKeyedNested(t *testing.T) {
	type Inner struct {
		ID  string `qdf:"id,key"`
		Sub []int
	}
	type Outer struct {
		Rows []Inner
		Map  map[string][]Inner
	}
	old := Outer{
		Rows: []Inner{{"a", []int{1}}, {"b", []int{2}}},
		Map:  map[string][]Inner{"g": {{"x", []int{9}}}},
	}
	neu := Outer{
		Rows: []Inner{{"b", []int{2}}, {"a", []int{1, 1}}}, // reorder + change a.Sub
		Map:  map[string][]Inner{"g": {{"y", []int{8}}, {"x", []int{9}}}},
	}
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	base := Outer{
		Rows: []Inner{{"a", []int{1}}, {"b", []int{2}}},
		Map:  map[string][]Inner{"g": {{"x", []int{9}}}},
	}
	if err := Apply(&base, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, neu) {
		t.Fatalf("got %+v want %+v", base, neu)
	}
}

// --- Task 8: never-larger on reorder + generative oracle + hostile fuzz ---

func TestKeyedReorderBeatsPositional(t *testing.T) {
	type E struct {
		ID  int64 `qdf:"id,key"`
		Big string
	}
	n := 200
	old := make([]E, n)
	for i := range old {
		old[i] = E{ID: int64(i), Big: "payload-" + string(rune('a'+i%26))}
	}
	neu := make([]E, n)
	for i := range neu {
		neu[i] = old[n-1-i] // reverse reorder, no value change
	}
	keyed, _ := Diff(old, neu, OptBalanced)
	full, _ := Marshal(neu, OptBalanced)
	if len(keyed) >= len(full) {
		t.Fatalf("keyed reorder patch (%d) must be below a full marshal (%d)", len(keyed), len(full))
	}
	t.Logf("reverse-reorder n=%d: keyed=%d bytes, full=%d bytes", n, len(keyed), len(full))
}

type kEnt struct {
	ID   int64 `qdf:"id,key"`
	A    string
	B    []int64
	Flag bool
}

func randKeyed(r *rand.Rand) []kEnt {
	n := r.Intn(8)
	seen := map[int64]bool{}
	var out []kEnt
	for range n {
		id := int64(r.Intn(10))
		if seen[id] {
			continue // unique keys → exercise the keyed path, not the fallback
		}
		seen[id] = true
		var b []int64
		switch r.Intn(3) {
		case 1:
			b = []int64{}
		case 2:
			b = make([]int64, r.Intn(3))
			for i := range b {
				b[i] = int64(r.Intn(100))
			}
		}
		out = append(out, kEnt{ID: id, A: []string{"x", "y", ""}[r.Intn(3)], B: b, Flag: r.Intn(2) == 0})
	}
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func cloneKeyed(s []kEnt) []kEnt {
	if s == nil {
		return nil
	}
	c := make([]kEnt, len(s))
	copy(c, s)
	for i := range c {
		if s[i].B != nil { // preserve empty-non-nil (append-from-nil would collapse it)
			c[i].B = make([]int64, len(s[i].B))
			copy(c[i].B, s[i].B)
		}
	}
	return c
}

func FuzzKeyedSliceOracle(f *testing.F) {
	f.Add(int64(1), int64(2))
	f.Add(int64(5), int64(5))
	f.Add(int64(7), int64(99))
	f.Fuzz(func(t *testing.T, so, sn int64) {
		old := randKeyed(rand.New(rand.NewSource(so)))
		neu := randKeyed(rand.New(rand.NewSource(sn)))
		for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
			patch, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatalf("opts=%v Diff: %v", opts, err)
			}
			base := cloneKeyed(old)
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("opts=%v Apply: %v", opts, err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("opts=%v mismatch\n old=%+v\n new=%+v\n got=%+v", opts, old, neu, base)
			}
		}
	})
}

func FuzzApplyKeyedHostile(f *testing.F) {
	good, _ := Diff([]kEnt{{ID: 1}}, []kEnt{{ID: 1, A: "x"}, {ID: 2}}, OptBalanced)
	f.Add(good)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, patch []byte) {
		var base []kEnt
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on hostile keyed patch: %v", r)
			}
		}()
		_ = Apply(&base, patch) // error or nil, never panic/OOM
	})
}

// TestKeyedNestedReentrancyDoesNotClobberMap guards a re-entrancy bug: a keyed
// slice whose elements themselves contain a keyed slice, both with > keyedLinearMax
// elements so both use the shared enc.keyIdx / dec.keyIdx scratch map. A nested
// keyed-slice diff/apply must route to its own map instead of clearing the
// parent frame's lookup mid-iteration (which corrupted the patch on encode and
// returned spurious ErrInvalidPatch on apply).
func TestKeyedNestedReentrancyDoesNotClobberMap(t *testing.T) {
	type Inner struct {
		K string `qdf:"k,key"`
		V int64
	}
	type Outer struct {
		OK    string `qdf:"ok,key"`
		Inner []Inner
	}
	const N = 40 // > keyedLinearMax (32) at both levels → both use the map
	mk := func() []Outer {
		out := make([]Outer, N)
		for i := range N {
			inn := make([]Inner, N)
			for j := range N {
				inn[j] = Inner{
					K: string(rune('A'+j%26)) + string(rune('0'+j/26)) + "_" + string(rune('a'+i%26)),
					V: int64(i*1000 + j),
				}
			}
			out[i] = Outer{OK: "outer_" + string(rune('A'+i%26)) + string(rune('0'+i/26)), Inner: inn}
		}
		return out
	}
	old := mk()
	neu := mk()
	// Change inner values in two different outer elements, identical order at both
	// levels (value-only path) so recursion happens between two parent-frame ops.
	neu[2].Inner[10].V = 999999
	neu[5].Inner[20].V = 888888

	for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
		patch, err := Diff(old, neu, opts)
		if err != nil {
			t.Fatalf("opts=%v Diff: %v", opts, err)
		}
		base := mk()
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opts=%v Apply: %v", opts, err)
		}
		if !reflect.DeepEqual(base, neu) {
			t.Fatalf("opts=%v: nested keyed re-entrancy corrupted round-trip", opts)
		}
	}
}
