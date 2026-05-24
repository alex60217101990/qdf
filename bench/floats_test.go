package bench

import (
	"encoding/json"
	"testing"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// Float-slice heavy payload — what you'd see in ML feature vectors,
// embeddings, signal processing telemetry.

type Vector struct {
	ID  string    `json:"id" msgpack:"id" qdf:"id"`
	Vec []float32 `json:"vec" msgpack:"vec" qdf:"vec"`
}

type DoubleVector struct {
	ID  string    `json:"id" msgpack:"id" qdf:"id"`
	Vec []float64 `json:"vec" msgpack:"vec" qdf:"vec"`
}

func makeVector(n int) Vector {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32(i) * 0.5
	}
	return Vector{ID: "embedding-512", Vec: v}
}

func makeDoubleVector(n int) DoubleVector {
	v := make([]float64, n)
	for i := range v {
		v[i] = float64(i) * 0.5
	}
	return DoubleVector{ID: "embedding-512", Vec: v}
}

func BenchmarkEncode_Float32Vec512(b *testing.B) {
	v := makeVector(512)
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := json.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := msgpack.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkEncode_Float64Vec512(b *testing.B) {
	v := makeDoubleVector(512)
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := json.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := msgpack.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecode_Float32Vec512(b *testing.B) {
	v := makeVector(512)
	jb, _ := json.Marshal(v)
	mb, _ := msgpack.Marshal(v)
	qb, _ := qdf.Marshal(v)
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Vector
			if err := json.Unmarshal(jb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Vector
			if err := msgpack.Unmarshal(mb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Vector
			if err := qdf.Unmarshal(qb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}
