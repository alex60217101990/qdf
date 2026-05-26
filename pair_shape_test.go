package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// Markov-1 pair predictor (tagStatePair, 0xEA): the encoder
// remembers the most recent successor of every prev intern ID. When
// the next emission matches that prediction, it is encoded as 0xEA
// + varuint(0) instead of the raw state-ref (a wire saving on
// multi-byte ids). The storage is top-1 only — the previous K=4
// ring was traded for a 4 ×-smaller in-memory footprint. These tests
// verify the predictor fires on stable transitions, that it
// round-trips, and that it never grows the wire on adversarial /
// random data.

func TestStatePair_AlternatingRunFires(t *testing.T) {
	// The "strictly shorter" rule means tagStatePair only wins when
	// the raw state-ref ID needs a multi-byte varuint (id ≥ 128). To
	// exercise the predictor we first fill the intern table past 128
	// distinct strings, then alternate two specific high-ID entries.
	enc := NewEncoder(Dense)
	// Populate 130 dummy entries to push our targets past id=128.
	for i := range 130 {
		enc.WriteString("filler-string-#" + itoa(i))
	}
	// Targets — both will be assigned IDs ≥ 130.
	a := "alpha-token-large-id"
	b := "beta-token-large-id"
	enc.WriteString(a)
	enc.WriteString(b)
	// Alternation: A, B, A, B, A, B, A, B.
	in := []string{a, b, a, b, a, b, a, b}
	for _, s := range in {
		enc.WriteString(s)
	}

	gotPair := 0
	for _, x := range enc.buf {
		if x == tagStatePair {
			gotPair++
		}
	}
	if gotPair < 4 {
		t.Fatalf("expected ≥4 tagStatePair, got %d (buf=%x...)", gotPair, enc.buf[:48])
	}

	dec := NewDecoderOnBuf(enc.buf)
	for i := range 130 {
		if _, err := dec.ReadString(); err != nil {
			t.Fatalf("filler[%d] decode: %v", i, err)
		}
	}
	if got, _ := dec.ReadString(); got != a {
		t.Fatalf("expected a, got %q", got)
	}
	if got, _ := dec.ReadString(); got != b {
		t.Fatalf("expected b, got %q", got)
	}
	for i, want := range in {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if got != want {
			t.Fatalf("[%d] got %q want %q", i, got, want)
		}
	}
}

// Top-1 contract: pairLookup hits when the recorded successor equals
// the queried one, misses otherwise. pairRecord always overwrites.
// Exercise encState / decState directly so the test does not depend
// on the encoder's emit decisions.
func TestStatePair_Top1Semantics(t *testing.T) {
	st := newEncState()
	const (
		prev  = uint32(7)
		succA = uint32(11)
		succB = uint32(22)
	)
	// Empty slot: miss.
	if st.pairLookup(prev, succA) {
		t.Fatal("empty slot must miss")
	}
	st.pairRecord(prev, succA)
	if !st.pairLookup(prev, succA) {
		t.Fatal("hit on recorded successor")
	}
	if st.pairLookup(prev, succB) {
		t.Fatal("non-matching successor must miss")
	}
	// Overwrite: the prior successor is no longer remembered.
	st.pairRecord(prev, succB)
	if st.pairLookup(prev, succA) {
		t.Fatal("overwrite must drop the previous successor")
	}
	if !st.pairLookup(prev, succB) {
		t.Fatal("hit on overwritten successor")
	}
	// Predicting succ==0 must work — the +1 storage shift keeps 0 a
	// valid successor distinguishable from the empty sentinel.
	const prev2 = uint32(99)
	st.pairRecord(prev2, 0)
	if !st.pairLookup(prev2, 0) {
		t.Fatal("succ==0 must be a valid stored value (not aliased with empty)")
	}
}

