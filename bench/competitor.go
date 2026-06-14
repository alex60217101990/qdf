package bench

import (
	"encoding/json"
	"testing"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

type qdfTier struct {
	name string
	opts qdf.Options
}

var qdfTiers = []qdfTier{
	{"speed", qdf.OptSpeed},
	{"balanced", qdf.OptBalanced},
	{"qpack", qdf.OptQPack},
	{"compression", qdf.OptCompression},
}

// runCodecMatrix benches encode+decode for json, msgpack, and every qdf tier
// over value. newOut returns a fresh decode target (*T). Emits ns/op,
// allocs/op (ReportAllocs), MB/s (SetBytes), and a wire-B size metric.
func runCodecMatrix[T any](b *testing.B, value T, newOut func() *T) {
	b.Helper()

	jsonBytes, _ := json.Marshal(value)
	b.Run("encode/json", func(b *testing.B) {
		var buf []byte
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBytes)))
		for i := 0; i < b.N; i++ {
			buf, _ = json.Marshal(value)
		}
		_ = buf
		b.ReportMetric(float64(len(jsonBytes)), "wire-B")
	})
	b.Run("decode/json", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBytes)))
		for i := 0; i < b.N; i++ {
			out := newOut()
			_ = json.Unmarshal(jsonBytes, out)
		}
	})

	mpBytes, _ := msgpack.Marshal(value)
	b.Run("encode/msgpack", func(b *testing.B) {
		var buf []byte
		b.ReportAllocs()
		b.SetBytes(int64(len(mpBytes)))
		for i := 0; i < b.N; i++ {
			buf, _ = msgpack.Marshal(value)
		}
		_ = buf
		b.ReportMetric(float64(len(mpBytes)), "wire-B")
	})
	b.Run("decode/msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(mpBytes)))
		for i := 0; i < b.N; i++ {
			out := newOut()
			_ = msgpack.Unmarshal(mpBytes, out)
		}
	})

	for _, tier := range qdfTiers {
		qb, err := qdf.Marshal(value, tier.opts)
		if err != nil {
			b.Fatalf("qdf.Marshal(%s): %v", tier.name, err)
		}
		// Fail loudly on a decode error instead of silently benchmarking a
		// half-finished, early-aborted decode: a broken decode does less work
		// and reports artificially low allocs/ns, which then shows up as a
		// phantom "regression" the moment the decode is fixed to run in full.
		if derr := qdf.Unmarshal(qb, newOut()); derr != nil {
			b.Fatalf("qdf.Unmarshal(%s): %v", tier.name, derr)
		}
		b.Run("encode/qdf_"+tier.name, func(b *testing.B) {
			var buf []byte
			b.ReportAllocs()
			b.SetBytes(int64(len(qb)))
			for i := 0; i < b.N; i++ {
				buf, _ = qdf.Marshal(value, tier.opts)
			}
			_ = buf
			b.ReportMetric(float64(len(qb)), "wire-B")
		})
		b.Run("decode/qdf_"+tier.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(qb)))
			for i := 0; i < b.N; i++ {
				out := newOut()
				_ = qdf.Unmarshal(qb, out)
			}
		})
	}
}
