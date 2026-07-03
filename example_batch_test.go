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

// ExampleBatch_Release shows the streaming reuse pattern: decode, use,
// Release, decode again. Release returns the slab and the Rows backing to
// their sync.Pools, so once the pools are warm a decode/Release loop settles
// to a small constant allocation floor instead of growing with iteration
// count (measured 2 allocs/op steady-state — see BatchSteadyState in
// batch_bench_test.go). Handles from one iteration must never be resolved
// after that iteration's Release; each iteration below resolves its handle
// before releasing, which is the safe pattern.
func ExampleBatch_Release() {
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
		{ID: 3, Name: "gamma"},
	}
	data, err := qdf.Marshal(src, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}

	var total int64
	for range 3 { // simulates a streaming pipeline decoding the same message repeatedly
		b, err := qdf.UnmarshalBatch[Row](data)
		if err != nil {
			panic(err)
		}
		for _, r := range b.Rows {
			total += r.ID + int64(len(b.Str(r.Name)))
		}
		b.Release() // slab + Rows backing return to the pool for the next iteration
	}

	fmt.Println(total)
	// Output: 60
}

// ExampleBatch_Str shows handle resolve semantics: qdf.Str is an
// (offset, length) pair into the Batch's slab, not a Go string — Str
// resolves it to a zero-copy view valid until Release. Low-cardinality
// values are deduplicated by the wire's dict codec: every row holding the
// same distinct string shares the identical slab handle, so comparing two
// qdf.Str values directly (before ever calling Str) is a cheap, correct way
// to test "same value" without resolving either one.
func ExampleBatch_Str() {
	type Source struct {
		ID     int64  `qdf:"id"`
		Region string `qdf:"region"`
	}
	type Row struct {
		ID     int64   `qdf:"id"`
		Region qdf.Str `qdf:"region"`
	}

	// >= columnarMinElems rows so the wire is columnar and the region
	// column is dict-coded (few distinct values, many repeats).
	src := make([]Source, 20)
	for i := range src {
		src[i] = Source{ID: int64(i), Region: []string{"us-east", "eu-west"}[i%2]}
	}
	data, err := qdf.Marshal(src, qdf.OptBalanced|qdf.OptDense|qdf.OptShapeIntern)
	if err != nil {
		panic(err)
	}

	b, err := qdf.UnmarshalBatch[Row](data)
	if err != nil {
		panic(err)
	}
	defer b.Release()

	fmt.Println(b.Str(b.Rows[0].Region))
	fmt.Println(b.Str(b.Rows[1].Region))
	fmt.Println("rows 0 and 2 share a slab entry:", b.Rows[0].Region == b.Rows[2].Region)
	// Output:
	// us-east
	// eu-west
	// rows 0 and 2 share a slab entry: true
}
