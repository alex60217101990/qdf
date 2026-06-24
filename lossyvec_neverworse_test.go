package qdf

import (
	"bytes"
	"math"
	"testing"
)

// fieldLossySize encodes a single []float64 field through the lossy-eligible
// slice codec and returns the body bytes (after the stream header). It is used
// to compare the lossy vs lossless wire size of the SAME field.
func encodeFloat64Field(t *testing.T, s []float64, opts Options) []byte {
	t.Helper()
	type wrap struct {
		V []float64
	}
	enc := NewEncoderWith(opts)
	if err := enc.EncodeValue(wrap{V: s}); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}
	return enc.Bytes()
}

func encodeFloat32Field(t *testing.T, s []float32, opts Options) []byte {
	t.Helper()
	type wrap struct {
		V []float32
	}
	enc := NewEncoderWith(opts)
	if err := enc.EncodeValue(wrap{V: s}); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}
	return enc.Bytes()
}

// TestLossyVecNeverWorseFloat64 proves the never-worse guarantee: a []float64
// field that is hostile to the lossy codec (all non-finite, so every element
// becomes an exception entry) must encode to <= its lossless size under
// OptLossyVec, and still round-trip exactly.
func TestLossyVecNeverWorseFloat64(t *testing.T) {
	type wrap struct {
		V []float64
	}
	// All +Inf: each element becomes an ~18 B exception → lossy body inflates.
	s := make([]float64, 64)
	for i := range s {
		s[i] = math.Inf(1)
	}

	lossless := encodeFloat64Field(t, s, OptBalanced)
	lossy := encodeFloat64Field(t, s, OptBalanced|OptLossyVec)

	if len(lossy) > len(lossless) {
		t.Fatalf("never-worse violated: lossy %d > lossless %d", len(lossy), len(lossless))
	}
	// In this hostile case the lossless form must win, so no 0xFD on the wire.
	if bytes.IndexByte(lossy, tagColVecLossy) >= 0 {
		t.Fatalf("expected lossless fallback (no 0xFD) for all-+Inf field; got a lossy block")
	}

	var out wrap
	if err := Unmarshal(lossy, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.V) != len(s) {
		t.Fatalf("len: got %d want %d", len(out.V), len(s))
	}
	for i := range s {
		if math.Float64bits(out.V[i]) != math.Float64bits(s[i]) {
			t.Fatalf("row %d: not bit-exact (got %v want %v)", i, out.V[i], s[i])
		}
	}
}

// TestLossyVecNeverWorseFloat32 mirrors the float64 case for []float32.
func TestLossyVecNeverWorseFloat32(t *testing.T) {
	type wrap struct {
		V []float32
	}
	s := make([]float32, 64)
	for i := range s {
		s[i] = float32(math.Inf(1))
	}

	lossless := encodeFloat32Field(t, s, OptBalanced)
	lossy := encodeFloat32Field(t, s, OptBalanced|OptLossyVec)

	if len(lossy) > len(lossless) {
		t.Fatalf("never-worse violated: lossy %d > lossless %d", len(lossy), len(lossless))
	}
	if bytes.IndexByte(lossy, tagColVecLossy) >= 0 {
		t.Fatalf("expected lossless fallback (no 0xFD) for all-+Inf field; got a lossy block")
	}

	var out wrap
	if err := Unmarshal(lossy, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.V) != len(s) {
		t.Fatalf("len: got %d want %d", len(out.V), len(s))
	}
	for i := range s {
		if math.Float32bits(out.V[i]) != math.Float32bits(s[i]) {
			t.Fatalf("row %d: not bit-exact (got %v want %v)", i, out.V[i], s[i])
		}
	}
}

// TestLossyVecScalarFloat64StaysLossless asserts that enabling OptLossyVec does
// NOT make a SCALAR float64 struct field lossy: the columnar transpose gathers
// the column into a []float64 but it must encode losslessly (no 0xFD for that
// column) and round-trip bit-exactly.
func TestLossyVecScalarFloat64StaysLossless(t *testing.T) {
	type scalarRow struct {
		ID    int64
		Tag   string
		Score float64
	}
	const nRows = 64
	rows := make([]scalarRow, nRows)
	for i := range rows {
		rows[i] = scalarRow{
			ID:    int64(i),
			Tag:   "host", // single value so the struct stays columnar-hybrid
			Score: math.Sin(float64(i) * 0.3),
		}
	}

	opts := OptBalanced | OptColumnIndex | OptLossyVec
	enc := NewEncoderWith(opts)
	enc.SetVectorBudget(MinCosine(0.999))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("EncodeValue: %v", err)
	}
	data := enc.Bytes()

	if !bytes.Contains(data, []byte{tagColStruct}) {
		t.Fatal("expected columnar path (tagColStruct); test invalid")
	}
	if bytes.IndexByte(data, tagColVecLossy) >= 0 {
		t.Fatal("scalar float64 column was encoded lossy (0xFD present) under OptLossyVec — must stay lossless")
	}

	var out []scalarRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out) != nRows {
		t.Fatalf("rows: got %d want %d", len(out), nRows)
	}
	for i, orig := range rows {
		if math.Float64bits(out[i].Score) != math.Float64bits(orig.Score) {
			t.Fatalf("row %d: scalar Score not bit-exact (got %v want %v)", i, out[i].Score, orig.Score)
		}
		if out[i].ID != orig.ID || out[i].Tag != orig.Tag {
			t.Fatalf("row %d: scalar fields corrupted", i)
		}
	}
}
