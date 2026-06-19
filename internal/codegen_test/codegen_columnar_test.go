package cgsample

import (
	"bytes"
	"testing"
	"time"

	"github.com/alex60217101990/qdf"
)

func sampleBatch(n int) GenMetricBatch {
	b := GenMetricBatch{Name: "host1", Metrics: make([]GenMetric, n)}
	for i := range b.Metrics {
		b.Metrics[i] = GenMetric{
			TS: int64(1000 + i), Value: float64(i) * 0.25,
			Count: uint32(i * 3), OK: i%2 == 0, Ratio: float32(i) * 1.5,
		}
	}
	return b
}

func TestColumnarCodegen_EncodesColStructFrame(t *testing.T) {
	b := sampleBatch(64)
	m, ok := any(&b).(interface{ MarshalQDF([]byte) ([]byte, error) })
	if !ok {
		t.Fatal("GenMetricBatch has no generated MarshalQDF")
	}
	buf, err := m.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	// 0xEF == tagColStruct: the Metrics field must be columnar-encoded.
	if !bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("expected a columnar frame (0xEF) in the encoded batch")
	}
}

func TestColumnarCodegen_RoundTrip(t *testing.T) {
	for _, n := range []int{0, 1, 8, 16, 64, 257} { // span gate boundary + nil-ish
		b := sampleBatch(n)
		mm := any(&b).(interface {
			MarshalQDF([]byte) ([]byte, error)
		})
		buf, err := mm.MarshalQDF(nil)
		if err != nil {
			t.Fatalf("n=%d marshal: %v", n, err)
		}
		var got GenMetricBatch
		um := any(&got).(interface {
			UnmarshalQDF([]byte) (int, error)
		})
		if _, err := um.UnmarshalQDF(buf); err != nil {
			t.Fatalf("n=%d unmarshal: %v", n, err)
		}
		if got.Name != b.Name || len(got.Metrics) != len(b.Metrics) {
			t.Fatalf("n=%d shape mismatch: %+v", n, got)
		}
		for i := range b.Metrics {
			if got.Metrics[i] != b.Metrics[i] {
				t.Fatalf("n=%d metric[%d] = %+v, want %+v", n, i, got.Metrics[i], b.Metrics[i])
			}
		}
	}
}

