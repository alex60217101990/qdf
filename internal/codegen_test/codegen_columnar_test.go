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

func TestColumnarCodegen_BlobColumn_RoundTrip(t *testing.T) {
	in := GenBlobSet{}
	for i := 0; i < 40; i++ {
		in.Rows = append(in.Rows, GenBlobRow{ID: int64(i), Data: []byte{byte(i), byte(i + 1), byte(i * 2)}})
	}
	buf, err := (&in).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("expected columnar frame for []byte column")
	}
	var got GenBlobSet
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != len(in.Rows) {
		t.Fatalf("len mismatch %d", len(got.Rows))
	}
	for i := range in.Rows {
		if got.Rows[i].ID != in.Rows[i].ID || !bytes.Equal(got.Rows[i].Data, in.Rows[i].Data) {
			t.Fatalf("row[%d]=%+v want %+v", i, got.Rows[i], in.Rows[i])
		}
	}
}

func TestColumnarCodegen_NullableColumns_RoundTrip(t *testing.T) {
	pi := func(v int32) *int32 { return &v }
	ps := func(v string) *string { return &v }
	pb := func(v bool) *bool { return &v }
	pf := func(v float64) *float64 { return &v }
	in := GenOptSet{}
	for i := 0; i < 40; i++ {
		var r GenOpt
		if i%2 == 0 {
			r.A = pi(int32(i))
		}
		if i%3 == 0 {
			r.B = ps([]string{"x", "y"}[i%2])
		}
		if i%5 == 0 {
			r.C = pb(i%10 == 0)
		}
		if i%4 == 0 {
			r.D = pf(float64(i) * 1.25)
		}
		in.Rows = append(in.Rows, r)
	}
	buf, err := (&in).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf, []byte{0xEF}) {
		t.Fatal("expected columnar frame for nullable columns")
	}
	var got GenOptSet
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != len(in.Rows) {
		t.Fatalf("len mismatch %d", len(got.Rows))
	}
	eqI := func(a, b *int32) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	eqS := func(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	eqB := func(a, b *bool) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	eqF := func(a, b *float64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	for i := range in.Rows {
		g, w := got.Rows[i], in.Rows[i]
		if !eqI(g.A, w.A) || !eqS(g.B, w.B) || !eqB(g.C, w.C) || !eqF(g.D, w.D) {
			t.Fatalf("row[%d] mismatch: got %+v want %+v", i, g, w)
		}
	}
}

// Regression: an all-nil *string nullable column has present=0, so its dense
// string part is empty. The empty column must emit nothing (not a stray raw
// block) or the decode cursor desyncs. (agent bug-hunt 2026-06-19)
func TestColumnarCodegen_AllNilNullableString(t *testing.T) {
	pi := func(v int32) *int32 { return &v }
	in := GenOptSet{}
	for i := 0; i < 20; i++ {
		in.Rows = append(in.Rows, GenOpt{A: pi(int32(i))}) // B (*string), C, D all nil
	}
	buf, err := (&in).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got GenOptSet
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatalf("all-nil nullable string desync: %v", err)
	}
	if len(got.Rows) != len(in.Rows) {
		t.Fatalf("len mismatch %d", len(got.Rows))
	}
	for i := range in.Rows {
		if got.Rows[i].B != nil {
			t.Fatalf("row[%d].B should be nil, got %q", i, *got.Rows[i].B)
		}
		if got.Rows[i].A == nil || *got.Rows[i].A != int32(i) {
			t.Fatalf("row[%d].A wrong: %v", i, got.Rows[i].A)
		}
	}
}

// Regression (agent round 2): a nil []byte must decode back to nil (not a
// non-nil empty slice), and []byte{} must stay non-nil — preserved by routing
// []byte through the nullable column (presence bit = field != nil).
func TestColumnarCodegen_NilEmptyByteColumn(t *testing.T) {
	in := GenBlobSet{}
	for i := 0; i < 32; i++ {
		var d []byte
		switch i % 3 {
		case 0:
			d = []byte{byte(i), byte(i + 1)}
		case 1:
			d = []byte{} // empty, non-nil
		default:
			d = nil // nil
		}
		in.Rows = append(in.Rows, GenBlobRow{ID: int64(i), Data: d})
	}
	buf, err := (&in).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got GenBlobSet
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	for i := range in.Rows {
		gn, wn := got.Rows[i].Data == nil, in.Rows[i].Data == nil
		if gn != wn {
			t.Fatalf("row[%d] nil-ness: got nil=%v want nil=%v", i, gn, wn)
		}
		if !bytes.Equal(got.Rows[i].Data, in.Rows[i].Data) {
			t.Fatalf("row[%d] data: got %v want %v", i, got.Rows[i].Data, in.Rows[i].Data)
		}
	}
}

// Regression (agent round 2 coverage gap): fields AFTER the columnar []struct
// field must decode correctly on the shared decoder (colMaxLen reset).
func TestColumnarCodegen_FieldsAfterColumnar(t *testing.T) {
	in := GenTrailed{Note: "trailing note", Tail: []int64{7, 8, 9, 10, 11}}
	for i := 0; i < 40; i++ {
		in.Rows = append(in.Rows, GenMetric{TS: int64(i), Value: float64(i), Count: uint32(i), OK: i%2 == 0, Ratio: float32(i)})
	}
	buf, err := (&in).MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got GenTrailed
	if _, err := (&got).UnmarshalQDF(buf); err != nil {
		t.Fatal(err)
	}
	if got.Note != in.Note || len(got.Rows) != len(in.Rows) || len(got.Tail) != len(in.Tail) {
		t.Fatalf("shape: note=%q rows=%d tail=%v", got.Note, len(got.Rows), got.Tail)
	}
	for i := range in.Tail {
		if got.Tail[i] != in.Tail[i] {
			t.Fatalf("tail[%d]=%d want %d", i, got.Tail[i], in.Tail[i])
		}
	}
}
