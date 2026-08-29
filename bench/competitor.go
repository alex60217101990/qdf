package bench

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
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

	// Three JSON arms, because in Go 1.27 encoding/json IS json/v2 under
	// DefaultOptionsV1: the v1/v2 gap measures the price of the compatibility
	// options, not a faster engine. json-v2 runs v2's own defaults (no map-key
	// sorting, no HTML escaping, [] instead of null for a nil slice — wire
	// differences, so its size column is not interchangeable with v1's), and
	// json-v2-compat runs v2 asked for v1 semantics, whose output is
	// byte-identical to v1.
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

	jsonV2Bytes, _ := jsonv2.Marshal(value)
	b.Run("encode/json-v2", func(b *testing.B) {
		var buf []byte
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonV2Bytes)))
		for i := 0; i < b.N; i++ {
			buf, _ = jsonv2.Marshal(value)
		}
		_ = buf
		b.ReportMetric(float64(len(jsonV2Bytes)), "wire-B")
	})
	b.Run("decode/json-v2", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonV2Bytes)))
		for i := 0; i < b.N; i++ {
			out := newOut()
			_ = jsonv2.Unmarshal(jsonV2Bytes, out)
		}
	})

	b.Run("encode/json-v2-compat", func(b *testing.B) {
		var buf []byte
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBytes)))
		for i := 0; i < b.N; i++ {
			buf, _ = jsonv2.Marshal(value, json.DefaultOptionsV1())
		}
		_ = buf
		b.ReportMetric(float64(len(jsonBytes)), "wire-B")
	})
	b.Run("decode/json-v2-compat", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jsonBytes)))
		for i := 0; i < b.N; i++ {
			out := newOut()
			_ = jsonv2.Unmarshal(jsonBytes, out, json.DefaultOptionsV1())
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
