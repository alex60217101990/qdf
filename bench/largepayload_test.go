package bench

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// Large-payload comparison. The builder produces a struct that
// serialises to roughly 200 MB across every encoder. It covers every
// field type qdf supports (scalars, strings with hot/cold split,
// nested struct, []float64 / []int32 numeric slices, []byte, map,
// per-event tags).
//
// Nothing is written to disk — GitHub's 100 MB per-file cap makes a
// committed fixture impractical. The generator is deterministic
// (fixed math/rand seed) so runs are reproducible across machines.
//
// All tests in this file gate on `testing.Short()`. Run them
// explicitly:
//
//	go test -C bench -run TestSizes_LargePayload -count=1 -timeout=10m
//	go test -C bench -bench BenchmarkLargePayload -benchtime=1x -count=3 -timeout=30m

type largeRecord struct {
	ID        uint64            `json:"id"        msgpack:"id"        qdf:"id"`
	UUID      string            `json:"uuid"      msgpack:"uuid"      qdf:"uuid"`
	Service   string            `json:"service"   msgpack:"service"   qdf:"service"`
	Region    string            `json:"region"    msgpack:"region"    qdf:"region"`
	Level     string            `json:"level"     msgpack:"level"     qdf:"level"`
	Host      string            `json:"host"      msgpack:"host"      qdf:"host"`
	Msg       string            `json:"msg"       msgpack:"msg"       qdf:"msg"`
	Active    bool              `json:"active"    msgpack:"active"    qdf:"active"`
	Score     float64           `json:"score"     msgpack:"score"     qdf:"score"`
	Latency   float32           `json:"latency"   msgpack:"latency"   qdf:"latency"`
	Timestamp int64             `json:"ts"        msgpack:"ts"        qdf:"ts"`
	Span      uint64            `json:"span"      msgpack:"span"      qdf:"span"`
	Tags      []string          `json:"tags"      msgpack:"tags"      qdf:"tags"`
	Attrs     map[string]string `json:"attrs"     msgpack:"attrs"     qdf:"attrs"`
	Path      []int32           `json:"path"      msgpack:"path"      qdf:"path"`
	Bytes     []byte            `json:"bytes"     msgpack:"bytes"     qdf:"bytes"`
	Vec       []float64         `json:"vec"       msgpack:"vec"       qdf:"vec"`
}

type largeBatch struct {
	Source    string        `json:"source"    msgpack:"source"    qdf:"source"`
	Generated time.Time     `json:"generated" msgpack:"generated" qdf:"generated"`
	Records   []largeRecord `json:"records"   msgpack:"records"   qdf:"records"`
}

// hotPool returns N strings drawn from a small alphabet repeated to
// fill the slice — simulates real telemetry where a tiny number of
// values dominate.
func hotPool(rng *rand.Rand, hot []string) string {
	return hot[rng.Intn(len(hot))]
}

func makeLargeBatch(records int, seed int64) largeBatch {
	rng := rand.New(rand.NewSource(seed))
	services := []string{"ingest", "auth", "billing", "api-gateway", "search", "ranker", "router", "indexer"}
	regions := []string{"eu-west-1", "us-east-1", "ap-south-1", "eu-central-1", "us-west-2"}
	levels := []string{"debug", "info", "warn", "error"}
	hosts := []string{"node-001", "node-002", "node-003", "node-004", "node-005", "node-006"}

	out := largeBatch{
		Source:    "qdf-large-corpus",
		Generated: time.Unix(1_700_000_000, 0).UTC(),
		Records:   make([]largeRecord, records),
	}
	for i := range out.Records {
		nTags := rng.Intn(4) + 1
		tags := make([]string, nTags)
		for j := range tags {
			tags[j] = hotPool(rng, services)
		}
		nAttrs := rng.Intn(3) + 2
		attrs := make(map[string]string, nAttrs)
		for j := range nAttrs {
			k := fmt.Sprintf("k%d", j)
			attrs[k] = fmt.Sprintf("v%d-%d", j, rng.Intn(64))
		}
		nPath := rng.Intn(8) + 2
		path := make([]int32, nPath)
		for j := range path {
			path[j] = rng.Int31n(1_000_000)
		}
		bufLen := rng.Intn(64) + 16
		buf := make([]byte, bufLen)
		rng.Read(buf)
		nVec := rng.Intn(16) + 4
		vec := make([]float64, nVec)
		for j := range vec {
			vec[j] = rng.NormFloat64()
		}
		out.Records[i] = largeRecord{
			ID:        rng.Uint64(),
			UUID:      fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", rng.Uint32(), rng.Uint32()&0xFFFF, rng.Uint32()&0xFFFF, rng.Uint32()&0xFFFF, rng.Uint64()&0xFFFFFFFFFFFF),
			Service:   hotPool(rng, services),
			Region:    hotPool(rng, regions),
			Level:     hotPool(rng, levels),
			Host:      hotPool(rng, hosts),
			Msg:       "request completed successfully in given window",
			Active:    rng.Intn(2) == 0,
			Score:     rng.Float64() * 100,
			Latency:   rng.Float32() * 50,
			Timestamp: 1_700_000_000 + int64(i),
			Span:      rng.Uint64(),
			Tags:      tags,
			Attrs:     attrs,
			Path:      path,
			Bytes:     buf,
			Vec:       vec,
		}
	}
	return out
}

