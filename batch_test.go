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

func TestUnmarshalBatchColumnar(t *testing.T) {
	src := mkBatSrc(64) // >= columnarMinElems under OptDense -> tagColStruct wire
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batDoc](data)
	if err != nil {
		t.Fatalf("UnmarshalBatch columnar: %v", err)
	}
	defer b.Release()
	if len(b.Rows) != 64 {
		t.Fatalf("rows = %d", len(b.Rows))
	}
	for i, r := range b.Rows {
		if r.ID != int64(i) || b.Str(r.Name) != src[i].Name || r.Val != src[i].Val ||
			!b.TimeOf(r.At).Equal(src[i].At) {
			t.Fatalf("row %d mismatch: id=%d name=%q val=%v at=%v", i, r.ID, b.Str(r.Name), r.Val, b.TimeOf(r.At))
		}
	}
}

func TestBatchColumnarAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("alloc budgets are not measured under -race (sync.Pool churn instrumentation)")
	}
	src := mkBatSrc(512)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	b0, err := UnmarshalBatch[batDoc](data) // warm pools
	if err != nil {
		t.Fatal(err)
	}
	b0.Release()
	allocs := testing.AllocsPerRun(10, func() {
		b, err := UnmarshalBatch[batDoc](data)
		if err != nil {
			t.Fatal(err)
		}
		b.Release()
	})
	// The columnar fast path's allocation floor is a small CONSTANT, independent
	// of the 512 rows: the wrapper's row-slice header + the returned *Batch, the
	// noscan rows backing, and the per-message columnar shape declaration
	// (names+kinds slices) that every independent message re-declares. The
	// string bodies land in the pooled slab (one grow), and every string reader
	// runs under noCopy so distinct values are aliased then copied once into the
	// slab — no per-distinct-string alloc. A regression that reintroduced a
	// per-row string materialization would push this to O(n); the constant
	// budget below catches that. (Measured ~11; a couple of slots of slack.)
	if allocs > 13 {
		t.Fatalf("allocs/op = %v, want <= 13 (columnar fast path did not fire / regressed to per-row alloc?)", allocs)
	}
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
