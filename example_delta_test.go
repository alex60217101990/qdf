package qdf_test

import (
	"fmt"
	"reflect"

	"github.com/alex60217101990/qdf"
)

// ExampleDiff shows the structural delta: Diff computes a patch carrying only
// the locations that changed, and Apply merges it back onto a base in place. The
// patch is far smaller than a full re-encode because unchanged fields cost no
// bytes, and it is self-describing — the receiver hands the bytes straight to
// Apply and never has to know which Options produced them.
func ExampleDiff() {
	type Config struct {
		Replicas int               `qdf:"replicas"`
		Image    string            `qdf:"image"`
		Env      map[string]string `qdf:"env"`
	}

	old := Config{Replicas: 3, Image: "app:v1", Env: map[string]string{"LOG": "info", "REGION": "eu"}}
	// Only Replicas and one Env key change.
	updated := Config{Replicas: 5, Image: "app:v1", Env: map[string]string{"LOG": "debug", "REGION": "eu"}}

	patch, _ := qdf.Diff(old, updated, qdf.OptBalanced)
	full, _ := qdf.Marshal(updated, qdf.OptBalanced)

	base := old
	_ = qdf.Apply(&base, patch)

	fmt.Println("applied == updated:", reflect.DeepEqual(base, updated))
	fmt.Println("patch smaller than full encode:", len(patch) < len(full))

	// Output:
	// applied == updated: true
	// patch smaller than full encode: true
}

// ExampleDiff_keyedSlice shows keyed-slice matching: tag an element's stable
// identity field with ",key" and a []struct patch matches elements by that key
// instead of by position. Reordering the slice then ships only the new key
// order (no element values), where a positional diff would reship the whole
// shifted tail.
func ExampleDiff_keyedSlice() {
	type Entity struct {
		ID string  `qdf:"id,key"`
		X  float64 `qdf:"x"`
	}

	old := []Entity{{"a", 1}, {"b", 2}, {"c", 3}}
	// Same entities, reversed order, with one value edit.
	updated := []Entity{{"c", 3}, {"b", 20}, {"a", 1}}

	patch, _ := qdf.Diff(old, updated, qdf.OptBalanced)

	base := append([]Entity(nil), old...)
	_ = qdf.Apply(&base, patch)

	fmt.Println("reordered roundtrip:", reflect.DeepEqual(base, updated))

	// Output:
	// reordered roundtrip: true
}

// ExampleBaselineRegistry shows applying a chain of patches in a state-sync
// stream without threading the previous value by hand. The registry resolves
// each patch's baseline from the baseFP that Diff already embeds (no wire
// change), holds baselines through weak pointers so the GC reclaims anything the
// caller drops, and auto-registers each result so it can base the next patch.
func ExampleBaselineRegistry() {
	type State struct {
		Seq  int    `qdf:"seq"`
		Note string `qdf:"note"`
	}

	s0 := &State{Seq: 1, Note: "init"}
	s1 := State{Seq: 2, Note: "init"}
	s2 := State{Seq: 3, Note: "done"}

	// Producer ships ordinary diffs along the chain.
	patch1, _ := qdf.Diff(*s0, s1, qdf.OptBalanced)
	patch2, _ := qdf.Diff(s1, s2, qdf.OptBalanced)

	// Consumer: bootstrap from a full value, then apply the chain.
	reg := qdf.NewBaselineRegistry[State]()
	reg.Register(s0)

	got1, _ := reg.Apply(patch1) // resolves s0 by baseFP → s1
	got2, _ := reg.Apply(patch2) // resolves got1 (== s1) → s2
	_ = got1                     // keep the previous result reachable for the next step

	fmt.Printf("got2 = {Seq:%d Note:%s}\n", got2.Seq, got2.Note)

	// Output:
	// got2 = {Seq:3 Note:done}
}
