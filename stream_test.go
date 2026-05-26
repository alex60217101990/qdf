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
