package qdf

import (
	"reflect"
	"strconv"
	"testing"
)

// UnmarshalKeys projects a dynamic-keyed payload the way a typed subset struct
// projects a static one: wanted values decode, everything else is skipped.
func TestUnmarshalKeys(t *testing.T) {
	m := map[string][]uint16{
		"a": mkWeights16(2048, 1, bf16Bits),
		"b": mkWeights16(2048, 2, bf16Bits),
		"c": mkWeights16(2048, 3, bf16Bits),
		"d": mkWeights16(2048, 4, bf16Bits),
	}
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression, OptBalanced | OptMapShape} {
		blob, err := Marshal(m, opts)
		if err != nil {
			t.Fatalf("%v encode: %v", opts, err)
		}
		var got map[string][]uint16
		if err := UnmarshalKeys(blob, &got, "b", "d"); err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		if len(got) != 2 {
			t.Fatalf("%v: got %d entries, want 2", opts, len(got))
		}
		if !reflect.DeepEqual(got["b"], m["b"]) || !reflect.DeepEqual(got["d"], m["d"]) {
			t.Fatalf("%v: projected values wrong", opts)
		}
		if _, ok := got["a"]; ok {
			t.Fatalf("%v: unwanted key present", opts)
		}
		// No keys behaves like Unmarshal.
		var all map[string][]uint16
		if err := UnmarshalKeys(blob, &all); err != nil {
			t.Fatalf("%v all: %v", opts, err)
		}
		if len(all) != len(m) {
			t.Fatalf("%v: empty filter dropped keys (%d of %d)", opts, len(all), len(m))
		}
		// Keys absent from the payload yield an empty result, not an error.
		var none map[string][]uint16
		if err := UnmarshalKeys(blob, &none, "nope"); err != nil {
			t.Fatalf("%v missing: %v", opts, err)
		}
		if len(none) != 0 {
			t.Fatalf("%v: %d entries for an absent key", opts, len(none))
		}
	}
}

// TestUnmarshalKeysNested pins the scoping rule: the filter applies to the
// OUTERMOST map only. Without that, a nested map inside a selected value
// silently loses every key not in the caller's list.
func TestUnmarshalKeysNested(t *testing.T) {
	m := map[string]map[string]int64{
		"keep": {"a": 1, "b": 2, "c": 3},
		"drop": {"a": 9, "b": 9, "c": 9},
	}
	for _, opts := range []Options{OptSpeed, OptBalanced, OptBalanced | OptMapShape} {
		blob, err := Marshal(m, opts)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]map[string]int64
		if err := UnmarshalKeys(blob, &got, "keep"); err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		if len(got) != 1 {
			t.Fatalf("%v: outer filter kept %d entries", opts, len(got))
		}
		if !reflect.DeepEqual(got["keep"], m["keep"]) {
			t.Fatalf("%v: nested map lost keys: %v", opts, got["keep"])
		}
	}
}

// TestUnmarshalKeysValueShapes: skipping must handle every value shape, not
// just the packed slices the checkpoint case uses.
func TestUnmarshalKeysValueShapes(t *testing.T) {
	type inner struct {
		S string
		N []int64
	}
	m := map[string]any{
		"str":    "hello",
		"num":    int64(42),
		"slice":  []int64{1, 2, 3},
		"nested": map[string]any{"x": int64(1)},
		"struct": inner{S: "s", N: []int64{7, 8}},
		"want":   int64(99),
	}
	for _, opts := range []Options{OptSpeed, OptBalanced} {
		blob, err := Marshal(m, opts)
		if err != nil {
			t.Fatalf("%v encode: %v", opts, err)
		}
		var got map[string]any
		if err := UnmarshalKeys(blob, &got, "want"); err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		if len(got) != 1 {
			t.Fatalf("%v: got %v", opts, got)
		}
		// decodeAny picks the narrowest integer kind that fits, so compare
		// numerically rather than against one concrete type.
		v := reflect.ValueOf(got["want"])
		num := int64(-1)
		switch {
		case v.CanInt():
			num = v.Int()
		case v.CanUint():
			num = int64(v.Uint())
		}
		if num != 99 {
			t.Fatalf("%v: want=%#v", opts, got["want"])
		}
	}
}

