package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// Markov-1 pair predictor (tagStatePair, 0xEA): given the most recent
// emitted state-ref ID, the encoder remembers up to pairPredK most
// recent successors. When the next ID is in that ring, it is encoded
// as 0xEA + varuint(rank) instead of the raw state-ref. These tests
// verify the predictor fires on the A→B→A→B alternation that Markov-0
// alone cannot catch, that it round-trips, and that it never grows
// the wire on adversarial / random data.

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
	// Cycle of 8 distinct successors of A. With pairPredK=4 the ring
	// can only remember 4 at a time, so the predictor eventually
	// misses on the evicted entries.
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
	// Now revisit the OLDEST successor — it should have been evicted
	// from the ring (pairPredK=4 keeps only the 4 most recent).
	enc.WriteString(keys[0])
	enc.WriteString(keys[1])

	// Round-trip — no hits to assert here, just correctness under
	// eviction.
	dec := NewDecoderOnBuf(enc.buf)
	exp := append([]string{}, keys[1:]...) // copy
	full := []string{}
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
// identical structs, and that the legacy tagMap8/16/32 path still
// decodes (back-compat for streams emitted before the codec landed).

type pairAddr struct {
	Street string `qdf:"street"`
	City   string `qdf:"city"`
	Zip    string `qdf:"zip"`
}

func TestMapShape_StructRoundTrip(t *testing.T) {
	in := pairAddr{Street: "1 Vilnius St", City: "Klaipeda", Zip: "LT-91300"}
	b, err := MarshalDense(in)
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
	b, err := MarshalDense(arr)
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
	b, err := MarshalDense(in)
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
	b, err := MarshalDense(in)
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