// TestSizes_LargePayload reports encoded sizes only. Skipped under
// -short. Use:
//
//	go test -C bench -run TestSizes_LargePayload -count=1 -timeout=10m
func TestSizes_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-payload sizes under -short")
	}
	const records = 200_000
	v := makeLargeBatch(records, 1)
	t.Logf("built %d records", records)

	jb, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("  json:       %d bytes (%.2f MiB)", len(jb), float64(len(jb))/(1<<20))
	mb, err := msgpack.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("  msgpack:    %d bytes (%.2f MiB)", len(mb), float64(len(mb))/(1<<20))
	fb, err := qdf.Marshal(v, qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("  qdf_fast:   %d bytes (%.2f MiB)", len(fb), float64(len(fb))/(1<<20))
	qb, err := qdf.Marshal(v, qdf.OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("  qdf_qpack:  %d bytes (%.2f MiB)", len(qb), float64(len(qb))/(1<<20))
	db, err := qdf.Marshal(v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("  qdf_dense:  %d bytes (%.2f MiB)", len(db), float64(len(db))/(1<<20))
	t.Logf("ratios vs json: fast=%.2fx qpack=%.2fx dense=%.2fx",
		float64(len(fb))/float64(len(jb)),
		float64(len(qb))/float64(len(jb)),
		float64(len(db))/float64(len(jb)))
}

// memSnapshot grabs the heap-in-use figure (delta-able). Use before
// and after a single Marshal/Unmarshal to estimate peak working-set.
func memSnapshot() uint64 {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// TestMem_LargePayload reports working-set memory + allocation
// counts for one encode + one decode round. Skipped under -short.
func TestMem_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-payload mem under -short")
	}
	const records = 100_000 // ~100 MiB-class
	v := makeLargeBatch(records, 2)

	type result struct {
		name   string
		bytes  int
		encDur time.Duration
		decDur time.Duration
		encMem int64
		decMem int64
	}
	var results []result

	run := func(name string, encode func(any) ([]byte, error), decode func([]byte) any) {
		// Encode.
		before := memSnapshot()
		t0 := time.Now()
		buf, err := encode(v)
		encDur := time.Since(t0)
		after := memSnapshot()
		if err != nil {
			t.Fatalf("%s encode: %v", name, err)
		}
		encMem := int64(after) - int64(before)
		// Decode.
		before = memSnapshot()
		t0 = time.Now()
		_ = decode(buf)
		decDur := time.Since(t0)
		after = memSnapshot()
		decMem := int64(after) - int64(before)

		results = append(results, result{
			name:   name,
			bytes:  len(buf),
			encDur: encDur,
			decDur: decDur,
			encMem: encMem,
			decMem: decMem,
		})
	}

	run("json", json.Marshal, func(b []byte) any { var out largeBatch; _ = json.Unmarshal(b, &out); return out })
	run("msgpack", msgpack.Marshal, func(b []byte) any { var out largeBatch; _ = msgpack.Unmarshal(b, &out); return out })
	run("qdf_fast", func(v any) ([]byte, error) { return qdf.Marshal(v, qdf.OptSpeed) }, func(b []byte) any { var out largeBatch; _ = qdf.Unmarshal(b, &out); return out })
	run("qdf_qpack", func(v any) ([]byte, error) { return qdf.Marshal(v, qdf.OptQPack) }, func(b []byte) any { var out largeBatch; _ = qdf.Unmarshal(b, &out); return out })
	run("qdf_dense", func(v any) ([]byte, error) { return qdf.Marshal(v, qdf.OptBalanced) }, func(b []byte) any { var out largeBatch; _ = qdf.Unmarshal(b, &out); return out })

	t.Logf("Large payload, %d records:", records)
	t.Logf("%-12s %10s  %10s  %10s  %10s  %10s",
		"format", "bytes/MiB", "enc ms", "dec ms", "enc MiB", "dec MiB")
	for _, r := range results {
		t.Logf("%-12s %10.2f  %10.0f  %10.0f  %10.2f  %10.2f",
			r.name,
			float64(r.bytes)/(1<<20),
			float64(r.encDur)/float64(time.Millisecond),
			float64(r.decDur)/float64(time.Millisecond),
			float64(r.encMem)/(1<<20),
			float64(r.decMem)/(1<<20))
	}
}

// BenchmarkLargePayload_Encode runs each encoder once per iter (b.N=1
// is the realistic case at this size). Use:
//
//	go test -C bench -bench BenchmarkLargePayload_Encode -benchtime=1x -count=3 -timeout=30m
func BenchmarkLargePayload_Encode(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large-payload bench under -short")
	}
	const records = 50_000
	v := makeLargeBatch(records, 3)
	// Pre-encode once per codec so each subtest reports MB/s based
	// on the actual wire size produced for the same fixture. Sizes
	// vary by codec (json ≈ 90 MiB, qdf_dense ≈ 25 MiB) so a single
	// shared SetBytes would skew the comparison.
	jb, _ := json.Marshal(v)
	mb, _ := msgpack.Marshal(v)
	fb, _ := qdf.Marshal(v, qdf.OptSpeed)
	qb, _ := qdf.Marshal(v, qdf.OptQPack)
	db, _ := qdf.Marshal(v, qdf.OptBalanced)

	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jb)))
		for b.Loop() {
			_, _ = json.Marshal(v)
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(mb)))
		for b.Loop() {
			_, _ = msgpack.Marshal(v)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(fb)))
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptSpeed)
		}
	})
	b.Run("qdf_qpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(qb)))
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptQPack)
		}
	})
	b.Run("qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(db)))
		for b.Loop() {
			_, _ = qdf.Marshal(v, qdf.OptBalanced)
		}
	})
}

