package bench

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"math/rand/v2"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
)

// These benchmarks construct a *fresh* payload every iteration so that the
// pool wins observed in the standard benchmarks are not artificially boosted
// by encoding the same byte sequence over and over. They are the realistic
// numbers to quote for production traffic.

// uniqueLog produces an entry whose strings are different from prior calls.
type uniqueState struct {
	rng *rand.Rand
	n   int
}

func newUniqueState() *uniqueState {
	return &uniqueState{rng: rand.New(rand.NewPCG(7, 11))}
}

func (u *uniqueState) next() LogEntry {
	u.n++
	return LogEntry{
		Time:     MakeLogBatch(1).Entries[0].Time,
		Level:    randomHex(u.rng, 5),
		Service:  randomHex(u.rng, 7),
		Host:     randomHex(u.rng, 8),
		Region:   randomHex(u.rng, 9),
		TraceID:  randomHex(u.rng, 32),
		SpanID:   randomHex(u.rng, 16),
		Msg:      randomHex(u.rng, 24),
		Duration: u.rng.Float64() * 1000,
		Status:   200 + u.rng.IntN(300),
	}
}

func BenchmarkEncode_UniqueLog(b *testing.B) {
	state := newUniqueState()
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v := state.next()
			if _, err := json.Marshal(&v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v := state.next()
			if _, err := jsonv2.Marshal(&v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v := state.next()
			if _, err := msgpack.Marshal(&v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v := state.next()
			if _, err := qdf.Marshal(&v, qdf.OptSpeed); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// MixedTypes alternates between several payload shapes so that
// type-descriptor cache hits and pool reuse have to handle different
// reflect.Type kinds across consecutive iterations.
func BenchmarkEncode_MixedTypes(b *testing.B) {
	tiny := MakeTiny()
	flat := MakeFlat()
	nested := MakeNested()
	state := newUniqueState()
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			switch i & 3 {
			case 0:
				_, _ = json.Marshal(tiny)
			case 1:
				_, _ = json.Marshal(flat)
			case 2:
				_, _ = json.Marshal(nested)
			default:
				v := state.next()
				_, _ = json.Marshal(&v)
			}
		}
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			switch i & 3 {
			case 0:
				_, _ = jsonv2.Marshal(tiny)
			case 1:
				_, _ = jsonv2.Marshal(flat)
			case 2:
				_, _ = jsonv2.Marshal(nested)
			default:
				v := state.next()
				_, _ = jsonv2.Marshal(&v)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			switch i & 3 {
			case 0:
				_, _ = msgpack.Marshal(tiny)
			case 1:
				_, _ = msgpack.Marshal(flat)
			case 2:
				_, _ = msgpack.Marshal(nested)
			default:
				v := state.next()
				_, _ = msgpack.Marshal(&v)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			switch i & 3 {
			case 0:
				_, _ = qdf.Marshal(tiny, qdf.OptSpeed)
			case 1:
				_, _ = qdf.Marshal(flat, qdf.OptSpeed)
			case 2:
				_, _ = qdf.Marshal(nested, qdf.OptSpeed)
			default:
				v := state.next()
				_, _ = qdf.Marshal(&v, qdf.OptSpeed)
			}
		}
	})
}

// RandomSize encodes payloads of widely varying byte size so the pool's
// buffer-reuse heuristic gets stress-tested.
func BenchmarkEncode_RandomSize(b *testing.B) {
	rng := rand.New(rand.NewPCG(3, 5))
	sizes := []int{1, 10, 100, 1000}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n := sizes[rng.IntN(len(sizes))]
			v := MakeWide(n)
			if _, err := json.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n := sizes[rng.IntN(len(sizes))]
			v := MakeWide(n)
			if _, err := jsonv2.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n := sizes[rng.IntN(len(sizes))]
			v := MakeWide(n)
			if _, err := msgpack.Marshal(v); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n := sizes[rng.IntN(len(sizes))]
			v := MakeWide(n)
			if _, err := qdf.Marshal(v, qdf.OptSpeed); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// EncodeManyParallel exercises the sharded pool / sync.Pool under contention.
func BenchmarkEncodeParallel_UniqueLog(b *testing.B) {
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			st := newUniqueState()
			for pb.Next() {
				v := st.next()
				_, _ = json.Marshal(&v)
			}
		})
	})
	b.Run("json-v2", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			st := newUniqueState()
			for pb.Next() {
				v := st.next()
				_, _ = jsonv2.Marshal(&v)
			}
		})
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			st := newUniqueState()
			for pb.Next() {
				v := st.next()
				_, _ = msgpack.Marshal(&v)
			}
		})
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			st := newUniqueState()
			for pb.Next() {
				v := st.next()
				_, _ = qdf.Marshal(&v, qdf.OptSpeed)
			}
		})
	})
}