func TestColumnarCodegen_ReflectInterop(t *testing.T) {
	b := sampleBatch(64)
	mm := any(&b).(interface{ MarshalQDF([]byte) ([]byte, error) })
	buf, err := mm.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Reflect decode of generated columnar bytes (GenMetricBatch is Unmarshaler,
	// so reflect delegates to the generated DecodeQDF).
	var got GenMetricBatch
	if err := qdf.Unmarshal(buf, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Metrics) != 64 || got.Metrics[63] != b.Metrics[63] {
		t.Fatalf("reflect interop mismatch: %+v", got.Metrics[63])
	}
}

func marshalQDF(t *testing.T, v interface{ MarshalQDF([]byte) ([]byte, error) }) []byte {
	t.Helper()
	b, err := v.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestColumnarCodegen_EventLog_PureStringTimeNumeric(t *testing.T) {
	in := GenEventLog{Source: "svc"}
	for i := 0; i < 64; i++ {
		in.Events = append(in.Events, GenEvent{
			TS:    time.Unix(int64(1_700_000_000+i), int64(i*1000)).UTC(),
			Level: []string{"INFO", "WARN", "ERROR"}[i%3],
			Code:  int32(i % 13),
			Msg:   "event message",
		})
	}
	buf := marshalQDF(t, &in)
	if !bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("expected pure columnar frame (0xEF)")
	}
	var got GenEventLog
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	if got.Source != in.Source || len(got.Events) != len(in.Events) {
		t.Fatalf("shape mismatch: %+v", got)
	}
	for i := range in.Events {
		if !got.Events[i].TS.Equal(in.Events[i].TS) || got.Events[i] != in.Events[i] {
			t.Fatalf("event[%d]=%+v want %+v", i, got.Events[i], in.Events[i])
		}
	}
}

func TestColumnarCodegen_RowSet_Hybrid(t *testing.T) {
	in := GenRowSet{}
	for i := 0; i < 40; i++ {
		in.Rows = append(in.Rows, GenRow{
			ID:    int64(i),
			Name:  []string{"a", "b"}[i%2],
			Inner: GenRowInner{X: i * 2, Y: "inner"},
			Tags:  map[string]int{"k": i, "z": i + 1},
		})
	}
	buf := marshalQDF(t, &in)
	if !bytes.Contains(buf, []byte{0xF7}) {
		t.Fatal("expected hybrid columnar frame (0xF7)")
	}
	var got GenRowSet
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != len(in.Rows) {
		t.Fatalf("len mismatch: %d", len(got.Rows))
	}
	for i := range in.Rows {
		if got.Rows[i].ID != in.Rows[i].ID || got.Rows[i].Name != in.Rows[i].Name ||
			got.Rows[i].Inner != in.Rows[i].Inner || len(got.Rows[i].Tags) != len(in.Rows[i].Tags) {
			t.Fatalf("row[%d]=%+v want %+v", i, got.Rows[i], in.Rows[i])
		}
		for k, v := range in.Rows[i].Tags {
			if got.Rows[i].Tags[k] != v {
				t.Fatalf("row[%d].Tags[%s]=%d want %d", i, k, got.Rows[i].Tags[k], v)
			}
		}
	}
}

func TestColumnarCodegen_NameList_StringOnlyProbe(t *testing.T) {
	// Low-cardinality → columnar (probe accepts).
	lo := GenNameList{}
	for i := 0; i < 64; i++ {
		lo.Names = append(lo.Names, GenName{First: []string{"Ann", "Bob"}[i%2], Last: "Lee"})
	}
	loBuf := marshalQDF(t, &lo)
	if !bytes.Contains(loBuf, []byte{0xEF}) {
		t.Fatal("low-cardinality string-only should be columnar")
	}
	var loGot GenNameList
	if _, err := (&loGot).UnmarshalQDF(loBuf); err != nil {
		t.Fatal(err)
	}
	for i := range lo.Names {
		if loGot.Names[i] != lo.Names[i] {
			t.Fatalf("lo name[%d]=%v want %v", i, loGot.Names[i], lo.Names[i])
		}
	}
	// High-cardinality → row-major (probe declines), still round-trips.
	hi := GenNameList{}
	for i := 0; i < 64; i++ {
		hi.Names = append(hi.Names, GenName{
			First: "first-" + string(rune('a'+i%26)) + string(rune('0'+i%10)),
			Last:  "last-" + string(rune('A'+i%26)) + string(rune('5'+i%5)),
		})
	}
	hiBuf := marshalQDF(t, &hi)
	if bytes.Contains(hiBuf, []byte{0xEF}) {
		t.Fatal("high-cardinality string-only should stay row-major")
	}
	var hiGot GenNameList
	if _, err := (&hiGot).UnmarshalQDF(hiBuf); err != nil {
		t.Fatal(err)
	}
	for i := range hi.Names {
		if hiGot.Names[i] != hi.Names[i] {
			t.Fatalf("hi name[%d]=%v want %v", i, hiGot.Names[i], hi.Names[i])
		}
	}
}

func benchEventLog() GenEventLog {
	in := GenEventLog{Source: "svc"}
	msgs := []string{"connection established", "request handled", "cache miss", "retry scheduled", "auth ok"}
	for i := 0; i < 2000; i++ {
		in.Events = append(in.Events, GenEvent{
			TS:    time.Unix(int64(1_700_000_000+i), int64(i*1000)).UTC(),
			Level: []string{"INFO", "WARN", "ERROR"}[i%3],
			Code:  int32(i % 13),
			Msg:   msgs[i%len(msgs)], // low-cardinality → dict-coded column
		})
	}
	return in
}

func BenchmarkColumnarCodegen_EventLog_Encode(b *testing.B) {
	in := benchEventLog()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (&in).MarshalQDF(nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkColumnarCodegen_EventLog_Decode(b *testing.B) {
	in := benchEventLog()
	buf, _ := (&in).MarshalQDF(nil)
	b.ReportAllocs()
	for b.Loop() {
		var out GenEventLog
		if _, err := (&out).UnmarshalQDF(buf); err != nil {
			b.Fatal(err)
		}
	}
}
