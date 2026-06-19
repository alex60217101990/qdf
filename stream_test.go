package qdf

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestStream_FastRoundTrip(t *testing.T) {
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Fast)
	in := []Inner{{1, 1.1}, {2, 2.2}, {3, 3.3}}
	for _, v := range in {
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()
	var out []Inner
	for {
		var v Inner
		if err := dec.Decode(&v); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		out = append(out, v)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("stream mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestStream_DenseSharesInternTable(t *testing.T) {
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	type Evt struct {
		Country string `qdf:"country"`
		City    string `qdf:"city"`
		ID      int    `qdf:"id"`
	}
	in := make([]Evt, 100)
	for i := range in {
		in[i] = Evt{Country: "Lithuania", City: "Vilnius", ID: i}
	}
	for _, v := range in {
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	streamSize := w.Len()

	// Encode each separately for comparison — each would re-emit the
	// dictionary, so the savings come from the shared state table.
	soloTotal := 0
	for _, v := range in {
		b, err := Marshal(v, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		soloTotal += len(b)
	}
	t.Logf("stream=%d solo=%d ratio=%.2f", streamSize, soloTotal, float64(streamSize)/float64(soloTotal))
	if streamSize >= soloTotal {
		t.Fatalf("expected stream encoding to be smaller than concatenated solo encodings (shared intern wins)")
	}
}

// TestStreamEncoderReset verifies one StreamEncoder (and StreamDecoder) reused
// across independent streams via Reset: each batch round-trips correctly, the
// reused encoder does not leak a fresh newEncState per batch, and the cleared
// cross-message state keeps the streams independent.
func TestStreamEncoderReset(t *testing.T) {
	type rec struct {
		Name string `qdf:"name"`
		N    int    `qdf:"n"`
	}
	batchA := []rec{{"alpha", 1}, {"alpha", 2}, {"beta", 3}}
	batchB := []rec{{"gamma", 10}, {"gamma", 11}}

	enc := NewStreamEncoderWith(nil, OptBalanced)
	defer enc.Close()
	dec := NewStreamDecoder(nil)
	defer dec.Close()

	encodeBatch := func(b []rec) []byte {
		var w bytes.Buffer
		enc.Reset(&w)
		for _, r := range b {
			if err := enc.Encode(r); err != nil {
				t.Fatal(err)
			}
		}
		if err := enc.Flush(); err != nil {
			t.Fatal(err)
		}
		return append([]byte(nil), w.Bytes()...)
	}
	decodeBatch := func(buf []byte, n int) []rec {
		dec.Reset(bytes.NewReader(buf))
		out := make([]rec, 0, n)
		for range n {
			var r rec
			if err := dec.Decode(&r); err != nil {
				t.Fatal(err)
			}
			out = append(out, r)
		}
		return out
	}

	bufA := encodeBatch(batchA)
	bufB := encodeBatch(batchB) // same encoder, Reset between batches

	gotA := decodeBatch(bufA, len(batchA))
	gotB := decodeBatch(bufB, len(batchB))
	gotA2 := decodeBatch(bufA, len(batchA)) // decode A again after B: decoder Reset must fully clear

	for i := range batchA {
		if gotA[i] != batchA[i] || gotA2[i] != batchA[i] {
			t.Fatalf("batchA[%d]: got %v / %v, want %v", i, gotA[i], gotA2[i], batchA[i])
		}
	}
	for i := range batchB {
		if gotB[i] != batchB[i] {
			t.Fatalf("batchB[%d]: got %v, want %v", i, gotB[i], batchB[i])
		}
	}

	// Reuse keeps the per-batch encode allocation-light (no fresh newEncState).
	var w bytes.Buffer
	allocs := testing.AllocsPerRun(50, func() {
		w.Reset()
		enc.Reset(&w)
		for _, r := range batchA {
			_ = enc.Encode(r)
		}
		_ = enc.Flush()
	})
	if allocs > 12 {
		t.Fatalf("Reset reuse should be low-alloc, got %.1f (newEncState leaking per batch?)", allocs)
	}
}

// TestStreamDecoderArena verifies StreamDecoder.SetArena: decoded string bodies
// bump into the caller's arena, values round-trip, the arena is reused across
// batches via the envelope pattern (arena.Reset + decoder.Reset), and the arena
// setting survives Reset.
func TestStreamDecoderArena(t *testing.T) {
	type rec struct {
		Name string `qdf:"name"`
		Tag  string `qdf:"tag"`
	}
	batch := []rec{{"alpha", "x"}, {"beta", "y"}, {"alpha", "z"}, {"gamma", "x"}}

	var w bytes.Buffer
	enc := NewStreamEncoderWith(&w, OptBalanced)
	for _, r := range batch {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	buf := append([]byte(nil), w.Bytes()...)

	a := NewArena()
	d := NewStreamDecoder(bytes.NewReader(buf))
	d.SetArena(a)
	defer d.Close()

	decodeAll := func() []rec {
		out := make([]rec, 0, len(batch))
		for range batch {
			var r rec
			if err := d.Decode(&r); err != nil {
				t.Fatal(err)
			}
			out = append(out, r)
		}
		return out
	}
	got := decodeAll()
	for i := range batch {
		if got[i] != batch[i] {
			t.Fatalf("arena decode [%d]=%v, want %v", i, got[i], batch[i])
		}
	}

	// Envelope reuse: Reset the arena (prior values dead) + the decoder, decode
	// again — the arena setting is preserved across Reset, and the result is
	// correct (no stale aliasing).
	a.Reset()
	d.Reset(bytes.NewReader(buf))
	got2 := decodeAll()
	for i := range batch {
		if got2[i] != batch[i] {
			t.Fatalf("after envelope reset [%d]=%v, want %v", i, got2[i], batch[i])
		}
	}
}
