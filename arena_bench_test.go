package qdf

import (
	"testing"
)

// benchLog mirrors a telemetry record but does NOT implement Unmarshaler, so
// Unmarshal takes the reflect path where WithArena applies. Values are unique
// (worst case: every field copied, no interning).
type benchLog struct {
	Level   string `qdf:"level"`
	Service string `qdf:"service"`
	Host    string `qdf:"host"`
	Region  string `qdf:"region"`
	TraceID string `qdf:"trace_id"`
	SpanID  string `qdf:"span_id"`
	Msg     string `qdf:"msg"`
	Status  int    `qdf:"status"`
}

func benchLogBytes() []byte {
	v := benchLog{
		Level: "abc12", Service: "srv7xyz", Host: "host-abcd", Region: "eu-west-1x",
		TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef",
		Msg: "request handled in time", Status: 200,
	}
	b, err := Marshal(&v, OptSpeed)
	if err != nil {
		panic(err)
	}
	return b
}

func BenchmarkArenaOff(b *testing.B) {
	src := benchLogBytes()
	b.ReportAllocs()
	for b.Loop() {
		var v benchLog
		if err := Unmarshal(src, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArenaOn(b *testing.B) {
	src := benchLogBytes()
	a := NewArena()
	b.ReportAllocs()
	for b.Loop() {
		a.Reset() // prior iteration's value is dead — safe to rewind
		var v benchLog
		if err := Unmarshal(src, &v, WithArena(a)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArenaOffPar(b *testing.B) {
	src := benchLogBytes()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			var v benchLog
			if err := Unmarshal(src, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkArenaOnPar(b *testing.B) {
	src := benchLogBytes()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		a := NewArena() // one arena per goroutine
		for p.Next() {
			a.Reset()
			var v benchLog
			if err := Unmarshal(src, &v, WithArena(a)); err != nil {
				b.Fatal(err)
			}
		}
	})
}
