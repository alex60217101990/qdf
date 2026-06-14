package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleArena shows the safe default pattern: one arena per epoch (here, one
// batch of messages). Every decoded string is bump-packed into the arena
// instead of a separate heap allocation. No Reset and no manual lifetime
// management are needed — the arena and its blocks are reclaimed by the GC once
// the decoded values are dropped.
func ExampleArena() {
	type Event struct {
		Service string `qdf:"service"`
		Level   string `qdf:"level"`
		Msg     string `qdf:"msg"`
	}

	// Pretend these arrived over the wire.
	msgs := make([][]byte, 3)
	for i, m := range []Event{
		{"api", "info", "request handled"},
		{"worker", "warn", "queue backlog"},
		{"edge", "error", "upstream timeout"},
	} {
		b, err := qdf.Marshal(m, qdf.OptSpeed)
		if err != nil {
			panic(err)
		}
		msgs[i] = b
	}

	a := qdf.NewArena() // one arena for this batch
	out := make([]Event, len(msgs))
	for i, msg := range msgs {
		if err := qdf.Unmarshal(msg, &out[i], qdf.WithArena(a)); err != nil {
			panic(err)
		}
	}
	// out's strings live in `a`; when `out` is dropped the GC frees the arena.

	for _, e := range out {
		fmt.Printf("%s/%s: %s\n", e.Service, e.Level, e.Msg)
	}
	// Output:
	// api/info: request handled
	// worker/warn: queue backlog
	// edge/error: upstream timeout
}

// ExampleArena_reset shows the max-performance pattern: reuse one arena across
// an unbounded stream, calling Reset at each epoch boundary for zero
// steady-state allocation.
//
// SAFETY: Reset rewinds the arena and the next decode overwrites its memory, so
// every value decoded before a Reset must be done being used before that Reset
// runs. Here each event is processed inside the loop body, before the next
// iteration's Reset.
func ExampleArena_reset() {
	type Event struct {
		Service string `qdf:"service"`
		Msg     string `qdf:"msg"`
	}

	stream := []Event{
		{"api", "ok"},
		{"worker", "retry"},
	}
	encoded := make([][]byte, len(stream))
	for i, e := range stream {
		b, _ := qdf.Marshal(e, qdf.OptSpeed)
		encoded[i] = b
	}

	a := qdf.NewArena()
	for _, msg := range encoded {
		a.Reset() // safe: the previous iteration's value is no longer used

		var ev Event
		if err := qdf.Unmarshal(msg, &ev, qdf.WithArena(a)); err != nil {
			panic(err)
		}
		fmt.Printf("%s: %s\n", ev.Service, ev.Msg)
	}
	// Output:
	// api: ok
	// worker: retry
}
