package qdf

import (
	"reflect"
	"strconv"
	"testing"
)

// TestMapRecycleRoundTrip exercises the []struct{map} batch recycle path: the
// second Unmarshal into the SAME target harvests the first decode's per-element
// maps onto the Decoder free-list and pops them back, cleared. Round-trip must
// be exact both times with no stale keys.
func TestMapRecycleRoundTrip(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	const N = 32
	in := make([]rec, N)
	for i := range in {
		in[i].Tags = map[string]string{
			"k" + strconv.Itoa(i):     "v" + strconv.Itoa(i),
			"shared":                  strconv.Itoa(i),
			"extra" + strconv.Itoa(i): "z",
		}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v", out)
	}
	// Second decode into the same out — recycle path (harvest + pop).
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
	// Decode a payload with FEWER keys into the same target — stale keys from
	// the recycled maps must not survive (clear before refill).
	in2 := make([]rec, N)
	for i := range in2 {
		in2[i].Tags = map[string]string{"only": strconv.Itoa(i)}
	}
	data2, _ := Marshal(in2, OptBalanced)
	if err := Unmarshal(data2, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in2, out) {
		t.Fatalf("stale keys leaked: got %v want %v", out, in2)
	}
}

// BenchmarkMapRecycle decodes a []struct{map[string]string} (256 records ×16
// entries) repeatedly into a REUSED target — the recycle path. -benchmem shows
// allocs/op and B/op; compare to the documented no-recycle baseline (~5518
// allocs / 382KB).
func BenchmarkMapRecycle(b *testing.B) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	const N = 256
	in := make([]rec, N)
	for i := range in {
		m := make(map[string]string, 16)
		for j := range 16 {
			m["k"+strconv.Itoa(j)] = "v" + strconv.Itoa(i*16+j)
		}
		in[i].Tags = m
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil { // warm: first decode allocates
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		if err := Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMapRecycleBaseline is the no-recycle control: each iteration decodes
// into a FRESH target, so decode-slice-reuse has nothing to harvest and every
// per-element map is allocated. Apples-to-apples alloc comparison vs
// BenchmarkMapRecycle on this host.
func BenchmarkMapRecycleBaseline(b *testing.B) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	const N = 256
	in := make([]rec, N)
	for i := range in {
		m := make(map[string]string, 16)
		for j := range 16 {
			m["k"+strconv.Itoa(j)] = "v" + strconv.Itoa(i*16+j)
		}
		in[i].Tags = m
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()

	for b.Loop() {
		var out []rec // fresh target → no reuse, no harvest, no recycle
		if err := Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// TestMapRecycleNestedStruct covers a map nested inside a sub-struct of a slice
// element — harvestMaps must recurse into struct fields to reach it.
func TestMapRecycleNestedStruct(t *testing.T) {
	type inner struct {
		Tags map[string]string `qdf:"tags"`
	}
	type outer struct {
		ID    int   `qdf:"id"`
		Inner inner `qdf:"inner"` // map is one struct level deep
	}
	in := make([]outer, 32)
	for i := range in {
		in[i] = outer{ID: i, Inner: inner{Tags: map[string]string{
			"k" + strconv.Itoa(i): "v" + strconv.Itoa(i), "s": strconv.Itoa(i),
		}}}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []outer
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v", out)
	}
	// Second decode into the same target — the nested maps must recycle and stay
	// correct (recursion reached them).
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled nested: got %v want %v", out, in)
	}
}

func TestMapReuseRoundTrip(t *testing.T) {
	type rec struct {
		ID   int               `qdf:"id"`
		Tags map[string]string `qdf:"tags"`
	}
	in := []rec{
		{ID: 1, Tags: map[string]string{"a": "x", "b": "y"}},
		{ID: 2, Tags: map[string]string{"c": "z"}},
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v", out)
	}
	// decode AGAIN into the SAME out (reuse path) — must still equal in,
	// no stale keys from the first decode.
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("reused: got %v", out)
	}
}

func TestMapReuseNoStaleKeys(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	a := []rec{{Tags: map[string]string{"old1": "1", "old2": "2"}}}
	b := []rec{{Tags: map[string]string{"new": "3"}}}
	da, _ := Marshal(a, OptBalanced)
	db, _ := Marshal(b, OptBalanced)
	var out []rec
	if err := Unmarshal(da, &out); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(db, &out); err != nil {
		t.Fatal(err)
	} // reuse the map
	if !reflect.DeepEqual(b, out) {
		t.Fatalf("stale keys leaked: got %v want %v", out, b)
	}
	if _, ok := out[0].Tags["old1"]; ok {
		t.Fatal("stale key old1 survived")
	}
}

// TestMapReuseCutsAllocs proves the win on a DIRECT struct-with-map target (the
// RPC single-message decode shape), where the map field survives between decodes
// and reuseOrMakeMap can clear+reuse it. NOTE: a []struct{map} batch target does
// NOT benefit — decode-slice-reuse zeroes the slice elements (nil-ing the maps)
// before the map decoder runs, so each map is allocated fresh; that limitation
// is by design (slice-clear is the schema-evolution stale-data guard).
func TestMapReuseCutsAllocs(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	in := rec{Tags: make(map[string]string, 16)}
	for j := range 16 {
		in.Tags["k"+strconv.Itoa(j)] = "v" + strconv.Itoa(j)
	}
	data, _ := Marshal(in, OptBalanced)

	// The alloc win is that decoding into a target whose map field is already
	// non-nil REUSES that map's backing (reuseOrMakeMap clears + refills it)
	// instead of allocating a new one. Assert that directly via map-pointer
	// identity — a deterministic, instrumentation-immune signal — rather than
	// testing.AllocsPerRun, whose absolute counts are perturbed by the race
	// detector's shadow memory and the coverage atomic counters (which made the
	// earlier reuse<fresh form flake in the -race+coverage CI lane even though
	// the optimization fires correctly).
	var out rec
	if err := Unmarshal(data, &out); err != nil { // warm: allocates the map once
		t.Fatal(err)
	}
	backing := reflect.ValueOf(out.Tags).Pointer()
	if backing == 0 {
		t.Fatal("decode produced a nil map")
	}
	for i := range 8 {
		if err := Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		if got := reflect.ValueOf(out.Tags).Pointer(); got != backing {
			t.Fatalf("decode %d reallocated the map (%#x != %#x): reuseOrMakeMap is not reusing the backing", i, got, backing)
		}
		if !reflect.DeepEqual(out.Tags, in.Tags) {
			t.Fatalf("decode %d: value wrong after reuse: %v", i, out.Tags)
		}
	}

	// Sanity that the stable pointer above is genuine reuse, not a fixed address:
	// a separate fresh target (out is still live, so its backing is not freed)
	// must get a DIFFERENT backing map.
	var fresh rec
	if err := Unmarshal(data, &fresh); err != nil {
		t.Fatal(err)
	}
	if reflect.ValueOf(fresh.Tags).Pointer() == backing {
		t.Fatal("a fresh target unexpectedly shares the reused map backing")
	}
}

// TestMapRecycleTwoFieldsSameType decodes a []struct with TWO map fields of the
// same type. Both fields' maps are harvested into the SAME per-type bucket; the
// recycle pop must not cross-wire A's map into B (or vice versa). Round-trip
// exact both times.
func TestMapRecycleTwoFieldsSameType(t *testing.T) {
	type rec struct {
		A map[string]string `qdf:"a"`
		B map[string]string `qdf:"b"`
	}
	const N = 24
	in := make([]rec, N)
	for i := range in {
		in[i].A = map[string]string{"a" + strconv.Itoa(i): "av" + strconv.Itoa(i), "shared": "A"}
		in[i].B = map[string]string{"b" + strconv.Itoa(i): "bv" + strconv.Itoa(i), "shared": "B"}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapRecycleTwoFieldsDiffType decodes a []struct with two map fields of
// DIFFERENT element types. The per-type buckets must keep them separate; a
// map[string]int must never be popped where a map[string]string is wanted.
func TestMapRecycleTwoFieldsDiffType(t *testing.T) {
	type rec struct {
		S map[string]string `qdf:"s"`
		I map[string]int    `qdf:"i"`
	}
	const N = 24
	in := make([]rec, N)
	for i := range in {
		in[i].S = map[string]string{"s" + strconv.Itoa(i): "v" + strconv.Itoa(i)}
		in[i].I = map[string]int{"n" + strconv.Itoa(i): i, "k": i * 2}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapRecycleSliceOfMap decodes a []map[string]string (slice whose ELEMENT
// is a map, not a struct) into a reused target. Recycle must work via the
// reflect.Map branch of harvestMaps.
func TestMapRecycleSliceOfMap(t *testing.T) {
	const N = 20
	in := make([]map[string]string, N)
	for i := range in {
		in[i] = map[string]string{"k" + strconv.Itoa(i): "v" + strconv.Itoa(i), "c": strconv.Itoa(i)}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]string
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle via reflect.Map branch
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapRecycleNilRoundTrips checks that the nil-vs-empty distinction qdf keeps
// survives the recycle path: a first batch with populated maps primes the
// free-list; a second batch whose maps are nil must decode back to nil (the
// absent/nil map must not be resurrected from a recycled non-nil map).
func TestMapRecycleNilRoundTrips(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	const N = 16
	withTags := make([]rec, N)
	for i := range withTags {
		withTags[i].Tags = map[string]string{"k" + strconv.Itoa(i): "v" + strconv.Itoa(i)}
	}
	nilTags := make([]rec, N) // all Tags == nil
	d1, err := Marshal(withTags, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := Marshal(nilTags, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(d1, &out); err != nil { // prime: populates + would harvest
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withTags, out) {
		t.Fatalf("prime: got %v want %v", out, withTags)
	}
	if err := Unmarshal(d2, &out); err != nil { // nil batch into reused target
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nilTags, out) {
		t.Fatalf("nil did not round-trip: got %v want %v", out, nilTags)
	}
	for i := range out {
		if out[i].Tags != nil {
			t.Fatalf("element %d: nil map resurrected as %v", i, out[i].Tags)
		}
	}
}

// TestMapRecycleCapCorrectness decodes a 100-record []struct{map} batch twice.
// The retention cap (maxRecycledMaps) bounds the free-list but must never affect
// correctness: every record round-trips exactly on the recycle pass.
func TestMapRecycleCapCorrectness(t *testing.T) {
	type rec struct {
		Tags map[string]string `qdf:"tags"`
	}
	const N = 100
	in := make([]rec, N)
	for i := range in {
		in[i].Tags = map[string]string{
			"id" + strconv.Itoa(i): strconv.Itoa(i),
			"grp":                  strconv.Itoa(i % 7),
		}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapReuseReflectNonStringKey exercises the reflect decodeMap recycle path
// for a non-string-keyed map (map[int]string has no generated fast-path codec).
// Decoding TWICE into the same target must harvest+pop the int-keyed maps and
// round-trip exactly both times.
func TestMapReuseReflectNonStringKey(t *testing.T) {
	type rec struct {
		M map[int]string `qdf:"m"`
	}
	const N = 32
	in := make([]rec, N)
	for i := range in {
		in[i].M = map[int]string{
			i:     "v" + strconv.Itoa(i),
			i + 1: "w" + strconv.Itoa(i),
			1000:  strconv.Itoa(i),
		}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapReuseReflectSliceValue exercises the reflect decodeMap recycle path for
// a slice-valued map (map[string][]int). Guards that recycled map values do not
// alias across entries — the existing vh.SetZero per entry must still force a
// fresh slice backing even when the map itself is recycled.
func TestMapReuseReflectSliceValue(t *testing.T) {
	type rec struct {
		M map[string][]int `qdf:"m"`
	}
	const N = 32
	in := make([]rec, N)
	for i := range in {
		in[i].M = map[string][]int{
			"a": {i, i + 1, i + 2},
			"b": {i * 2},
			"c": {i, i, i, i},
		}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapReuseSchemalessAny exercises decodeAny's popOrMakeMap: a []map[string]any
// decoded schemalessly TWICE into the same target. The harvested map[string]any
// elements must be recycled and cleared, round-tripping exactly both times.
func TestMapReuseSchemalessAny(t *testing.T) {
	// Use string values only: schemaless `any` round-trips int64 as uint64 for
	// small fixint values (documented inherent coercion — the wire can't tell
	// int from uint), which is orthogonal to map recycling. Strings round-trip
	// exactly, so DeepEqual cleanly isolates the recycle path under test.
	in := []map[string]any{
		{"a": "1", "b": "x"},
		{"b": "x", "c": "2"},
		{"a": "3", "z": "q"},
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path (decodeAny)
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}

// TestMapReuseMixedFastAndReflect exercises a struct carrying BOTH a fast-path
// map (map[string]string) and a reflect-path map (map[int]int) in the same
// element. Each type buckets separately on the free-list; decoding twice into
// the same target must recycle both and round-trip exactly.
func TestMapReuseMixedFastAndReflect(t *testing.T) {
	type rec struct {
		Fast map[string]string `qdf:"fast"`
		Refl map[int]int       `qdf:"refl"`
	}
	const N = 32
	in := make([]rec, N)
	for i := range in {
		in[i].Fast = map[string]string{
			"k" + strconv.Itoa(i): "v" + strconv.Itoa(i),
			"shared":              strconv.Itoa(i),
		}
		in[i].Refl = map[int]int{i: i * 2, i + 100: i * 3}
	}
	data, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []rec
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("fresh: got %v want %v", out, in)
	}
	if err := Unmarshal(data, &out); err != nil { // recycle path
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("recycled: got %v want %v", out, in)
	}
}
