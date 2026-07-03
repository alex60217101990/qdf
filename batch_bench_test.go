package qdf

import (
	"runtime"
	"testing"
)

// benchBatchRows is the row count used for the decode-speed benches: large
// enough to clear columnarMinElems (so both UnmarshalBatch and Unmarshal take
// the columnar wire path) and to make per-op allocation counts stable.
const benchBatchRows = 1000

// BenchmarkBatchDecode compares UnmarshalBatch[batDoc] (pointer-free handles)
// against plain Unmarshal into []batSrc (real strings) decoding the identical
// columnar wire. Gate: handles must not be slower than the strings path —
// resolving Str lazily should make first-decode wall-clock at least parity.
func BenchmarkBatchDecode(b *testing.B) {
	src := mkBatSrc(benchBatchRows)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("handles", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			bat, err := UnmarshalBatch[batDoc](data)
			if err != nil {
				b.Fatal(err)
			}
			bat.Release()
		}
	})

	b.Run("strings", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out []batSrc
			if err := Unmarshal(data, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkBatchSteadyState decodes and immediately releases in a tight loop,
// the shape a server request handler runs in once slab pools are warm. Gate:
// steady-state allocs/op <= 2 (the value-return Batch[T] header build plus
// the Rows slice header — both stack-resident in the caller in real use; the
// benchmark's own escape to the benchmark loop keeps the count small
// but nonzero).
func BenchmarkBatchSteadyState(b *testing.B) {
	src := mkBatSrc(benchBatchRows)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		b.Fatal(err)
	}

	warm, err := UnmarshalBatch[batDoc](data) // prime the slab/rows pools
	if err != nil {
		b.Fatal(err)
	}
	warm.Release()

	b.ReportAllocs()
	for b.Loop() {
		bat, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			b.Fatal(err)
		}
		bat.Release()
	}
}

// BenchmarkBatchHeldGC is the 3.89x gate: hold K live decode results across a
// runtime.GC() scan and compare pointer-free Batch handles against
// pointer-dense []batSrc (real strings, one *string-header per Name field).
// The GC's mark phase must scan every pointer-containing word in the heap;
// Batch's Rows backing is noscan (Str/Bytes/Time/scalars only), so it should
// cost the collector a fraction of what K string-bearing slices cost.
func BenchmarkBatchHeldGC(b *testing.B) {
	src := mkBatSrc(1024) // rows per held batch — arbitrary but fixed, matches the brief's shape
	data, _ := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	const K = 256 // held batches: large enough to make GC scan cost dominate noise
	b.Run("handles", func(b *testing.B) {
		held := make([]Batch[batDoc], K)
		for i := range held {
			var err error
			if held[i], err = UnmarshalBatch[batDoc](data); err != nil {
				b.Fatal(err)
			}
		}
		// b.Loop() excludes the setup above and the Release teardown below
		// from the timed region — no manual Reset/StopTimer needed.
		for b.Loop() {
			runtime.GC()
		}
		for i := range held {
			held[i].Release()
		}
	})
	b.Run("strings", func(b *testing.B) {
		held := make([][]batSrc, K)
		for i := range held {
			if err := Unmarshal(data, &held[i]); err != nil {
				b.Fatal(err)
			}
		}
		for b.Loop() {
			runtime.GC()
		}
		runtime.KeepAlive(held)
	})
}
