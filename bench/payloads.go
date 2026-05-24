// Package bench provides shared payload types and helpers for cross-format
// benchmarks.
package bench

import (
	"math/rand/v2"
	"time"
)

// Tiny is the smallest realistic message.
type Tiny struct {
	ID   int    `json:"id" msgpack:"id" qdf:"id"`
	Name string `json:"name" msgpack:"name" qdf:"name"`
}

// Flat is a 20-field struct of mixed primitives, representative of an API
// response object.
type Flat struct {
	A int     `json:"a" msgpack:"a" qdf:"a"`
	B int64   `json:"b" msgpack:"b" qdf:"b"`
	C uint32  `json:"c" msgpack:"c" qdf:"c"`
	D float64 `json:"d" msgpack:"d" qdf:"d"`
	E float32 `json:"e" msgpack:"e" qdf:"e"`
	F bool    `json:"f" msgpack:"f" qdf:"f"`
	G string  `json:"g" msgpack:"g" qdf:"g"`
	H string  `json:"h" msgpack:"h" qdf:"h"`
	I string  `json:"i" msgpack:"i" qdf:"i"`
	J int     `json:"j" msgpack:"j" qdf:"j"`
	K int     `json:"k" msgpack:"k" qdf:"k"`
	L int     `json:"l" msgpack:"l" qdf:"l"`
	M string  `json:"m" msgpack:"m" qdf:"m"`
	N string  `json:"n" msgpack:"n" qdf:"n"`
	O bool    `json:"o" msgpack:"o" qdf:"o"`
	P bool    `json:"p" msgpack:"p" qdf:"p"`
	Q float64 `json:"q" msgpack:"q" qdf:"q"`
	R float64 `json:"r" msgpack:"r" qdf:"r"`
	S int     `json:"s" msgpack:"s" qdf:"s"`
	T int     `json:"t" msgpack:"t" qdf:"t"`
}

// Nested is a 4-level nested structure.
type Nested struct {
	L1 struct {
		Tag string `json:"tag" msgpack:"tag" qdf:"tag"`
		L2  struct {
			Tag string `json:"tag" msgpack:"tag" qdf:"tag"`
			L3  struct {
				Tag string `json:"tag" msgpack:"tag" qdf:"tag"`
				L4  struct {
					Tag   string  `json:"tag" msgpack:"tag" qdf:"tag"`
					Value float64 `json:"value" msgpack:"value" qdf:"value"`
				} `json:"l4" msgpack:"l4" qdf:"l4"`
			} `json:"l3" msgpack:"l3" qdf:"l3"`
		} `json:"l2" msgpack:"l2" qdf:"l2"`
	} `json:"l1" msgpack:"l1" qdf:"l1"`
}

// Deep is a 16-level deep linked list of structs.
type Deep struct {
	V    int   `json:"v" msgpack:"v" qdf:"v"`
	Next *Deep `json:"next,omitempty" msgpack:"next,omitempty" qdf:"next"`
}

// LogEntry mirrors a typical structured log line.
type LogEntry struct {
	Time     time.Time `json:"time" msgpack:"time" qdf:"time"`
	Level    string    `json:"level" msgpack:"level" qdf:"level"`
	Service  string    `json:"service" msgpack:"service" qdf:"service"`
	Host     string    `json:"host" msgpack:"host" qdf:"host"`
	Region   string    `json:"region" msgpack:"region" qdf:"region"`
	TraceID  string    `json:"trace_id" msgpack:"trace_id" qdf:"trace_id"`
	SpanID   string    `json:"span_id" msgpack:"span_id" qdf:"span_id"`
	Msg      string    `json:"msg" msgpack:"msg" qdf:"msg"`
	Duration float64   `json:"duration" msgpack:"duration" qdf:"duration"`
	Status   int       `json:"status" msgpack:"status" qdf:"status"`
}

// LogBatch is a 1000-entry log batch — the canonical "where Dense wins" case.
type LogBatch struct {
	Entries []LogEntry `json:"entries" msgpack:"entries" qdf:"entries"`
}

// Wide is a 1k-element slice of Flat.
type Wide struct {
	Items []Flat `json:"items" msgpack:"items" qdf:"items"`
}

// MakeTiny returns a single Tiny payload.
func MakeTiny() Tiny { return Tiny{ID: 42, Name: "alice"} }

// MakeFlat returns a single Flat payload.
func MakeFlat() Flat {
	return Flat{
		A: 1, B: 1234567890, C: 65535, D: 3.14159, E: 2.71828,
		F: true, G: "hello", H: "world", I: "qdf-test",
		J: 100, K: 200, L: 300, M: "alpha", N: "beta",
		O: false, P: true, Q: 1e10, R: 1e-10, S: -42, T: 0,
	}
}

// MakeNested returns a single Nested payload.
func MakeNested() Nested {
	var n Nested
	n.L1.Tag = "level1"
	n.L1.L2.Tag = "level2"
	n.L1.L2.L3.Tag = "level3"
	n.L1.L2.L3.L4.Tag = "level4"
	n.L1.L2.L3.L4.Value = 99.99
	return n
}

// MakeDeep returns a Deep payload of depth d.
func MakeDeep(d int) *Deep {
	if d <= 0 {
		return nil
	}
	root := &Deep{V: 1}
	cur := root
	for i := 1; i < d; i++ {
		cur.Next = &Deep{V: i + 1}
		cur = cur.Next
	}
	return root
}

// MakeLogBatch returns a synthetic log batch of size n. Most string fields
// have low cardinality (good for Dense intern), TraceID and SpanID are
// unique per entry (no intern win).
func MakeLogBatch(n int) LogBatch {
	rng := rand.New(rand.NewPCG(1, 2))
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	services := []string{"api", "worker", "scheduler", "ingest", "edge"}
	hosts := []string{"host-a", "host-b", "host-c", "host-d"}
	regions := []string{"eu-west-1", "us-east-1", "ap-south-1"}
	msgs := []string{"request received", "request handled", "cache miss", "db timeout", "panic recovered"}
	now := time.Unix(1_700_000_000, 0)
	out := LogBatch{Entries: make([]LogEntry, n)}
	for i := range out.Entries {
		out.Entries[i] = LogEntry{
			Time:     now.Add(time.Duration(i) * time.Millisecond),
			Level:    levels[rng.IntN(len(levels))],
			Service:  services[rng.IntN(len(services))],
			Host:     hosts[rng.IntN(len(hosts))],
			Region:   regions[rng.IntN(len(regions))],
			TraceID:  randomHex(rng, 32),
			SpanID:   randomHex(rng, 16),
			Msg:      msgs[rng.IntN(len(msgs))],
			Duration: rng.Float64() * 1000,
			Status:   200 + rng.IntN(300),
		}
	}
	return out
}

// MakeWide returns Wide containing n Flat items.
func MakeWide(n int) Wide {
	out := Wide{Items: make([]Flat, n)}
	flat := MakeFlat()
	for i := range out.Items {
		f := flat
		f.A = i
		out.Items[i] = f
	}
	return out
}

func randomHex(rng *rand.Rand, n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rng.IntN(16)]
	}
	return string(b)
}
