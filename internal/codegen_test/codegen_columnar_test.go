package cgsample

import (
	"bytes"
	"testing"

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
