package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleWithNoCopy decodes with zero-copy: the returned string / []byte
// fields alias the input buffer instead of being copied, cutting allocations
// to near zero (~1.7x faster on string-heavy batches).
//
// Lifetime contract: the decoded values are valid only while data stays alive
// and is never modified or reused. Never use WithNoCopy when data is a pooled
// or recycled buffer (e.g. an HTTP request body) — the values would silently
// corrupt. It is opt-in for this reason; the default Unmarshal copies.
func ExampleWithNoCopy() {
	type Event struct {
		Service string `qdf:"service"`
		Message string `qdf:"message"`
	}
	// data is owned here and outlives `out`, so noCopy is safe.
	data, _ := qdf.Marshal(&Event{Service: "api", Message: "ok"}, qdf.OptSpeed)

	var out Event
	if err := qdf.Unmarshal(data, &out, qdf.WithNoCopy()); err != nil {
		panic(err)
	}
	fmt.Println(out.Service, out.Message)
	// Output: api ok
}
