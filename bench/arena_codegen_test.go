package bench

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

func leArenaBytes() []byte {
	st := newUniqueState()
	e := st.next()
	b, err := qdf.Marshal(&e, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}
	return b
}

// Codegen path (LogEntry implements UnmarshalerArena): decode with vs without
// an epoch arena, unique strings (worst case: every field copied).
func BenchmarkLEArenaOff(b *testing.B) {
	src := leArenaBytes()
	b.ReportAllocs()
	for b.Loop() {
		var v LogEntry
		if err := qdf.Unmarshal(src, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLEArenaOn(b *testing.B) {
	src := leArenaBytes()
	a := qdf.NewArena()
	b.ReportAllocs()
	for b.Loop() {
		a.Reset()
		var v LogEntry
		if err := qdf.Unmarshal(src, &v, qdf.WithArena(a)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLEArenaOffPar(b *testing.B) {
	src := leArenaBytes()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			var v LogEntry
			if err := qdf.Unmarshal(src, &v); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkLEArenaOnPar(b *testing.B) {
	src := leArenaBytes()
	b.ReportAllocs()
	b.RunParallel(func(p *testing.PB) {
		a := qdf.NewArena()
		for p.Next() {
			a.Reset()
			var v LogEntry
			if err := qdf.Unmarshal(src, &v, qdf.WithArena(a)); err != nil {
				b.Fatal(err)
			}
		}
	})
}
