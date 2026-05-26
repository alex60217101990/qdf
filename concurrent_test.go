package qdf

import (
	"bytes"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

// TestPool_ConcurrentEncoders runs many goroutines that share the encoder
// pool. Detects races (run with -race), state-table leakage between
// callers, and pool-key bugs.
func TestPool_ConcurrentEncoders(t *testing.T) {
	const goroutines = 32
	const itersPerG = 500

	type payload struct {
		A int      `qdf:"a"`
		B string   `qdf:"b"`
		C []string `qdf:"c"`
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	var errCount atomic.Int64
	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			for i := range itersPerG {
				in := payload{A: id*1000 + i, B: "goroutine-" + string(rune('A'+id%26)), C: []string{"one", "two", "three"}}
				b, err := Marshal(in, OptSpeed)
				if err != nil {
					errCount.Add(1)
					return
				}
				var out payload
				if err := Unmarshal(b, &out); err != nil {
					errCount.Add(1)
					return
				}
				if !reflect.DeepEqual(in, out) {
					errCount.Add(1)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if errCount.Load() != 0 {
		t.Fatalf("%d errors across %d goroutines", errCount.Load(), goroutines)
	}
}

// TestDense_NoCrossCallStateBleed asserts that two consecutive Dense
// encodes do NOT share intern IDs from the previous call, even though the
// pooled encoder is reused. A bug in Reset would let the first message's
// state-table leak into the second, producing a decode that crashes when
// the second decoder hits an unknown state ID.
func TestDense_NoCrossCallStateBleed(t *testing.T) {
	type kv struct {
		K string `qdf:"k"`
		V string `qdf:"v"`
	}
	for i := range 1000 {
		in1 := kv{K: "alpha", V: "beta"}
		in2 := kv{K: "gamma", V: "delta"}
		b1, _ := Marshal(in1, OptBalanced)
		b2, _ := Marshal(in2, OptBalanced)
		var out1, out2 kv
		if err := Unmarshal(b1, &out1); err != nil {
			t.Fatalf("iter %d: decode b1: %v", i, err)
		}
		if err := Unmarshal(b2, &out2); err != nil {
			t.Fatalf("iter %d: decode b2: %v", i, err)
		}
		if out1 != in1 || out2 != in2 {
			t.Fatalf("iter %d: cross-call bleed: out1=%+v out2=%+v", i, out1, out2)
		}
	}
}

// TestRace_AppendMarshal ensures the AppendMarshal path is safe under
// concurrent use, including the detach-buffer trick that swaps enc.buf for
// nil before returning to the pool.
func TestRace_AppendMarshal(t *testing.T) {
	const goroutines = 16
	const itersPerG = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			scratch := make([]byte, 0, 64)
			for i := range itersPerG {
				out, err := AppendMarshal(scratch[:0], i, OptSpeed)
				if err != nil {
					t.Errorf("append: %v", err)
					return
				}
				var v int
				if err := Unmarshal(out, &v); err != nil {
					t.Errorf("decode: %v", err)
					return
				}
				if v != i {
					t.Errorf("mismatch: %d vs %d", v, i)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// TestStream_MixedMessageTypes verifies that a Dense stream encoder + the
// matching stream decoder can carry multiple distinct message types and
// preserve every value through round-trip.
func TestStream_MixedMessageTypes(t *testing.T) {
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	defer enc.Close()

	type Sale struct {
		SKU string  `qdf:"sku"`
		Qty int     `qdf:"qty"`
		Per float64 `qdf:"per"`
	}
	type Sig struct {
		Service string `qdf:"service"`
		Code    int    `qdf:"code"`
	}
	sales := []Sale{
		{"sku-001", 3, 9.99},
		{"sku-002", 1, 19.99},
		{"sku-001", 5, 9.99}, // repeats SKU
	}
	sigs := []Sig{
		{"api", 200},
		{"api", 500}, // repeats service
	}

	// Interleave the writes so the intern table sees varied keys.
	for i := range sales {
		if err := enc.Encode(sales[i]); err != nil {
			t.Fatal(err)
		}
		if i < len(sigs) {
			if err := enc.Encode(sigs[i]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := enc.Flush(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()

	gotSales := make([]Sale, 0, len(sales))
	gotSigs := make([]Sig, 0, len(sigs))
	for i := range sales {
		var s Sale
		if err := dec.Decode(&s); err != nil {
			t.Fatalf("decode sale %d: %v", i, err)
		}
		gotSales = append(gotSales, s)
		if i < len(sigs) {
			var sg Sig
			if err := dec.Decode(&sg); err != nil {
				t.Fatalf("decode sig %d: %v", i, err)
			}
			gotSigs = append(gotSigs, sg)
		}
	}
	if !reflect.DeepEqual(sales, gotSales) {
		t.Fatalf("sales mismatch:\nwant=%+v\n got=%+v", sales, gotSales)
	}
	if !reflect.DeepEqual(sigs, gotSigs) {
		t.Fatalf("sigs mismatch:\nwant=%+v\n got=%+v", sigs, gotSigs)
	}
}