// TestUnmarshalKeysRejectsNonMapRoots: the projection is defined on the root
// map. Targets it cannot project must be rejected outright — silently ignoring
// the keys, or letting the filter drift down to some map nested at an
// arbitrary depth, are both worse than an error.
func TestUnmarshalKeysRejectsNonMapRoots(t *testing.T) {
	type row struct {
		Tags map[string]string
	}
	blobMap, err := Marshal(map[int64]string{1: "a", 2: "b"}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	blobRows, err := Marshal([]row{{Tags: map[string]string{"env": "prod"}}}, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var intKeyed map[int64]string
	if err := UnmarshalKeys(blobMap, &intKeyed, "1"); err == nil {
		t.Fatal("non-string-keyed root accepted")
	}
	var rows []row
	if err := UnmarshalKeys(blobRows, &rows, "env"); err == nil {
		t.Fatal("slice root accepted; the filter would have hit a nested map")
	}
	// Empty key list is still a plain Unmarshal, whatever the root is.
	rows = nil
	if err := UnmarshalKeys(blobRows, &rows); err != nil {
		t.Fatalf("empty filter: %v", err)
	}
	if len(rows) != 1 || rows[0].Tags["env"] != "prod" {
		t.Fatalf("empty filter dropped data: %v", rows)
	}
}

// TestUnmarshalKeysCheckpoint is the shape the API exists for.
func TestUnmarshalKeysCheckpoint(t *testing.T) {
	const tensors, per = 32, 4096
	m := make(map[string][]uint16, tensors)
	for i := range tensors {
		m["model.layers."+strconv.Itoa(i)+".weight"] = mkWeights16(per, int64(i), bf16Bits)
	}
	blob, err := Marshal(m, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	want := "model.layers.19.weight"
	var got map[string][]uint16
	if err := UnmarshalKeys(blob, &got, want); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[want], m[want]) {
		t.Fatal("checkpoint projection wrong")
	}
}

// TestUnmarshalKeysNoLeakIntoColumnar pins the first bug an adversarial review
// found: a selected value containing a columnar []struct had the OUTER key
// list matched against its COLUMN names, silently dropping every column.
func TestUnmarshalKeysNoLeakIntoColumnar(t *testing.T) {
	type inner struct {
		Name string
		N    int64
		Tag  string
	}
	rows := make([]inner, 40)
	for i := range rows {
		rows[i] = inner{Name: "row" + strconv.Itoa(i), N: int64(i), Tag: "t"}
	}
	m := map[string]any{"aaa": rows, "zzz": int64(7)}
	for _, opts := range []Options{
		OptSpeed, OptBalanced, OptCompression,
		OptBalanced | OptColumnIndex, OptBalanced | OptMapShape, OptBalanced | OptCanonical,
	} {
		blob, err := Marshal(m, opts)
		if err != nil {
			t.Fatalf("%v encode: %v", opts, err)
		}
		var full, proj map[string]any
		if err := Unmarshal(blob, &full); err != nil {
			t.Fatalf("%v full: %v", opts, err)
		}
		if err := UnmarshalKeys(blob, &proj, "aaa"); err != nil {
			t.Fatalf("%v proj: %v", opts, err)
		}
		if !reflect.DeepEqual(proj["aaa"], full["aaa"]) {
			t.Fatalf("%v: selected value decoded differently under projection:\n proj=%v\n full=%v",
				opts, proj["aaa"], full["aaa"])
		}
	}
}

// TestUnmarshalColumnsKeepsMapFields pins the second bug: sharing one filter
// field between column names and map keys made a COLUMN projection drop
// entries from string-keyed map fields — silent data loss in an API that
// predates the key projection entirely.
func TestUnmarshalColumnsKeepsMapFields(t *testing.T) {
	type r struct {
		ID   int64
		Name string
		Tags map[string]string
	}
	rows := make([]r, 24)
	for i := range rows {
		rows[i] = r{
			ID: int64(i), Name: "n" + strconv.Itoa(i),
			Tags: map[string]string{"env": "prod", "region": "eu", "ID": "shadow"},
		}
	}
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression, OptBalanced | OptColumnIndex} {
		blob, err := Marshal(rows, opts)
		if err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		var proj []r
		if err := UnmarshalColumns(blob, &proj, "ID", "Tags"); err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		if len(proj) != len(rows) {
			t.Fatalf("%v: %d rows", opts, len(proj))
		}
		if !reflect.DeepEqual(proj[0].Tags, rows[0].Tags) {
			t.Fatalf("%v: column projection ate map entries: %v", opts, proj[0].Tags)
		}
	}
}

// TestUnmarshalKeysSkipKeepsInternState pins the third bug: Skip ran with the
// filter still live, so a skipped columnar value byte-skipped its columns
// instead of replaying them, and the intern/shape registrations inside were
// lost — later WANTED values then failed with an unknown state id, or decoded
// as garbage.
func TestUnmarshalKeysSkipKeepsInternState(t *testing.T) {
	type inner struct {
		Name string
		Tag  string
	}
	const shared = "interned-payload-string"
	rows := make([]inner, 40)
	for i := range rows {
		rows[i] = inner{Name: "row" + strconv.Itoa(i), Tag: shared}
	}
	tail := make([]string, 40)
	for i := range tail {
		tail[i] = shared
	}
	m := map[string]any{"aaa": rows, "bbb": tail, "ccc": rows, "zzz": tail}
	for _, opts := range []Options{OptBalanced, OptBalanced | OptColumnIndex, OptCompression} {
		blob, err := Marshal(m, opts)
		if err != nil {
			t.Fatalf("%v: %v", opts, err)
		}
		var full, proj map[string]any
		if err := Unmarshal(blob, &full); err != nil {
			t.Fatal(err)
		}
		if err := UnmarshalKeys(blob, &proj, "zzz"); err != nil {
			t.Fatalf("%v: skipping earlier values broke decoder state: %v", opts, err)
		}
		if !reflect.DeepEqual(proj["zzz"], full["zzz"]) {
			t.Fatalf("%v: projected value corrupted by skipped state: %v", opts, proj["zzz"])
		}
	}
}

// TestUnmarshalKeysLargeFilter pins the complexity fix: a big key list must not
// turn the projection into O(keys x entries) — entry counts are data-driven.
func TestUnmarshalKeysLargeFilter(t *testing.T) {
	const n = 4000
	m := make(map[string]int64, n)
	keys := make([]string, 0, n/2)
	for i := range n {
		k := "k" + strconv.Itoa(i)
		m[k] = int64(i)
		if i%2 == 0 {
			keys = append(keys, k)
		}
	}
	blob, err := Marshal(m, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]int64
	if err := UnmarshalKeys(blob, &got, keys...); err != nil {
		t.Fatal(err)
	}
	if len(got) != n/2 {
		t.Fatalf("got %d entries, want %d", len(got), n/2)
	}
	if got["k100"] != 100 || got["k3998"] != 3998 {
		t.Fatal("wrong values")
	}
}

// TestEmptyKeyFilterKeepsEverything pins one half of a footgun both
// projections shared: an empty-but-NON-NIL filter (how a caller hits it:
// keys := []string{} then keys...) must mean "no filter", not "drop
// everything". The column half of the same guard (wantField's len check) is
// hardening only — no payload shape was found that reaches wantField with an
// empty filter, so it is deliberately left untested rather than pinned by a
// test that would pass vacuously.
func TestEmptyKeyFilterKeepsEverything(t *testing.T) {
	m := map[string]int64{"a": 1, "b": 2}
	blob, err := Marshal(m, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	empty := []string{}
	var got map[string]int64
	if err := UnmarshalKeys(blob, &got, empty...); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("empty key filter dropped data: %v", got)
	}
}
