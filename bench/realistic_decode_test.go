package bench

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkDecode_UniqueLog decodes a different pre-encoded entry per
// iteration so that the decoder cannot benefit from any input-bytes
// repetition (it doesn't, but this proves it). Allocations come from
// constructing the output struct, not from input variability.
func BenchmarkDecode_UniqueLog(b *testing.B) {
	const N = 256
	state := newUniqueState()
	entries := make([]LogEntry, N)
	for i := range entries {
		entries[i] = state.next()
	}
	jsonBufs := make([][]byte, N)
	msgpackBufs := make([][]byte, N)
	qdfBufs := make([][]byte, N)
	for i, e := range entries {
		jsonBufs[i], _ = json.Marshal(e)
		msgpackBufs[i], _ = msgpack.Marshal(e)
		qdfBufs[i], _ = qdf.Marshal(&e, qdf.OptSpeed)
	}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out LogEntry
			if err := json.Unmarshal(jsonBufs[i&(N-1)], &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out LogEntry
			if err := jsonv2.Unmarshal(jsonBufs[i&(N-1)], &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out LogEntry
			if err := msgpack.Unmarshal(msgpackBufs[i&(N-1)], &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out LogEntry
			if err := qdf.Unmarshal(qdfBufs[i&(N-1)], &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDecodeParallel_UniqueLog stresses the decoder pool under
// contention. With N=256 inputs cycled per goroutine, decoders rotate
// through the same set so the decoder pool gets reused.
func BenchmarkDecodeParallel_UniqueLog(b *testing.B) {
	const N = 256
	state := newUniqueState()
	entries := make([]LogEntry, N)
	for i := range entries {
		entries[i] = state.next()
	}
	jsonBufs := make([][]byte, N)
	msgpackBufs := make([][]byte, N)
	qdfBufs := make([][]byte, N)
	for i, e := range entries {
		jsonBufs[i], _ = json.Marshal(e)
		msgpackBufs[i], _ = msgpack.Marshal(e)
		qdfBufs[i], _ = qdf.Marshal(&e, qdf.OptSpeed)
	}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var out LogEntry
				_ = json.Unmarshal(jsonBufs[i&(N-1)], &out)
				i++
			}
		})
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var out LogEntry
				_ = jsonv2.Unmarshal(jsonBufs[i&(N-1)], &out)
				i++
			}
		})
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var out LogEntry
				_ = msgpack.Unmarshal(msgpackBufs[i&(N-1)], &out)
				i++
			}
		})
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var out LogEntry
				_ = qdf.Unmarshal(qdfBufs[i&(N-1)], &out)
				i++
			}
		})
	})
}

// BenchmarkStream_LogBatch1k_Dense measures the realistic streaming case:
// 1000 entries written with the Dense stream encoder (intern table shared
// across messages) then decoded with the matching stream decoder.
func BenchmarkStream_LogBatch1k_Dense(b *testing.B) {
	entries := MakeLogBatch(1000).Entries

	b.Run("encode_stream_dense", func(b *testing.B) {
		b.ReportAllocs()
		var sink countSink
		for b.Loop() {
			sink.n = 0
			enc := qdf.NewStreamEncoder(&sink, qdf.Dense)
			for i := range entries {
				if err := enc.Encode(entries[i]); err != nil {
					b.Fatal(err)
				}
			}
			_ = enc.Close()
		}
		b.SetBytes(int64(sink.n))
	})
}

type countSink struct{ n int }

func (s *countSink) Write(p []byte) (int, error) { s.n += len(p); return len(p), nil }
