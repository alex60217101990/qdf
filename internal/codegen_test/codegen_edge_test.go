package cgsample

import (
	"math"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
)

// mustMarshalQDF calls MarshalQDF and fails the test on error.
func mustMarshalQDF(t *testing.T, v interface {
	MarshalQDF([]byte) ([]byte, error)
}) []byte {
	t.Helper()
	b, err := v.MarshalQDF(nil)
	if err != nil {
		t.Fatalf("MarshalQDF: %v", err)
	}
	return b
}

// mustUnmarshalQDF calls UnmarshalQDF and fails the test on error.
func mustUnmarshalQDF(t *testing.T, b []byte, v interface {
	UnmarshalQDF([]byte) (int, error)
}) {
	t.Helper()
	if _, err := v.UnmarshalQDF(b); err != nil {
		t.Fatalf("UnmarshalQDF: %v", err)
	}
}

func TestCodegen_EdgeValues(t *testing.T) {
	// Verify that TestGenerate has already been run (sample_qdf.go exists).
	// The generated methods are always present in the pre-committed file, so
	// we can call them directly without the reflection indirection used in
	// codegen_test.go.

	cases := []struct {
		name string
		in   Sample
	}{
		{
			name: "zero_value",
			in:   Sample{},
		},
		{
			name: "min_max_ints",
			in: Sample{
				Age:    math.MinInt,
				Counts: [3]int32{math.MinInt32, 0, math.MaxInt32},
				Inner:  Inner{X: math.MinInt, Y: 0},
			},
		},
		{
			name: "max_int_age",
			in: Sample{
				Age:    math.MaxInt,
				Counts: [3]int32{math.MaxInt32, math.MaxInt32, math.MaxInt32},
				Inner:  Inner{X: math.MaxInt, Y: math.MaxFloat64},
			},
		},
		{
			name: "float_special_nan",
			in: Sample{
				Score: math.NaN(),
				Inner: Inner{Y: math.NaN()},
			},
		},
		{
			name: "float_special_inf",
			in: Sample{
				Score: math.Inf(1),
				Inner: Inner{Y: math.Inf(-1)},
			},
		},
		{
			name: "float_negative_zero",
			in: Sample{
				Score: math.Float64frombits(1 << 63), // −0
				Inner: Inner{Y: math.Float64frombits(1 << 63)},
			},
		},
		{
			name: "empty_slices_maps",
			in: Sample{
				Tags: []string{},
				Meta: map[string]string{},
				Buf:  []byte{},
			},
		},
		{
			name: "nil_slices_maps",
			in: Sample{
				Tags: nil,
				Meta: nil,
				Buf:  nil,
			},
		},
		{
			name: "nil_opt_ptr",
			in: Sample{
				OptPtr: nil,
			},
		},
		{
			name: "non_nil_opt_ptr",
			in: Sample{
				OptPtr: &Inner{X: 42, Y: -3.14},
			},
		},
		{
			name: "unicode_string",
			in:   Sample{Name: "こんにちは 🌏"},
		},
		{
			name: "invalid_utf8_string",
			// string([]byte{0xff, 0xfe}) per spec
			in: Sample{Name: string([]byte{0xff, 0xfe})},
		},
		{
			name: "time_zero",
			in:   Sample{When: time.Time{}},
		},
		{
			name: "time_unix_epoch",
			in:   Sample{When: time.Unix(0, 0).UTC()},
		},
		{
			name: "time_large",
			in:   Sample{When: time.Unix(1<<32, 999999999).UTC()},
		},
		{
			name: "full_payload",
			in: Sample{
				Name:   "alice",
				Age:    33,
				Active: true,
				Score:  98.6,
				Tags:   []string{"a", "b", "c"},
				Meta:   map[string]string{"k1": "v1", "k2": "v2"},
				Inner:  Inner{X: 7, Y: 1.5},
				When:   time.Unix(1700000000, 0).UTC(),
				Buf:    []byte{1, 2, 3, 4, 5},
				OptPtr: &Inner{X: 99, Y: -2.0},
				Counts: [3]int32{10, 20, 30},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal via generated method.
			b := mustMarshalQDF(t, &tc.in)

			// Unmarshal via generated method.
			var out Sample
			mustUnmarshalQDF(t, b, &out)

			// Verify roundtrip with equalSample (handles float NaN / −0 via
			// the existing helper already in codegen_test.go).
			if !equalSampleEdge(tc.in, out) {
				t.Fatalf("roundtrip mismatch:\n in =%+v\n out=%+v", tc.in, out)
			}
		})
	}
}

// TestCodegen_MatchesOptSpeed asserts that for a representative value the
// generated MarshalQDF output equals qdf.Marshal(v, qdf.OptSpeed) because
// the generator hard-codes the Fast/OptSpeed wire format.
func TestCodegen_MatchesOptSpeed(t *testing.T) {
	in := Sample{
		Name:   "bob",
		Age:    25,
		Active: false,
		Score:  1.5,
		Tags:   []string{"x", "y"},
		Meta:   map[string]string{"foo": "bar"},
		Inner:  Inner{X: 3, Y: 2.0},
		When:   time.Unix(1700000000, 0).UTC(),
		Buf:    []byte{0xDE, 0xAD},
		Counts: [3]int32{1, 2, 3},
	}

	genBytes := mustMarshalQDF(t, &in)

	reflectBytes, err := qdf.Marshal(&in, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("qdf.Marshal(OptSpeed): %v", err)
	}

	if string(genBytes) != string(reflectBytes) {
		t.Fatalf("generated output != qdf.Marshal(OptSpeed):\n gen    =%x\n reflect=%x", genBytes, reflectBytes)
	}
}

// equalSampleEdge is like equalSample from codegen_test.go but handles
// float NaN (by bits) and −0.
func equalSampleEdge(a, b Sample) bool {
	if math.Float64bits(a.Score) != math.Float64bits(b.Score) {
		return false
	}
	if math.Float64bits(a.Inner.Y) != math.Float64bits(b.Inner.Y) {
		return false
	}
	if a.Inner.X != b.Inner.X {
		return false
	}
	// Delegate the rest of the fields to the existing equalSample helper.
	// Temporarily zero out the float fields we already compared.
	aCopy := a
	bCopy := b
	aCopy.Score = 0
	bCopy.Score = 0
	aCopy.Inner.Y = 0
	bCopy.Inner.Y = 0
	aCopy.Inner.X = 0
	bCopy.Inner.X = 0
	return equalSample(aCopy, bCopy)
}
