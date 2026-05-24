package bench

import (
	"encoding/json"
	"runtime"
	"strconv"
	"testing"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// BenchmarkDecode_MapHeavy_DistinctValues forces the map to have unique
// per-call values but stable keys. Common shape for analytics: keys are
// schema-defined (low cardinality), values vary per row. Repeated keys
// across calls benefit from decoder-side key interning when the decoder
// is reused (it is — qdf has a sync.Pool).
func BenchmarkDecode_MapHeavy_RepeatedKeys(b *testing.B) {
	// Build N=1000 different payloads. Each shares the SAME 20 keys but
	// has DIFFERENT values. Across iterations the decoder sees the same
	// keys over and over.
	const N = 1000
	keys := make([]string, 20)
	for i := range keys {
		keys[i] = "k" + strconv.Itoa(i)
	}
	payloads := make([]map[string]string, N)
	for i := range payloads {
		m := make(map[string]string, 20)
		for j, k := range keys {
			m[k] = "v" + strconv.Itoa(i*100+j)
		}
		payloads[i] = m
	}
	jsonBufs := make([][]byte, N)
	msgpackBufs := make([][]byte, N)
	qdfBufs := make([][]byte, N)
	for i, p := range payloads {
		jsonBufs[i], _ = json.Marshal(p)
		msgpackBufs[i], _ = msgpack.Marshal(p)
		qdfBufs[i], _ = qdf.Marshal(p)
	}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out map[string]string
			_ = json.Unmarshal(jsonBufs[i%N], &out)
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out map[string]string
			_ = msgpack.Unmarshal(msgpackBufs[i%N], &out)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out map[string]string
			_ = qdf.Unmarshal(qdfBufs[i%N], &out)
		}
	})
}

// BenchmarkMemory_DecodeLogBatch1k_Bytes measures *total* heap bytes
// allocated across 100 decode runs, not per-op steady state. This is
// the metric that actually matters for GC pressure in production
// services.
func BenchmarkMemory_DecodeLogBatch1k_Bytes(b *testing.B) {
	batch := MakeLogBatch(1000)
	jsonBytes, _ := json.Marshal(batch)
	msgpackBytes, _ := msgpack.Marshal(batch)
	qdfBytes, _ := qdf.Marshal(batch)

	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		var ms runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms)
		startAllocs := ms.TotalAlloc
		const runs = 100
		for i := 0; i < runs; i++ {
			var out LogBatch
			_ = json.Unmarshal(jsonBytes, &out)
		}
		runtime.ReadMemStats(&ms)
		b.ReportMetric(float64(ms.TotalAlloc-startAllocs)/runs, "B/decode")
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		var ms runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms)
		startAllocs := ms.TotalAlloc
		const runs = 100
		for i := 0; i < runs; i++ {
			var out LogBatch
			_ = msgpack.Unmarshal(msgpackBytes, &out)
		}
		runtime.ReadMemStats(&ms)
		b.ReportMetric(float64(ms.TotalAlloc-startAllocs)/runs, "B/decode")
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		var ms runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms)
		startAllocs := ms.TotalAlloc
		const runs = 100
		for i := 0; i < runs; i++ {
			var out LogBatch
			_ = qdf.Unmarshal(qdfBytes, &out)
		}
		runtime.ReadMemStats(&ms)
		b.ReportMetric(float64(ms.TotalAlloc-startAllocs)/runs, "B/decode")
	})
}

// BenchmarkDecode_MapStringAny_RepeatedKeys exercises the interface{}
// path (decodeAny → map[string]any) where intern is most valuable.
func BenchmarkDecode_MapStringAny_RepeatedKeys(b *testing.B) {
	const N = 200
	keys := []string{"id", "name", "type", "status", "region", "service", "version", "host"}
	payloads := make([]map[string]any, N)
	for i := range payloads {
		m := make(map[string]any, len(keys))
		for _, k := range keys {
			m[k] = i // values change, keys constant
		}
		payloads[i] = m
	}
	jsonBufs := make([][]byte, N)
	qdfBufs := make([][]byte, N)
	for i, p := range payloads {
		jsonBufs[i], _ = json.Marshal(p)
		qdfBufs[i], _ = qdf.Marshal(p)
	}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out map[string]any
			_ = json.Unmarshal(jsonBufs[i%N], &out)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			var out map[string]any
			_ = qdf.Unmarshal(qdfBufs[i%N], &out)
		}
	})
}
