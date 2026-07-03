package qdf

import (
	"testing"
	"time"
)

type batDoc struct {
	ID   int64   `qdf:"id"`
	Name Str     `qdf:"name"`
	Val  float64 `qdf:"val"`
	At   Time    `qdf:"at"`
}

// source struct with real strings — encodes the wire the Batch decodes.
type batSrc struct {
	ID   int64     `qdf:"id"`
	Name string    `qdf:"name"`
	Val  float64   `qdf:"val"`
	At   time.Time `qdf:"at"`
}

func mkBatSrc(n int) []batSrc {
	out := make([]batSrc, n)
	for i := range out {
		out[i] = batSrc{
			ID:   int64(i),
			Name: []string{"alpha", "beta", "gamma"}[i%3],
			Val:  float64(i) * 1.5,
			At:   time.Unix(1_700_000_000+int64(i), 500).UTC(),
		}
	}
	return out
}

func TestUnmarshalBatchRowMajor(t *testing.T) {
	src := mkBatSrc(4) // < columnarMinElems -> row-major wire
	data, err := Marshal(src, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batDoc](data)
	if err != nil {
		t.Fatalf("UnmarshalBatch: %v", err)
	}
	defer b.Release()
	if len(b.Rows) != 4 {
		t.Fatalf("rows = %d", len(b.Rows))
	}
	for i, r := range b.Rows {
		if r.ID != int64(i) || b.Str(r.Name) != src[i].Name || r.Val != src[i].Val {
			t.Fatalf("row %d = %+v (name=%q)", i, r, b.Str(r.Name))
		}
		if !b.TimeOf(r.At).Equal(src[i].At) {
			t.Fatalf("row %d time = %v want %v", i, b.TimeOf(r.At), src[i].At)
		}
	}
}
