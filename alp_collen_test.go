package qdf

import (
	"bytes"
	"errors"
	"testing"
)

type alpColRow struct {
	V          float64
	A, B, C, D int64
	S          string
}

// A columnar float column body that is a tagPackALP block whose element count
// exceeds the columnar row count must be rejected by colLenOK BEFORE the
// make([]float64, n) allocation — not decoded into an oversized slice that only
// the post-decode length check would catch. The columnar float encoder writes
// the column as a plain element array, but the columnar decoder ALSO accepts a
// tagPackALP body (an encode/decode asymmetry: a hostile or future-encoder wire
// can carry one), and that reader sets d.colMaxLen to the row count. The reader
// must bound the ALP count by colMaxLen exactly as the Gorilla reader does. This
// splices a hostile constant-ALP block over the float column of a real columnar
// wire and asserts rejection rather than a 128 MB over-allocation.
func TestALPColumnarHostileCountRejected(t *testing.T) {
	const n = 300
	rows := make([]alpColRow, n)
	for i := range rows {
		rows[i] = alpColRow{V: float64(i) * 1.5, A: int64(i), B: int64(i * 2), C: int64(i % 7), D: int64(1000 + i), S: "row"}
	}
	b, err := MarshalT(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if b[5] != tagColStruct {
		t.Fatalf("payload is not columnar (b[5]=0x%02x)", b[5])
	}
	if b[4]&FlagRANS != 0 {
		t.Fatal("wire is rANS-framed; column bodies are not directly addressable")
	}

	// The float64 column V is written as a raw-LE block: tagPackRaw + kind(float64)
	// + count + n×8 bytes. That tag pair uniquely marks the float column body
	// start — the byte the columnar decoder peeks to choose its float reader.
	marker := []byte{tagPackRaw, qpackKindFloat64}
	off := bytes.Index(b, marker)
	if off < 0 {
		t.Fatal("raw float64 column not located in columnar wire")
	}

	// Hostile constant (width==0) ALP block claiming far more elements than the
	// row count; decode reaches it via decodeSliceFloat64Into -> tagPackALP with
	// d.colMaxLen == n.
	var alp []byte
	alp = append(alp, tagPackALP, qpackKindFloat64)
	alp = appendUvarint(alp, uint64(n*1000)) // count >> row count
	alp = append(alp, 0x00)                  // d (exponent)
	alp = appendUvarint(alp, zigzagEncode64(0))
	alp = append(alp, 0x00) // width == 0 (constant path)
	alp = appendUvarint(alp, 0)
	if len(alp) > len(b)-off {
		t.Fatal("wire too short to splice")
	}

	patched := make([]byte, len(b))
	copy(patched, b)
	copy(patched[off:], alp) // overwrite the array head; decode errors before the tail matters

	var out []alpColRow
	err = Unmarshal(patched, &out)
	if err == nil {
		t.Fatal("hostile columnar ALP count (>> row count) accepted; colLenOK gate missing")
	}
	// colLenOK rejects with ErrInvalidLength BEFORE the make([]float64, count).
	// Without the gate the count (well under alpMaxElems) would allocate first and
	// only the post-decode length-mismatch check would reject — a different error.
	if !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("rejected with %v, want ErrInvalidLength (colLenOK gate); a different error means the oversized make() ran first", err)
	}
}
