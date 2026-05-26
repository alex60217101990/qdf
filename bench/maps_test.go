package bench

import (
	"encoding/json"
	"strconv"
	"testing"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// Map-heavy realistic payload: a service-attribute record common in tracing
// / OpenTelemetry-style spans. Mostly map[string]string + map[string]int.

type Attrs struct {
	Service string            `json:"service" msgpack:"service" qdf:"service"`
	Span    string            `json:"span" msgpack:"span" qdf:"span"`
	Tags    map[string]string `json:"tags" msgpack:"tags" qdf:"tags"`
	Counts  map[string]int    `json:"counts" msgpack:"counts" qdf:"counts"`
}

func makeAttrs(n int) Attrs {
	tags := make(map[string]string, n)
	counts := make(map[string]int, n)
	for i := 0; i < n; i++ {
		k := "k" + strconv.Itoa(i)
		tags[k] = "v" + strconv.Itoa(i)
		counts[k] = i
	}
	return Attrs{
		Service: "checkout",
		Span:    "process_payment",
		Tags:    tags,
		Counts:  counts,
	}
}

func BenchmarkEncode_MapHeavy(b *testing.B) {
	v := makeAttrs(20)
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
			if _, err := qdf.Marshal(v, qdf.OptSpeed); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(v, qdf.OptBalanced); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecode_MapHeavy(b *testing.B) {
	v := makeAttrs(20)
	jb, _ := json.Marshal(v)
	mb, _ := msgpack.Marshal(v)
	qb, _ := qdf.Marshal(v, qdf.OptSpeed)
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Attrs
			if err := json.Unmarshal(jb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Attrs
			if err := msgpack.Unmarshal(mb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out Attrs
			if err := qdf.Unmarshal(qb, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}
