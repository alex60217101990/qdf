package bench

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	msgpack "github.com/vmihailenco/msgpack/v5"

	qdf "github.com/alex60217101990/qdf"
)

// ----- encode benchmarks -----

func benchEncode[T any](b *testing.B, name string, v T) {
	b.Run(name+"/json", func(b *testing.B) {
		b.ReportAllocs()
		var buf bytes.Buffer
		for b.Loop() {
			buf.Reset()
			enc := json.NewEncoder(&buf)
			if err := enc.Encode(v); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(buf.Len()))
	})
	b.Run(name+"/json_marshal", func(b *testing.B) {
		b.ReportAllocs()
		var last []byte
		for b.Loop() {
			out, err := json.Marshal(v)
			if err != nil {
				b.Fatal(err)
			}
			last = out
		}
		b.SetBytes(int64(len(last)))
	})
	b.Run(name+"/msgpack", func(b *testing.B) {
		b.ReportAllocs()
		var last []byte
		for b.Loop() {
			out, err := msgpack.Marshal(v)
			if err != nil {
				b.Fatal(err)
			}
			last = out
		}
		b.SetBytes(int64(len(last)))
	})
	b.Run(name+"/qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		var last []byte
		for b.Loop() {
			out, err := qdf.Marshal(v, qdf.OptSpeed)
			if err != nil {
				b.Fatal(err)
			}
			last = out
		}
		b.SetBytes(int64(len(last)))
	})
	b.Run(name+"/qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		var last []byte
		for b.Loop() {
			out, err := qdf.Marshal(v, qdf.OptBalanced)
			if err != nil {
				b.Fatal(err)
			}
			last = out
		}
		b.SetBytes(int64(len(last)))
	})
}

// ----- decode benchmarks -----

func benchDecode[T any](b *testing.B, name string, v T) {
	jsonBytes, _ := json.Marshal(v)
	msgpackBytes, _ := msgpack.Marshal(v)
	qdfFastBytes, _ := qdf.Marshal(v, qdf.OptSpeed)
	qdfDenseBytes, _ := qdf.Marshal(v, qdf.OptBalanced)
	b.Run(name+"/json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := json.Unmarshal(jsonBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(jsonBytes)))
	})
	b.Run(name+"/msgpack", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := msgpack.Unmarshal(msgpackBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(msgpackBytes)))
	})
	b.Run(name+"/qdf_fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := qdf.Unmarshal(qdfFastBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(qdfFastBytes)))
	})
	b.Run(name+"/qdf_dense", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := qdf.Unmarshal(qdfDenseBytes, &out); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(qdfDenseBytes)))
	})
	// noCopy cases: WithNoCopy aliases the input buffer (safe here — the bytes
	// live for the whole benchmark and are never reused). It cuts allocations
	// on the reflect path (e.g. Flat/Nested/Wide). Codegen/Unmarshaler types
	// (LogBatch) decode via their own decoder and do NOT inherit the flag, so
	// their nocopy rows stay flat by design.
	b.Run(name+"/qdf_fast_nocopy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := qdf.Unmarshal(qdfFastBytes, &out, qdf.WithNoCopy()); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(qdfFastBytes)))
	})
	b.Run(name+"/qdf_dense_nocopy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out T
			if err := qdf.Unmarshal(qdfDenseBytes, &out, qdf.WithNoCopy()); err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(int64(len(qdfDenseBytes)))
	})
}

// ----- size comparison (also runs roundtrip correctness) -----

func TestSizes(t *testing.T) {
	cases := []struct {
		name string
		v    any
	}{
		{"Tiny", MakeTiny()},
		{"Flat", MakeFlat()},
		{"Nested", MakeNested()},
		{"Deep16", MakeDeep(16)},
		{"Wide_x1000", MakeWide(1000)},
		{"LogBatch_x1000", MakeLogBatch(1000)},
	}
	t.Logf("%-20s %-10s %-10s %-10s %-10s", "payload", "json", "msgpack", "qdf_fast", "qdf_dense")
	for _, c := range cases {
		jb, _ := json.Marshal(c.v)
		mb, _ := msgpack.Marshal(c.v)
		qf, _ := qdf.Marshal(c.v, qdf.OptSpeed)
		qd, _ := qdf.Marshal(c.v, qdf.OptBalanced)
		t.Logf("%-20s %-10d %-10d %-10d %-10d", c.name, len(jb), len(mb), len(qf), len(qd))
		// Round-trip correctness on each.
		out := reflect.New(reflect.TypeOf(c.v)).Interface()
		if err := qdf.Unmarshal(qf, out); err != nil {
			t.Errorf("%s qdf fast decode: %v", c.name, err)
		}
		out2 := reflect.New(reflect.TypeOf(c.v)).Interface()
		if err := qdf.Unmarshal(qd, out2); err != nil {
			t.Errorf("%s qdf dense decode: %v", c.name, err)
		}
	}
}

