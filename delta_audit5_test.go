package qdf

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

// TestKeyedMapShapeNoLeak is the audit-5 RED repro for the residual of the
// audit-4 suspension fix: mapStringShapeOrder (the OptMapShape declaring site)
// was the one shape site left ungated on stateSuspended. During diffKeyedSlice's
// suspended never-larger trial, a re-encoded element's map[string]V field
// declares/binds a key-set shape on enc.state; the keyed-win re-base fixes only
// shapeCount, never the map-shape binding, so a later sibling map that reuses the
// key-set emits a tagMapShape reference whose declaration lived only in the
// discarded/trial bytes — Apply of a valid Diff fails with ErrUnknownStateID.
func TestKeyedMapShapeNoLeak(t *testing.T) {
	type msElem struct {
		K string            `qdf:"id,key"`
		M map[string]string `qdf:"m"`
	}
	type msHolder struct {
		Items []msElem          `qdf:"items"`
		Tail  map[string]string `qdf:"tail"`
	}

	const base = 60
	key := func(i int) string { return string(rune('a'+i%26)) + string(rune('0'+i/26)) }

	mkOld := func() msHolder {
		items := make([]msElem, base)
		for i := range base {
			items[i] = msElem{key(i), map[string]string{"a": "x", "b": "y"}}
		}
		return msHolder{Items: items, Tail: map[string]string{"p": "0", "q": "2", "r": "3"}}
	}
	mkNew := func() msHolder {
		// rotate by one, then append 5 new-key elements whose M uses the {p,q,r}
		// key-set that Tail also uses.
		items := make([]msElem, base, base+5)
		for i := range base {
			items[i] = msElem{key((i + 1) % base), map[string]string{"a": "x", "b": "y"}}
		}
		for i := range 5 {
			items = append(items, msElem{"NEW" + string(rune('0'+i)),
				map[string]string{"p": "1", "q": "2", "r": "3"}})
		}
		return msHolder{Items: items, Tail: map[string]string{"p": "9", "q": "2", "r": "3"}}
	}

	for _, opt := range []Options{OptMapShape | OptDense, OptBalanced | OptMapShape, OptCompression | OptMapShape} {
		old, nw := mkOld(), mkNew()
		patch, err := Diff(old, nw, opt)
		if err != nil {
			t.Fatalf("opt %v: Diff: %v", opt, err)
		}
		base := msHolder{Items: slices.Clone(old.Items), Tail: maps.Clone(old.Tail)}
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opt %v: Apply rejected a valid patch: %v", opt, err)
		}
		if !reflect.DeepEqual(base, nw) {
			t.Fatalf("opt %v: round-trip mismatch", opt)
		}
	}
}

// TestColWinLastIDNoPairCorruption is the audit-5 RED probe for the columnar
// column-win lastID divergence: after a column body wins the never-larger trial,
// the encoder left lastID at the (suspended) positional-build value while the
// decoder, reading the wire-stateless column body, keeps its pre-slice lastID.
// The two then disagree on the Markov-1 pair predictor, which under OptBalanced
// can make a later interned value resolve to the WRONG string with no error.
// Randomized over many orderings to hit the specific pre/post sequence.
func TestColWinLastIDNoPairCorruption(t *testing.T) {
	type colRow struct {
		N int32  `qdf:"n"`
		S string `qdf:"s"`
	}
	type doc struct {
		Pre  []string `qdf:"pre"`
		Cols []colRow `qdf:"cols"`
		Post []string `qdf:"post"`
	}
	alpha := []string{"red", "green", "blue", "amber", "violet", "cyan", "gray"}
	rng := func(seed uint64) func(int) int {
		s := seed
		return func(n int) int { s = s*6364136223846793005 + 1442695040888963407; return int((s >> 33) % uint64(n)) }
	}
	mkStrs := func(r func(int) int, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = alpha[r(len(alpha))]
		}
		return out
	}
	mkCols := func(r func(int) int, n int) []colRow {
		out := make([]colRow, n)
		for i := range out {
			out[i] = colRow{int32(r(1000)), alpha[r(len(alpha))]}
		}
		return out
	}
	for seed := uint64(1); seed <= 4000; seed++ {
		r := rng(seed)
		old := doc{Pre: mkStrs(r, 12), Cols: mkCols(r, 20), Post: mkStrs(r, 12)}
		r2 := rng(seed * 2654435761)
		nw := doc{Pre: mkStrs(r2, 12), Cols: mkCols(r2, 20), Post: mkStrs(r2, 12)}
		patch, err := Diff(old, nw, OptBalanced)
		if err != nil {
			t.Fatalf("seed %d: Diff: %v", seed, err)
		}
		base := old
		base.Pre = slices.Clone(old.Pre)
		base.Cols = slices.Clone(old.Cols)
		base.Post = slices.Clone(old.Post)
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("seed %d: Apply: %v", seed, err)
		}
		if !reflect.DeepEqual(base, nw) {
			t.Fatalf("seed %d: round-trip mismatch (lastID/pair desync)\n got %+v\nwant %+v", seed, base, nw)
		}
	}
}