// BenchmarkLargePayload_Decode mirrors the encode variant.
func BenchmarkLargePayload_Decode(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large-payload bench under -short")
	}
	const records = 50_000
	v := makeLargeBatch(records, 4)
	jb, _ := json.Marshal(v)
	mb, _ := msgpack.Marshal(v)
	fb, _ := qdf.Marshal(v, qdf.OptSpeed)
	qb, _ := qdf.Marshal(v, qdf.OptQPack)
	db, _ := qdf.Marshal(v, qdf.OptBalanced)

	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(jb)))
		for b.Loop() {
			var out largeBatch
			_ = json.Unmarshal(jb, &out)
		}
	})
	b.Run("msgpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(mb)))
		for b.Loop() {
			var out largeBatch
			_ = msgpack.Unmarshal(mb, &out)
		}
	})
	b.Run("qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(fb)))
		for b.Loop() {
			var out largeBatch
			_ = qdf.Unmarshal(fb, &out)
		}
	})
	b.Run("qdf_qpack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(qb)))
		for b.Loop() {
			var out largeBatch
			_ = qdf.Unmarshal(qb, &out)
		}
	})
	b.Run("qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(db)))
		for b.Loop() {
			var out largeBatch
			_ = qdf.Unmarshal(db, &out)
		}
	})
}

// A payload the encoder sends row-major must stay row-major. Per-column
// demotion refines a columnar decision; it must never reverse a row-major one,
// because emitting a container builds per-column scratch sized to the row count
// — on this payload that was 18 MB/op becoming 66 MB/op while the wire did not
// move at all, and it reached CI as a 3.65x memory alert.
//
// Asserted on the container, not on allocation: the container is what changed,
// it is deterministic, and an allocation ceiling measured through
// testing.Benchmark inside a test does not reproduce the benchmark's own
// amortisation. The byte budget itself is watched by bench-trend, which is what
// caught this in the first place.
func TestLargePayloadEncodeStaysRowMajor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the 200k-record fixture under -short")
	}
	v := makeLargeBatch(200_000, 1)
	b, err := qdf.Marshal(v, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Byte 5 is the first tag after the 5-byte header. 0xF7 is
	// tagHybridColStruct; anything else means the root did not become a
	// columnar container.
	if b[5] == 0xF7 {
		t.Fatalf("the root encoded as a hybrid columnar container — per-column demotion reversed a row-major decision")
	}
	t.Logf("root tag=0x%02x wire=%d", b[5], len(b))
}
