package bench

import (
	"encoding/json"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
)

// QPack head-to-head: encode and decode a payload that exercises the
// QPack codecs (numeric slices). Compared against JSON, msgpack, and the
// legacy qdf_fast path so the wins are visible.

type qpackPayload struct {
	Bools  []bool    `json:"bools" msgpack:"bools" qdf:"bools"`
	IDs    []uint64  `json:"ids" msgpack:"ids" qdf:"ids"`
	TSDiff []int64   `json:"ts" msgpack:"ts" qdf:"ts"`
	Vec64  []float64 `json:"vec64" msgpack:"vec64" qdf:"vec64"`
}

func samplePayload() qpackPayload {
	p := qpackPayload{
		Bools:  make([]bool, 256),
		IDs:    make([]uint64, 512),
		TSDiff: make([]int64, 512),
		Vec64:  make([]float64, 256),
	}
	for i := range p.Bools {
		p.Bools[i] = i%3 == 0
	}
	for i := range p.IDs {
		p.IDs[i] = 1_700_000_000 + uint64(i)*5 // monotonic stride
	}
	for i := range p.TSDiff {
		p.TSDiff[i] = -1000 + int64(i)
	}
	for i := range p.Vec64 {
		p.Vec64[i] = float64(i) * 0.5
	}
	return p
}

func BenchmarkQPack_Encode(b *testing.B) {
	v := samplePayload()
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = json.Marshal(v)
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = msgpack.Marshal(v)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptSpeed)
		}
	})
	b.Run("qdf_qpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptQPack)
		}
	})
	b.Run("qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptBalanced)
		}
	})
}

func BenchmarkQPack_Decode(b *testing.B) {
	v := samplePayload()
	jb, _ := json.Marshal(v)
	mb, _ := msgpack.Marshal(v)
	fb, _ := qdf.Marshal(v, qdf.OptSpeed)
	qb, _ := qdf.Marshal(v, qdf.OptQPack)
	db, _ := qdf.Marshal(v, qdf.OptBalanced)
	b.Logf("size json=%d msgpack=%d qdf_fast=%d qdf_qpack=%d qdf_dense=%d", len(jb), len(mb), len(fb), len(qb), len(db))
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out qpackPayload
			_ = json.Unmarshal(jb, &out)
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out qpackPayload
			_ = msgpack.Unmarshal(mb, &out)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out qpackPayload
			_ = qdf.Unmarshal(fb, &out)
		}
	})
	b.Run("qdf_qpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out qpackPayload
			_ = qdf.Unmarshal(qb, &out)
		}
	})
	b.Run("qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out qpackPayload
			_ = qdf.Unmarshal(db, &out)
		}
	})
}
