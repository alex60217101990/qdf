package cgsample

import (
	"bytes"
	"testing"
)

// Encode a valid columnar batch, then flip the "value" column's kind byte in the
// header (f64=0x02 → int=0x00). The pure-columnar decoder switches on NAME, so
// without a kind check it ignores the corrupt kind byte and silently ACCEPTS the
// malformed frame. The kind check must reject it.
func TestColumnarKindByteFlipRejected(t *testing.T) {
	var batch GenMetricBatch
	batch.Name = "host"
	for i := 0; i < 32; i++ {
		batch.Metrics = append(batch.Metrics, GenMetric{TS: int64(i), Value: float64(i) + 0.5, Count: uint32(i), OK: i%2 == 0, Ratio: float32(i)})
	}
	m := any(&batch).(interface{ MarshalQDF([]byte) ([]byte, error) })
	b, err := m.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Locate the "value" column name in the header, flip the kind byte right after.
	idx := bytes.Index(b, []byte("value"))
	if idx < 0 {
		t.Fatal("value column name not found in frame")
	}
	kindPos := idx + len("value")
	if b[kindPos] != 0x02 { // f64 kind
		t.Fatalf("expected f64 kind 0x02 after 'value', got 0x%02x", b[kindPos])
	}
	corrupt := append([]byte(nil), b...)
	corrupt[kindPos] = 0x00 // claim int

	var out GenMetricBatch
	um := any(&out).(interface{ UnmarshalQDF([]byte) (int, error) })
	if _, derr := um.UnmarshalQDF(corrupt); derr == nil {
		t.Fatalf("decode of kind-flipped frame returned nil error — malformed frame silently accepted")
	}
}
