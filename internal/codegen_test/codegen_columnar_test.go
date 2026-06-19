package cgsample

import (
	"bytes"
	"testing"
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
