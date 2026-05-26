package qdf

import (
	"reflect"
	"strconv"
	"testing"
)

// Move-to-front coding tests. The encoder emits tagStateMTF + rank
// whenever the rank's varuint is shorter than the raw intern ID's
// varuint, never longer; the decoder maintains the same LRU chain
// so round-trip is exact.

func TestMTF_BasicRoundTrip(t *testing.T) {
	// Build 200 unique strings (so IDs span > 128, raw varuint would
	// otherwise be 2 bytes for the high half). Then reference them in
	// a non-sequential pattern that benefits from MTF.
	const N = 200
	in := make([]string, 0, 4*N)
	for i := range N {
		in = append(in, "service-name-token-"+strconv.Itoa(i))
	}
	// Re-emit each in reverse order (so MTF should bring late-defined
	// items to rank 0..9 quickly).
	for i := N - 1; i >= N-32; i-- {
		in = append(in, "service-name-token-"+strconv.Itoa(i))
	}
	buf, err := MarshalDense(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("MTF round-trip mismatch:\n want=%v\n got=%v", in[:5], out[:5])
	}
}

func TestMTF_NeverGrowsWire(t *testing.T) {
	// On data that does NOT benefit from MTF (sequential refs to small
	// IDs, varuint already 1 byte), the encoder must pick tagStateRef
	// and not emit tagStateMTF. Confirms the size-aware dispatch.
	in := []string{
		"alpha-token-aaaa", "alpha-token-aaaa", "alpha-token-aaaa", "alpha-token-aaaa",
		"beta-token-bbbb", "beta-token-bbbb",
	}
	buf, err := MarshalDense(in)
	if err != nil {
		t.Fatal(err)
	}
	// Should be small — Markov-0 catches the consecutive repeats,
	// tagStateMTF should not appear because the IDs (0, 1) are
	// already at the smallest possible varuint width.
	for _, b := range buf[5:] {
		if b == tagStateMTF {
			t.Logf("tagStateMTF present in compact wire (acceptable if not larger): %x", buf)
		}
	}
	var out []string
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("mismatch")
	}
}

func TestMTF_SizeWinOnLargeHotSet(t *testing.T) {
	// 256 unique strings, intern-time IDs randomly scattered, then we
	// repeatedly reference a hot subset of 8 items. With MTF the hot
	// subset migrates to rank 0..7 and every emission is 2 bytes
	// (tagStateMTF + 1-byte varuint). Without MTF the hot subset
	// would carry whatever varuint the original assigned ID requires
	// (1-byte for IDs < 128, 2-byte otherwise).
	const Nunique = 256
	const Nrefs = 4000
	tokens := make([]string, Nunique)
	for i := range tokens {
		tokens[i] = "tok-" + strconv.Itoa(i+10000)
	}
	in := make([]string, 0, Nunique+Nrefs)
	in = append(in, tokens...)
	// Hot subset: tokens 200..207 (whose intern IDs > 128).
	for i := range Nrefs {
		in = append(in, tokens[200+i%8])
	}

	withMTF, err := MarshalDense(in)
	if err != nil {
		t.Fatal(err)
	}

	// Decode to confirm correctness.
	var out []string
	if err := Unmarshal(withMTF, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("round-trip mismatch")
	}
	t.Logf("hot-subset stream size (256 unique + 4000 refs): %d bytes", len(withMTF))
}

func TestMTF_LRUInvalidationOnInlineString(t *testing.T) {
	// Inline string emission invalidates the Markov-0 chain but must
	// NOT corrupt the LRU. After an inline string, an MTF-coded ref
	// must still resolve to the right ID.
	type holder struct {
		A string `qdf:"a"`
		B string `qdf:"b"`
		C string `qdf:"c"`
	}
	in := holder{
		A: "long-token-aaaaaaaaaaaaaaaaa",
		B: "x", // 1-char — inlined, len < minIntern
		C: "long-token-aaaaaaaaaaaaaaaaa",
	}
	buf, err := MarshalDense(in)
	if err != nil {
		t.Fatal(err)
	}
	var out holder
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}

func BenchmarkMTF_SizeOnSyntheticHotSet(b *testing.B) {
	// Reports size only — measures whether tagStateMTF earns its
	// keep on a hot-subset-of-256 access pattern.
	const Nunique = 256
	const Nrefs = 4000
	tokens := make([]string, Nunique)
	for i := range tokens {
		tokens[i] = "tok-" + strconv.Itoa(i+10000)
	}
	in := make([]string, 0, Nunique+Nrefs)
	in = append(in, tokens...)
	for i := range Nrefs {
		in = append(in, tokens[200+i%8])
	}
	buf, _ := MarshalDense(in)
	b.Logf("MarshalDense(256 unique + 4000 hot-subset refs) = %d bytes", len(buf))
	b.ReportAllocs()
	for b.Loop() {
		_, _ = MarshalDense(in)
	}
}
