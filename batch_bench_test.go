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
		for i := 0; i < b.N; i++ {
			bat, err := UnmarshalBatch[batDoc](data)
			if err != nil {
				b.Fatal(err)
			}
			bat.Release()
		}
	})

	b.Run("strings", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
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
// benchmark's own escape to the interface{} b.N loop keeps the count small
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
	for i := 0; i < b.N; i++ {
		bat, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			b.Fatal(err)
		}
		bat.Release()
	}
}

// BenchmarkBatchHeldGC is the 5.2x gate: hold K live decode results across a
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
			held[i], _ = UnmarshalBatch[batDoc](data)
		}
		b.ResetTimer()
		for range b.N {
			runtime.GC()
		}
		b.StopTimer()
		for i := range held {
			held[i].Release()
		}
	})
	b.Run("strings", func(b *testing.B) {
		held := make([][]batSrc, K)
		for i := range held {
			_ = Unmarshal(data, &held[i])
		}
		b.ResetTimer()
		for range b.N {
			runtime.GC()
		}
		runtime.KeepAlive(held)
	})
}
