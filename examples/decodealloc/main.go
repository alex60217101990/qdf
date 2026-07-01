// decodealloc shows the decode-side allocation lever. Decode is usually
// allocation/GC-bound, and the dominant cost is copying each decoded string out
// of the input buffer onto the heap. A decode Arena moves those strings into one
// bump-allocated region, collapsing thousands of tiny heap allocations into a
// handful of big ones — freed together when the batch is dropped.
//
//	go run ./examples/decodealloc
package main

import (
	"fmt"
	"runtime"

	"github.com/alex60217101990/qdf"
)

type Record struct {
	ID      string `qdf:"id"`
	Service string `qdf:"service"`
	Message string `qdf:"message"`
}

func main() {
	const n = 2000
	msgs := make([][]byte, n)
	for i := range msgs {
		b, err := qdf.Marshal(Record{
			ID:      fmt.Sprintf("req-%06d", i),
			Service: "api",
			Message: "request handled",
		}, qdf.OptSpeed)
		if err != nil {
			panic(err)
		}
		msgs[i] = b
	}

	// Default decode: each message's strings are copied to the heap.
	def := mallocs(func() {
		var r Record
		for _, m := range msgs {
			if err := qdf.Unmarshal(m, &r); err != nil {
				panic(err)
			}
		}
	})

	// Arena decode: strings alias one bump-allocated region for the batch.
	arena := mallocs(func() {
		a := qdf.NewArena()
		var r Record
		for _, m := range msgs {
			if err := qdf.Unmarshal(m, &r, qdf.WithArena(a)); err != nil {
				panic(err)
			}
		}
	})

	fmt.Printf("decode %d messages:\n", n)
	fmt.Printf("  default: %6d heap allocations\n", def)
	fmt.Printf("  arena:   %6d heap allocations  (%.0f%% fewer)\n",
		arena, 100*(1-float64(arena)/float64(def)))
}

// mallocs reports the number of heap allocations f performs.
func mallocs(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.Mallocs - before.Mallocs
}
