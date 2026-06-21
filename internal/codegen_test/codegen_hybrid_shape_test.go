package cgsample

import (
	"bytes"
	"testing"
)

// Encode a hybrid batch, then corrupt a column name in the hybrid shape header.
// The hybrid body is decoded POSITIONALLY, so without a shape-validation guard
// the decoder ignores the names and silently accepts the malformed frame. The
// slices.Equal(names/kinds, expected) guard must reject it.
func TestHybridShapeMismatchRejected(t *testing.T) {
	var set GenRowSet
	for i := 0; i < 32; i++ {
		set.Rows = append(set.Rows, GenRow{ID: int64(i), Name: "r", Inner: GenRowInner{}, Tags: map[string]int{"a": i}})
	}
	m := any(&set).(interface{ MarshalQDF([]byte) ([]byte, error) })
	b, err := m.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.Index(b, []byte("inner")) // residual column name in the hybrid shape
	if idx < 0 {
		t.Fatal("'inner' column name not found in hybrid frame")
	}
	corrupt := append([]byte(nil), b...)
	corrupt[idx] = 'x' // "inner" -> "xnner": wire shape no longer matches expected

	var out GenRowSet
	um := any(&out).(interface{ UnmarshalQDF([]byte) (int, error) })
	if _, derr := um.UnmarshalQDF(corrupt); derr == nil {
		t.Fatalf("decode of shape-corrupted hybrid frame returned nil error — malformed frame silently accepted")
	}
}