// ----- top-level benchmarks -----

func BenchmarkEncode_Tiny(b *testing.B)       { benchEncode(b, "Tiny", MakeTiny()) }
func BenchmarkEncode_Flat(b *testing.B)       { benchEncode(b, "Flat", MakeFlat()) }
func BenchmarkEncode_Nested(b *testing.B)     { benchEncode(b, "Nested", MakeNested()) }
func BenchmarkEncode_Deep16(b *testing.B)     { benchEncode(b, "Deep16", MakeDeep(16)) }
func BenchmarkEncode_Wide_x1000(b *testing.B) { benchEncode(b, "Wide1k", MakeWide(1000)) }
func BenchmarkEncode_LogBatch1k(b *testing.B) { benchEncode(b, "LogBatch1k", MakeLogBatch(1000)) }

func BenchmarkDecode_Tiny(b *testing.B)       { benchDecode(b, "Tiny", MakeTiny()) }
func BenchmarkDecode_Flat(b *testing.B)       { benchDecode(b, "Flat", MakeFlat()) }
func BenchmarkDecode_Nested(b *testing.B)     { benchDecode(b, "Nested", MakeNested()) }
func BenchmarkDecode_Deep16(b *testing.B)     { benchDecode(b, "Deep16", MakeDeep(16)) }
func BenchmarkDecode_Wide_x1000(b *testing.B) { benchDecode(b, "Wide1k", MakeWide(1000)) }
func BenchmarkDecode_LogBatch1k(b *testing.B) { benchDecode(b, "LogBatch1k", MakeLogBatch(1000)) }

// BenchmarkEncode_LogBatch1k_Codegen exercises the qdfgen-emitted
// MarshalQDF method on the same 1000-record LogBatch fixture so
// the bench matrix can compare the reflection-free path side by
// side with qdf_fast / qdf_dense. The generated code lives in
// bench_qdf.go (qdfgen -type LogEntry,LogBatch).
//
// Note on the trade-off this bench captures: generated MarshalQDF
// emits Fast-mode wire (no intern table, no MTF, no shape
// interning). For a single struct or a slice of records with
// little repetition, that beats the reflect path by ~25-35 %
// (see internal/codegen_test/bench_test.go for the Sample
// fixture). For a slice with high string repetition (LogBatch1k
// has 1000 rows that share a few hot service/region/level
// values), the reflect+Dense path saves enough wire that the
// reflect overhead is hidden — Dense wins on encode CPU AND
// produces a much smaller wire. Use codegen for hot one-shot
// types, Dense for batched repetitive payloads.
func BenchmarkEncode_LogBatch1k_Codegen(b *testing.B) {
	v := MakeLogBatch(1000)
	b.ReportAllocs()
	b.SetBytes(int64(approxLogBatchBytes(v)))
	for b.Loop() {
		if _, err := v.MarshalQDF(nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode_LogBatch1k_Codegen mirrors the encode side using
// the generated UnmarshalQDF.
func BenchmarkDecode_LogBatch1k_Codegen(b *testing.B) {
	v := MakeLogBatch(1000)
	buf, _ := v.MarshalQDF(nil)
	b.ReportAllocs()
	b.SetBytes(int64(len(buf)))
	for b.Loop() {
		var out LogBatch
		if _, err := out.UnmarshalQDF(buf); err != nil {
			b.Fatal(err)
		}
	}
}

// approxLogBatchBytes returns the encoded byte size, falling back
// to a fresh MarshalQDF call when called outside the benchmark
// setup. Used as a SetBytes hint so the throughput column in
// `go test -bench` output is comparable to the qdf_fast / qdf_dense
// rows.
func approxLogBatchBytes(v LogBatch) int {
	b, _ := v.MarshalQDF(nil)
	return len(b)
}
