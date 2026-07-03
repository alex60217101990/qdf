package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleUnmarshalBatch decodes into a pointer-free Batch[T]: string fields
// become qdf.Str handles (an offset/length pair into one pooled slab) instead
// of real Go strings, so T is GC-noscan. This matters for HELD batches — a
// cache, an index, a streaming pipeline that decodes in a loop — where the
// collector would otherwise scan every string pointer on every GC cycle.
//
// Measured (i7-9750H, 1000-row columnar batch): holding 256 decoded batches
// across a GC scan is ~3.89x cheaper than holding []struct with real strings;
// decode itself is ~1.8x faster than plain Unmarshal (Str resolution is lazy);
// steady-state decode+Release is 2 allocs/op.
//
// Use it for held batches / release-and-reuse streaming. For a one-shot
// decode you unmarshal once and discard, plain Unmarshal is simpler.
//
// Debug builds (-race or -tags qdfdebug) panic on a stale handle used after
// Release; in production builds use-after-Release is undefined, the same
// documented contract as WithNoCopy.
func ExampleUnmarshalBatch() {
	type Source struct {
		ID   int64  `qdf:"id"`
		Name string `qdf:"name"`
	}
	type Row struct {
		ID   int64   `qdf:"id"`
		Name qdf.Str `qdf:"name"`
	}

	src := []Source{
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "beta"},
	}
	data, err := qdf.Marshal(src, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}

	b, err := qdf.UnmarshalBatch[Row](data)
	if err != nil {
		panic(err)
	}
	defer b.Release()

	fmt.Println(b.Str(b.Rows[0].Name))
	// Output: alpha
}