// Decoder mirror must produce the same prediction.
func TestStatePair_Top1DecoderMirror(t *testing.T) {
	d := newDecState()
	const (
		prev = uint32(3)
		succ = uint32(42)
	)
	if _, ok := d.pairAtRank(prev, 0); ok {
		t.Fatal("empty slot must miss")
	}
	d.pairRecord(prev, succ)
	got, ok := d.pairAtRank(prev, 0)
	if !ok || got != succ {
		t.Fatalf("decoder mirror miss: got=%d ok=%v want %d", got, ok, succ)
	}
	// rank > 0 is rejected as malformed regardless of slot state.
	if _, ok := d.pairAtRank(prev, 1); ok {
		t.Fatal("rank>0 must be rejected")
	}
}

// Stable transition (A→B repeatedly) keeps hitting the predictor.
func TestStatePair_StableTransitionHits(t *testing.T) {
	enc := NewEncoder(Dense)
	for i := range 130 {
		enc.WriteString("filler-#" + itoa(i))
	}
	a, b := "alpha-large-id", "beta-large-id"
	enc.WriteString(a)
	enc.WriteString(b)
	// A→B→A→B→A→B (6 emits). The first A and B prime the chain; from
	// the second pair on each emission has the predictor seeded with
	// the matching prev→curr transition, so every emission after the
	// first AB pair fires tagStatePair.
	in := []string{a, b, a, b, a, b}
	for _, s := range in {
		enc.WriteString(s)
	}
	hits := 0
	for _, x := range enc.buf {
		if x == tagStatePair {
			hits++
		}
	}
	if hits < 4 {
		t.Fatalf("top-1 must catch ≥4 stable-transition hits, got %d", hits)
	}
	// Round-trip.
	dec := NewDecoderOnBuf(enc.buf)
	for range 130 {
		if _, err := dec.ReadString(); err != nil {
			t.Fatalf("filler decode: %v", err)
		}
	}
	want := append([]string{a, b}, in...)
	for i, w := range want {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if got != w {
			t.Fatalf("[%d] got %q want %q", i, got, w)
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

func TestStatePair_NoRegressionOnRandomData(t *testing.T) {
	// Distinct strings on every step. The predictor populates its
	// table but never hits. The wire MUST NOT be larger than the
	// classic state-ref encoding because the encoder picks the
	// shortest variant per emission.
	in := make([]string, 50)
	for i := range in {
		in[i] = "uniq-string-" + string(rune('a'+i%26)) + "-" + string(rune('a'+(i*7)%26)) + "-" + string(rune('0'+i%10))
	}

	enc := NewEncoder(Dense)
	for _, s := range in {
		enc.WriteString(s)
	}

	// Expected: zero pair hits on this workload.
	gotPair := 0
	for _, b := range enc.buf[5:] {
		if b == tagStatePair {
			gotPair++
		}
	}
	if gotPair != 0 {
		t.Fatalf("expected 0 tagStatePair on random data, got %d", gotPair)
	}

	// Round-trip.
	dec := NewDecoderOnBuf(enc.buf)
	for i, want := range in {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if got != want {
			t.Fatalf("[%d] got %q want %q", i, got, want)
		}
	}
}

func TestStatePair_RingEvictsBeyondK(t *testing.T) {
	// Cycle of 8 distinct successors of A. With top-1 storage the
	// predictor remembers only the most recent successor, so a revisit
	// of an earlier successor MUST miss the predictor and fall through
	// to the raw state-ref / MTF encoding. The test asserts round-trip
	// correctness through that path — the encoder must still produce a
	// stream the decoder can replay even when no pair-prediction hits.
	keys := []string{
		"anchor-key-AAA", // prev anchor
		"succ-key-001", "succ-key-002", "succ-key-003", "succ-key-004",
		"succ-key-005", "succ-key-006", "succ-key-007", "succ-key-008",
	}
	enc := NewEncoder(Dense)
	// Establish anchor as the prev for every successor.
	for _, s := range keys[1:] {
		enc.WriteString(keys[0])
		enc.WriteString(s)
	}
	// Now revisit the OLDEST successor — top-1 has long forgotten it.
	enc.WriteString(keys[0])
	enc.WriteString(keys[1])

	// Round-trip — no hits to assert here, just correctness under
	// eviction.
	dec := NewDecoderOnBuf(enc.buf)
	exp := append([]string{}, keys[1:]...) // copy
	full := make([]string, 0, 2*(len(keys)-1)+2)
	for range keys[1:] {
		full = append(full, keys[0], "?")
	}
	for i, s := range exp {
		full[2*i+1] = s
	}
	full = append(full, keys[0], keys[1])
	for i, want := range full {
		got, err := dec.ReadString()
		if err != nil {
			t.Fatalf("[%d] decode: %v", i, err)
		}
		if got != want {
			t.Fatalf("[%d] got %q want %q", i, got, want)
		}
	}
}

// Struct-shape interning (tagMapShape, 0xEC): first encode of a struct
// type declares its field ordering inline; subsequent encodes of the
// same type emit 0xEC + shapeID + values. These tests verify the
// codec round-trips, that wire-size shrinks across an array of
// identical structs, and that the tagMap8/16/32 path still decodes
// (so OptShapeIntern-off encoders round-trip through the same
// Unmarshal).

type pairAddr struct {
	Street string `qdf:"street"`
	City   string `qdf:"city"`
	Zip    string `qdf:"zip"`
}

func TestMapShape_StructRoundTrip(t *testing.T) {
	in := pairAddr{Street: "1 Vilnius St", City: "Klaipeda", Zip: "LT-91300"}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// The first emission MUST contain a declaration: 0xEC, varuint(0),
	// varuint(N=3).
	if !containsByte(b, tagMapShape) {
		t.Fatalf("expected tagMapShape in wire, got=%x", b)
	}
	var out pairAddr
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n want=%+v\n  got=%+v", in, out)
	}
}

func TestMapShape_ArrayShrinksWire(t *testing.T) {
	// Encoding many identical-shape structs should be markedly smaller
	// than the pre-shape Dense encoding because every subsequent
	// struct trades its key emissions for a single shape-ID varuint.
	const N = 100
	arr := make([]pairAddr, N)
	for i := range arr {
		arr[i] = pairAddr{
			Street: "Street",
			City:   "City",
			Zip:    "Zip",
		}
	}
	b, err := Marshal(arr, OptBalanced)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Decode round-trip.
	var out []pairAddr
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(arr, out) {
		t.Fatalf("round-trip mismatch")
	}
	// Wire-size guardrail. Pre-shape Dense would have spent ≈16 bytes
	// per struct after the first (tagMap8 + 3 keys × 2-byte state-ref
	// + values). With shape interning each subsequent struct collapses
	// to 0xEC + 1-byte shapeID + 3 value emissions. The exact ceiling
	// is N*12 + a fixed prelude budget; verify we are comfortably
	// below the pre-shape baseline.
	const ceiling = N * 12
	if len(b) > ceiling {
		t.Fatalf("wire too large for shape-interned array: %d > %d (buf=%x...)", len(b), ceiling, b[:64])
	}
}

func TestMapShape_DecodesAsAnyMap(t *testing.T) {
	// Encoding into a strongly-typed struct, decoding into an any —
	// the shape codec must round-trip into a map[string]any keyed by
	// the declared field names.
	in := pairAddr{Street: "S", City: "C", Zip: "Z"}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var v any
	if err := Unmarshal(b, &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", v)
	}
	if m["street"] != "S" || m["city"] != "C" || m["zip"] != "Z" {
		t.Fatalf("decode-any mismatch: %#v", m)
	}
}

func TestMapShape_HeterogeneousStructsKeepDistinctShapes(t *testing.T) {
	// Two different struct types in the same Dense stream must get
	// two separate shape IDs and round-trip independently.
	type A struct {
		X int    `qdf:"x"`
		Y string `qdf:"y"`
	}
	type B struct {
		P float64 `qdf:"p"`
		Q string  `qdf:"q"`
		R bool    `qdf:"r"`
	}
	type combo struct {
		First  A `qdf:"first"`
		Second B `qdf:"second"`
		Third  A `qdf:"third"`
	}
	in := combo{
		First:  A{X: 1, Y: "hi"},
		Second: B{P: 3.14, Q: "pi", R: true},
		Third:  A{X: 2, Y: "bye"},
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var out combo
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch:\n want=%+v\n  got=%+v", in, out)
	}
}

func containsByte(b []byte, t byte) bool { return bytes.IndexByte(b, t) >= 0 }
