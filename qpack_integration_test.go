package qdf

import (
	"reflect"
	"testing"
)

// End-to-end Marshal/Unmarshal exercises that go through the public API
// after Phase 7 wired QPack into MarshalDense and MarshalQPack.

type qpackBatch struct {
	Bools   []bool    `qdf:"bools"`
	IDs     []uint64  `qdf:"ids"`
	TS      []int64   `qdf:"ts"`
	Vec32   []float32 `qdf:"vec32"`
	Vec64   []float64 `qdf:"vec64"`
	Tag     string    `qdf:"tag"`
	Counter int       `qdf:"counter"`
}

func sampleQPackBatch() qpackBatch {
	b := qpackBatch{
		Bools:   make([]bool, 100),
		IDs:     make([]uint64, 256),
		TS:      make([]int64, 256),
		Vec32:   make([]float32, 64),
		Vec64:   make([]float64, 64),
		Tag:     "hello-qpack",
		Counter: 42,
	}
	for i := range b.Bools {
		b.Bools[i] = i%3 == 0
	}
	for i := range b.IDs {
		b.IDs[i] = 1_700_000_000 + uint64(i)*10
	}
	for i := range b.TS {
		b.TS[i] = -1_000 + int64(i)
	}
	for i := range b.Vec32 {
		b.Vec32[i] = float32(i) * 0.5
	}
	for i := range b.Vec64 {
		b.Vec64[i] = float64(i) * 0.25
	}
	return b
}

func TestMarshalDense_QPackRoundTrip(t *testing.T) {
	in := sampleQPackBatch()
	raw, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out qpackBatch
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("dense round-trip mismatch")
	}
}

func TestMarshalQPack_RoundTrip(t *testing.T) {
	in := sampleQPackBatch()
	raw, err := Marshal(in, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	var out qpackBatch
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("qpack round-trip mismatch")
	}
}

func TestQPack_SizeWinVsLegacy(t *testing.T) {
	in := sampleQPackBatch()
	leg, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	qp, err := Marshal(in, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("legacy=%d  qpack=%d  ratio=%.2fx", len(leg), len(qp), float64(len(leg))/float64(len(qp)))
	if len(qp) >= len(leg) {
		t.Fatalf("qpack not smaller than legacy: %d vs %d", len(qp), len(leg))
	}
}

func TestQPack_DenseSmallerThanFast(t *testing.T) {
	// Repeat the same string + qpack-amenable arrays. Dense should beat
	// fast-qpack here because of string interning.
	type rep struct {
		Country []string `qdf:"country"`
		IDs     []uint64 `qdf:"ids"`
	}
	in := rep{
		Country: make([]string, 64),
		IDs:     make([]uint64, 128),
	}
	for i := range in.Country {
		in.Country[i] = "Lithuania"
	}
	for i := range in.IDs {
		in.IDs[i] = uint64(i) + 1000
	}
	qp, _ := Marshal(in, OptQPack)
	dn, _ := Marshal(in, OptBalanced)
	t.Logf("fast-qpack=%d  dense=%d", len(qp), len(dn))
	if len(dn) >= len(qp) {
		t.Fatalf("dense should be smaller on string-repetitive payload: %d vs %d", len(dn), len(qp))
	}
}

func TestQPack_DecodeLegacyBuffer(t *testing.T) {
	// A buffer produced by the legacy Marshal must still decode by the
	// new Unmarshal path.
	in := sampleQPackBatch()
	leg, _ := Marshal(in, OptSpeed)
	var out qpackBatch
	if err := Unmarshal(leg, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("legacy decode mismatch")
	}
}
