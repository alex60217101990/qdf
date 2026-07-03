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

// TestUnmarshalBatchTruncated: every truncation of a valid columnar payload
// must return an error, never panic (the bare tag index in
// readStringColumnHandles panicked on truncated input before the guard).
func TestUnmarshalBatchTruncated(t *testing.T) {
	src := mkBatSrc(64)
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	for cut := range len(data) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at truncation %d: %v", cut, r)
				}
			}()
			if b, err := UnmarshalBatch[batDoc](data[:cut]); err == nil {
				b.Release()
			}
		}()
	}
}

// TestUnmarshalBatchZoneMapWire: OptZoneMap wraps int/uint/float columns in
// zone-chunk blocks (tagZoneChunk) inside tagColStruct; the batch fast path
// consumes them through the shared *Into readers, which dispatch that tag.
// Guards option-interaction: batch result must equal the reference decode.
func TestUnmarshalBatchZoneMapWire(t *testing.T) {
	src := make([]batSrc, 600) // > zoneChunkMinLen rows -> zone-chunk eligible
	for i := range src {
		src[i] = batSrc{ID: int64(i), Name: "n", Val: float64(i)}
	}
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern|OptZoneMap)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batDoc](data)
	if err != nil {
		t.Fatalf("batch decode of OptZoneMap wire: %v", err)
	}
	defer b.Release()
	var ref []batSrc
	if err := Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	for i := range ref {
		if b.Rows[i].ID != ref[i].ID || b.Rows[i].Val != ref[i].Val {
			t.Fatalf("row %d diverges from reference decode", i)
		}
	}
}

// batNarrow exercises every scalar width the batch scatter dispatches on
// (storeIntWidth/storeUintWidth cases 1/2/4/8, float32, bool) — the committed
// fixtures above only cover int64/float64, so a width-dispatch regression
// would otherwise go undetected.
type batNarrow struct {
	I8  int8    `qdf:"i8"`
	I16 int16   `qdf:"i16"`
	I32 int32   `qdf:"i32"`
	U8  uint8   `qdf:"u8"`
	U16 uint16  `qdf:"u16"`
	U32 uint32  `qdf:"u32"`
	F32 float32 `qdf:"f32"`
	OK  bool    `qdf:"ok"`
	I64 int64   `qdf:"i64"`
}

func TestUnmarshalBatchNarrowWidths(t *testing.T) {
	src := make([]batNarrow, 64) // columnar-eligible
	for i := range src {
		src[i] = batNarrow{
			I8: int8(i - 32), I16: int16(i * 100), I32: int32(i * 100_000),
			U8: uint8(i), U16: uint16(i * 200), U32: uint32(i * 1_000_000),
			F32: float32(i) * 0.5, OK: i%3 == 0, I64: int64(i) << 40,
		}
	}
	data, err := Marshal(src, OptBalanced|OptDense|OptShapeIntern)
	if err != nil {
		t.Fatal(err)
	}
	b, err := UnmarshalBatch[batNarrow](data)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	defer b.Release()
	for i := range src {
		if b.Rows[i] != src[i] {
			t.Fatalf("row %d: %+v != %+v", i, b.Rows[i], src[i])
		}
	}
}
