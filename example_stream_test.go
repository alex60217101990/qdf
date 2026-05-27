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
