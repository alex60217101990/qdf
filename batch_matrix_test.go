package qdf

import (
	"fmt"
	"testing"
	"time"
)

// TestBatchStringCodecMatrix drives each string-column codec (raw / const /
// dict / front-coded dict / FSST) through the columnar batch fast path and
// compares every materialized row string against a normal reflect Unmarshal
// of the same wire. Covers both the OptBalanced and OptCompression tiers so
// the rANS-wrapped path (header replaced with a decompressed body) is
// exercised too.
func TestBatchStringCodecMatrix(t *testing.T) {
	cases := []struct {
		name string
		gen  func(i int) string // per-row name generator
		n    int
	}{
		{"raw_highcard", func(i int) string { return fmt.Sprintf("id-%08x-%08x", i*2654435761, i) }, 64},
		{"const", func(i int) string { return "constant-value" }, 64},
		{"dict_lowcard", func(i int) string { return []string{"eu", "us", "ap"}[i%3] }, 256},
		{"frontcoded_sorted", func(i int) string { return fmt.Sprintf("path/%04d/leaf", i) }, 128},
		{"fsst_substr", func(i int) string { return fmt.Sprintf("GET /api/v1/users/%d/profile HTTP/1.1", i) }, 128},
	}
	for _, tier := range []Options{OptBalanced | OptDense | OptShapeIntern, OptCompression | OptDense | OptShapeIntern} {
		for _, c := range cases {
			t.Run(fmt.Sprintf("%s_%x", c.name, tier), func(t *testing.T) {
				src := make([]batSrc, c.n)
				for i := range src {
					src[i] = batSrc{ID: int64(i), Name: c.gen(i), Val: float64(i), At: time.Unix(1700000000+int64(i), 0).UTC()}
				}
				data, err := Marshal(src, tier)
				if err != nil {
					t.Fatal(err)
				}
				b, err := UnmarshalBatch[batDoc](data)
				if err != nil {
					t.Fatalf("batch: %v", err)
				}
				defer b.Release()
				var ref []batSrc
				if err := Unmarshal(data, &ref); err != nil {
					t.Fatal(err)
				}
				if len(b.Rows) != len(ref) {
					t.Fatalf("rows = %d, want %d", len(b.Rows), len(ref))
				}
				for i := range ref {
					if got := b.Str(b.Rows[i].Name); got != ref[i].Name {
						t.Fatalf("row %d: %q != %q", i, got, ref[i].Name)
					}
					if b.Rows[i].ID != ref[i].ID {
						t.Fatalf("row %d id: %d != %d", i, b.Rows[i].ID, ref[i].ID)
					}
					if !b.TimeOf(b.Rows[i].At).Equal(ref[i].At) {
						t.Fatalf("row %d at: %v != %v", i, b.TimeOf(b.Rows[i].At), ref[i].At)
					}
				}
			})
		}
	}
}

// batBytesDoc / batBytesSrc exercise a qdf.Bytes column (which rides the same
// string-column wire) plus empty string bodies through the columnar fast path.
type batBytesDoc struct {
	ID   int64 `qdf:"id"`
	Blob Bytes `qdf:"blob"`
	Name Str   `qdf:"name"`
}

type batBytesSrc struct {
	ID   int64  `qdf:"id"`
	Blob []byte `qdf:"blob"`
	Name string `qdf:"name"`
}

func TestBatchColumnarBytesAndEmpty(t *testing.T) {
	const n = 96
	src := make([]batBytesSrc, n)
	for i := range src {
		var blob []byte
		if i%4 != 0 { // every 4th row: empty blob AND empty name
			blob = []byte(fmt.Sprintf("blob-%d-%x", i, i*7))
		}
		name := ""
		if i%4 != 0 {
			name = fmt.Sprintf("n-%08x", i*2654435761)
		}
		src[i] = batBytesSrc{ID: int64(i), Blob: blob, Name: name}
	}
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batBytesDoc](data)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	defer b.Release()
	if len(b.Rows) != n {
		t.Fatalf("rows = %d, want %d", len(b.Rows), n)
	}
	for i, r := range b.Rows {
		if r.ID != int64(i) {
			t.Fatalf("row %d id %d", i, r.ID)
		}
		if got := b.Str(r.Name); got != src[i].Name {
			t.Fatalf("row %d name %q != %q", i, got, src[i].Name)
		}
		if got := b.BytesOf(r.Blob); string(got) != string(src[i].Blob) {
			t.Fatalf("row %d blob %q != %q", i, got, src[i].Blob)
		}
	}
}
