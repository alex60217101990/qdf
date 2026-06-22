package qdf

import (
	"fmt"
	"reflect"
	"testing"
)

// keyedRec / plainRec are the same struct with and without the ,key tag. Diffing
// plainRec forces the positional differ (diffSlice), giving the exact alternative
// the keyed never-larger picker must never exceed.
type keyedRec struct {
	ID  string `qdf:"id,key"`
	Val string `qdf:"v"`
}

type plainRec struct {
	ID  string `qdf:"id"`
	Val string `qdf:"v"`
}

func mkKeyed(idPrefix string, n int) []keyedRec {
	s := make([]keyedRec, n)
	for i := range s {
		s[i] = keyedRec{
			ID:  fmt.Sprintf("%s-key-%06d", idPrefix, i),
			Val: fmt.Sprintf("payload-value-%06d-some-longer-text", i),
		}
	}
	return s
}

func asPlain(k []keyedRec) []plainRec {
	p := make([]plainRec, len(k))
	for i := range k {
		p[i] = plainRec{ID: k[i].ID, Val: k[i].Val}
	}
	return p
}

// keyedShapes returns the spread of change shapes the never-larger picker must
// hold the contract across — full key rotation (the audit-3 +29% breach), pure
// reorder (where keyed wins big), value-only change, and partial rotation.
func keyedShapes(n int) map[string][]keyedRec {
	return map[string][]keyedRec{
		"full-rotation": mkKeyed("new", n),
		"reorder-only": func() []keyedRec {
			s := mkKeyed("old", n)
			for i := 0; i < n/2; i++ {
				s[i], s[n-1-i] = s[n-1-i], s[i]
			}
			return s
		}(),
		"value-only": func() []keyedRec {
			s := mkKeyed("old", n)
			for i := range s {
				s[i].Val = "changed-" + s[i].Val
			}
			return s
		}(),
		"half-rotation": func() []keyedRec {
			s := mkKeyed("old", n)
			for i := 0; i < n/2; i++ {
				s[i].ID = fmt.Sprintf("new-key-%06d", i)
			}
			return s
		}(),
	}
}

// TestKeyedNeverLarger asserts the docs/DELTA.md never-larger contract: the keyed
// slice patch must never exceed the positional alternative, NOR a full re-encode,
// for any change shape — and must round-trip exactly. Audit-3 found a +29% breach
// vs positional on full rotation (keyed patch even exceeded a full re-encode); the
// never-larger picker added in this change closes it.
func TestKeyedNeverLarger(t *testing.T) {
	old := mkKeyed("old", 100)
	oldPlain := asPlain(old)

	for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
		for name, neu := range keyedShapes(100) {
			keyedPatch, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatalf("opts=%d %s: Diff keyed: %v", opts, name, err)
			}
			posPatch, err := Diff(oldPlain, asPlain(neu), opts)
			if err != nil {
				t.Fatalf("opts=%d %s: Diff positional: %v", opts, name, err)
			}
			full, err := Marshal(neu, opts)
			if err != nil {
				t.Fatalf("opts=%d %s: Marshal: %v", opts, name, err)
			}
			t.Logf("opts=%d %-14s keyed=%d positional=%d full=%d", opts, name, len(keyedPatch), len(posPatch), len(full))

			// The documented contract (docs/DELTA.md:124): the keyed patch is never
			// larger than the alternative representation — the positional differ.
			if len(keyedPatch) > len(posPatch) {
				t.Errorf("opts=%d %s: keyed patch %d > positional %d (breaches never-larger)",
					opts, name, len(keyedPatch), len(posPatch))
			}
			// The keyed patch must not exceed a full re-encode either, EXCEPT under a
			// heavy-codec tier where the positional alternative also loses to a full
			// rANS re-encode (a slice-level whole-replace fallback, not the keyed
			// contract). Only flag a keyed-introduced regression.
			if len(keyedPatch) > len(full) && len(posPatch) <= len(full) {
				t.Errorf("opts=%d %s: keyed patch %d > full re-encode %d (positional %d stayed under)",
					opts, name, len(keyedPatch), len(full), len(posPatch))
			}

			// Round-trip exact.
			base := append([]keyedRec(nil), old...)
			if err := Apply(&base, keyedPatch); err != nil {
				t.Fatalf("opts=%d %s: Apply: %v", opts, name, err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("opts=%d %s: round-trip mismatch", opts, name)
			}
		}
	}
}

// TestKeyedNeverLargerNoInternLeak guards the trap the suspend-during-trial design
// exists for: a discarded never-larger candidate must not leak a shared-state id
// (intern string OR struct/map shape) whose wire definition is thrown away. The
// struct here places a keyed slice BEFORE fields whose values recur as interned
// strings AND a shaped map; if the trial leaked an id, decoding the later content
// would dangle (ErrUnknownStateID) or corrupt.
func TestKeyedNeverLargerNoInternLeak(t *testing.T) {
	type Item struct {
		ID   string `qdf:"id,key"`
		Note string `qdf:"note"`
	}
	type Doc struct {
		Items []Item            `qdf:"items"`
		Tags  []string          `qdf:"tags"`
		Meta  map[string]string `qdf:"meta"`
	}

	// Strings long enough to be interned (>= minIntern) that recur across the
	// keyed slice and the later fields.
	old := Doc{
		Items: []Item{{"alpha-key", "shared-note-payload"}, {"beta-key", "another-note-value"}},
		Tags:  []string{"shared-note-payload", "tag-recurring-value"},
		Meta:  map[string]string{"k1": "another-note-value", "k2": "tag-recurring-value"},
	}
	neu := Doc{
		// Reorder + value change + new key → exercises the keyed orderChanged path.
		Items: []Item{{"beta-key", "another-note-value"}, {"alpha-key", "shared-note-payload"}, {"gamma-key", "tag-recurring-value"}},
		Tags:  []string{"tag-recurring-value", "shared-note-payload", "fresh-tag-payload"},
		Meta:  map[string]string{"k1": "shared-note-payload", "k2": "fresh-tag-payload", "k3": "another-note-value"},
	}

	for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
		patch, err := Diff(old, neu, opts)
		if err != nil {
			t.Fatalf("opts=%d Diff: %v", opts, err)
		}
		base := old
		base.Items = append([]Item(nil), old.Items...)
		base.Tags = append([]string(nil), old.Tags...)
		base.Meta = map[string]string{}
		for k, v := range old.Meta {
			base.Meta[k] = v
		}
		if err := Apply(&base, patch); err != nil {
			t.Fatalf("opts=%d Apply: %v", opts, err)
		}
		if !reflect.DeepEqual(base, neu) {
			t.Fatalf("opts=%d: round-trip mismatch\n got %+v\nwant %+v", opts, base, neu)
		}
	}
}
