package qdf_test

import (
	"bytes"
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleStreamEncoder writes a batch of messages into a single
// io.Writer where the Dense-mode intern table, shape table, and
// predictors survive across calls. The first message of every
// shape pays for the key intern records; every subsequent
// message of the same shape emits values only.
//
// This is the form to reach for when piping events to a file or
// a network socket — per-message Marshal would reset the state
// table each call and lose the cross-message compression.
func ExampleStreamEncoder() {
	type Event struct {
		Service string `qdf:"service"`
		Status  int    `qdf:"status"`
	}

	var sink bytes.Buffer
	enc := qdf.NewStreamEncoderWith(&sink, qdf.OptBalanced)
	defer enc.Close() //nolint:errcheck // example

	events := []Event{
		{"billing", 200},
		{"billing", 200},
		{"billing", 500},
	}
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			panic(err)
		}
	}
	if err := enc.Flush(); err != nil {
		panic(err)
	}

	dec := qdf.NewStreamDecoder(&sink)
	for {
		var out Event
		if err := dec.Decode(&out); err != nil {
			break
		}
		fmt.Printf("%s=%d\n", out.Service, out.Status)
	}
	// Output:
	// billing=200
	// billing=200
	// billing=500
}

// ExampleStreamDecoder_SetNoCopy decodes a multi-message stream
// without copying the decoded strings out of the input buffer.
// The decoded values alias the io.Reader's payload; the caller
// must not retain the values past the next Decode call.
func ExampleStreamDecoder_SetNoCopy() {
	type Event struct {
		Service string `qdf:"service"`
	}

	// Build a 2-message stream.
	var sink bytes.Buffer
	enc := qdf.NewStreamEncoderWith(&sink, qdf.OptBalanced)
	_ = enc.Encode(Event{Service: "auth"})
	_ = enc.Encode(Event{Service: "auth"})
	_ = enc.Flush()
	_ = enc.Close()

	dec := qdf.NewStreamDecoder(&sink)
	dec.SetNoCopy(true) // aliased reads — zero allocations
	defer dec.Close()   //nolint:errcheck // example
	var ev Event
	for dec.Decode(&ev) == nil {
		fmt.Println(ev.Service)
	}
	// Output:
	// auth
	// auth
}

// ExampleStreamEncoder_Reset reuses ONE StreamEncoder across two independent
// streams via Reset, instead of constructing a fresh one per stream. The heavy
// per-stream construction (the intern table) is paid once; Reset clears the
// cross-message state so each stream stays independent, while keeping the
// configured Options and the grown buffers for reuse.
func ExampleStreamEncoder_Reset() {
	type Event struct {
		Service string `qdf:"service"`
		Status  int    `qdf:"status"`
	}

	enc := qdf.NewStreamEncoderWith(nil, qdf.OptBalanced)
	defer enc.Close() //nolint:errcheck // example

	encodeBatch := func(evs []Event) []byte {
		var sink bytes.Buffer
		enc.Reset(&sink) // new independent stream, reuse encoder + intern table
		for _, e := range evs {
			_ = enc.Encode(e)
		}
		_ = enc.Flush()
		return append([]byte(nil), sink.Bytes()...)
	}

	a := encodeBatch([]Event{{"billing", 200}, {"billing", 500}})
	b := encodeBatch([]Event{{"auth", 200}})

	dec := qdf.NewStreamDecoder(nil)
	defer dec.Close() //nolint:errcheck // example
	for _, stream := range [][]byte{a, b} {
		dec.Reset(bytes.NewReader(stream)) // reuse decoder + window across streams
		var ev Event
		for dec.Decode(&ev) == nil {
			fmt.Println(ev.Service, ev.Status)
		}
	}
	// Output:
	// billing 200
	// billing 500
	// auth 200
}

// ExampleStreamDecoder_SetArena decodes a stream with string bodies bump-packed
// into a caller-owned Arena instead of one heap allocation per value. The arena
// is Reset per envelope so its blocks are reused; the decoder is Reset alongside
// to cap the read window. Together they bound a long stream's memory.
func ExampleStreamDecoder_SetArena() {
	type Event struct {
		Service string `qdf:"service"`
	}

	var sink bytes.Buffer
	enc := qdf.NewStreamEncoderWith(&sink, qdf.OptBalanced)
	_ = enc.Encode(Event{Service: "billing"})
	_ = enc.Encode(Event{Service: "auth"})
	_ = enc.Flush()
	_ = enc.Close()
	stream := sink.Bytes()

	a := qdf.NewArena()
	dec := qdf.NewStreamDecoder(nil)
	dec.SetArena(a)   // copied string bodies live in the arena, not the heap
	defer dec.Close() //nolint:errcheck // example

	a.Reset()
	dec.Reset(bytes.NewReader(stream))
	var ev Event
	for dec.Decode(&ev) == nil {
		fmt.Println(ev.Service)
	}
	// Output:
	// billing
	// auth
}
