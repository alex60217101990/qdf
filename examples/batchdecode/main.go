// batchdecode compares plain Unmarshal against UnmarshalBatch on a held
// decode result.
//
// A decoded []struct{ ...string fields... } that you HOLD (a cache, an
// in-memory index, a streaming pipeline's working set) puts one pointer per
// string field per row into the heap — the garbage collector has to walk
// every one of them on every mark phase for as long as you keep the slice
// alive. UnmarshalBatch[T] decodes into a pointer-free Batch[T] instead:
// string/time.Time fields become qdf.Str/qdf.Time handles (an offset/length
// or two plain integers), so []T carries zero pointers and the collector
// skips scanning it entirely. See docs/BATCH-HANDLES.md for the full design
// and the measured numbers this example's shape is drawn from.
//
//	go run ./examples/batchdecode
package main

import (
	"fmt"
	"time"

	"github.com/alex60217101990/qdf"
)

// Source is the wire shape: everything is a normal Go type, encoded and
// decoded like any other qdf struct.
type Source struct {
	TS     time.Time `qdf:"ts"`
	Region string    `qdf:"region"` // low-cardinality: a handful of repeated values
	Msg    string    `qdf:"msg"`    // high-cardinality: distinct per row
	ID     int64     `qdf:"id"`
	Val    float64   `qdf:"val"`
}

// Row is the SAME wire shape, but every pointer-carrying field is spelled
// as its pointer-free handle type. UnmarshalBatch validates this once per
// type and rejects anything that isn't strictly pointer-free (see
// docs/BATCH-HANDLES.md's type-rules table).
type Row struct {
	ID     int64    `qdf:"id"`
	Region qdf.Str  `qdf:"region"`
	Msg    qdf.Str  `qdf:"msg"`
	TS     qdf.Time `qdf:"ts"`
	Val    float64  `qdf:"val"`
}

// regions is deliberately small: a low-cardinality string column is where
// the dict codec (tagColStrDict family) pays off hardest for UnmarshalBatch
// — each DISTINCT value is copied into the slab once, not once per row.
var regions = []string{"us-east", "us-west", "eu-central", "ap-south"}

func main() {
	const n = 1000 // >= columnarMinElems (16), so the wire is columnar
	src := make([]Source, n)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range src {
		src[i] = Source{
			ID:     int64(i),
			Region: regions[i%len(regions)],
			Msg:    fmt.Sprintf("event %d processed in region shard %d", i, i%len(regions)), // high-cardinality
			TS:     base.Add(time.Duration(i) * time.Second),
			Val:    float64(i) * 0.5,
		}
	}

	data, err := qdf.Marshal(src, qdf.OptBalanced|qdf.OptDense|qdf.OptShapeIntern)
	if err != nil {
		panic(err)
	}
	fmt.Printf("encoded: %d rows, %d bytes\n\n", n, len(data))

	// --- Decode A: plain Unmarshal into the string-bearing struct. -----
	// Every Region/Msg field becomes a real Go string: a heap pointer the
	// GC must walk on every mark phase for as long as `plain` is held.
	var plain []Source
	t0 := time.Now()
	if err := qdf.Unmarshal(data, &plain); err != nil {
		panic(err)
	}
	plainTime := time.Since(t0)

	// --- Decode B: UnmarshalBatch into the pointer-free handle struct. -
	// Region/Msg/TS become qdf.Str/qdf.Time: plain integers, no pointers.
	// []Row is GC-noscan regardless of row count. Str resolution is lazy —
	// decode itself does not materialize a Go string until you call b.Str.
	t1 := time.Now()
	b, err := qdf.UnmarshalBatch[Row](data)
	if err != nil {
		panic(err)
	}
	batchTime := time.Since(t1)
	defer b.Release()

	fmt.Println("decode time (single run, NOT a benchmark — see batch_bench_test.go for real numbers):")
	fmt.Printf("  Unmarshal:      %v\n", plainTime)
	fmt.Printf("  UnmarshalBatch: %v\n\n", batchTime)

	// Prove the handles resolve to the same data plain Unmarshal produced.
	fmt.Println("spot-check (handles resolve to real data):")
	for _, i := range []int{0, 500, 999} {
		fmt.Printf("  row %-3d region=%-10s msg=%q val=%v\n",
			i, b.Str(b.Rows[i].Region), b.Str(b.Rows[i].Msg), b.Rows[i].Val)
	}
	fmt.Println()

	// Low-cardinality columns are free deduplication: every row sharing a
	// Region value shares the SAME slab handle (dict codec, one slab copy
	// per distinct value — not per row). Rows 0 and 4 share "us-east".
	sameHandle := b.Rows[0].Region == b.Rows[4].Region
	fmt.Printf("rows 0 and 4 share one slab entry for %q: %v\n\n", b.Str(b.Rows[0].Region), sameHandle)

	// --- Streaming reuse: decode -> use -> Release -> decode again. -----
	// This is the shape a long-running pipeline actually runs in: each
	// iteration's slab and Rows backing come from a sync.Pool, so once the
	// pool is warm this loop settles to a small, constant allocation floor
	// instead of growing with iteration count (see BatchSteadyState in
	// batch_bench_test.go: 2 allocs/op once warm).
	const iterations = 1000
	var checksum int64
	for range iterations {
		rb, err := qdf.UnmarshalBatch[Row](data)
		if err != nil {
			panic(err)
		}
		checksum += rb.Rows[0].ID + int64(len(rb.Rows))
		rb.Release() // slab + Rows backing return to the pool for the next iteration
	}
	fmt.Printf("streaming reuse: %d decode+Release iterations, checksum=%d (proves every iteration decoded real data)\n",
		iterations, checksum)
}
