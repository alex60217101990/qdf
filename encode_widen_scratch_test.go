package qdf

import (
	"testing"
	"unsafe"
)

// TestEncodeWidenScratchPooled verifies the int32→int64 / uint32→uint64 QPack
// widening scratch is REUSED across calls rather than reallocated per field. It
// tests the mechanism directly — the backing array must not move when a
// later widen of equal-or-smaller length runs — instead of counting allocations
// (which the race detector's bookkeeping makes unreliable), so it is exact and
// runs identically under -race.
func TestEncodeWidenScratchPooled(t *testing.T) {
	e := &Encoder{qpack: true}

	// First widen sizes the scratch.
	if w := e.widenI64([]int32{10, 20, 30, 40, 50}); len(w) != 5 {
		t.Fatalf("widenI64 len = %d, want 5", len(w))
	}
	backing := unsafe.SliceData(e.wideI64)
	capacity := cap(e.wideI64)

	// Every subsequent widen of length <= capacity must reuse the SAME backing
	// array — that reuse is exactly what turns one make per narrow-int field into
	// zero allocations on a steady-state encode.
	for _, s := range [][]int32{{1, 2}, {3, 4, 5}, {-9, -8, -7, -6, -5}} {
		w := e.widenI64(s)
		if unsafe.SliceData(e.wideI64) != backing || cap(e.wideI64) != capacity {
			t.Fatalf("widenI64 reallocated scratch for len %d (cap %d->%d): not pooled", len(s), capacity, cap(e.wideI64))
		}
		for i, v := range s {
			if w[i] != int64(v) {
				t.Fatalf("widenI64 value [%d] = %d, want %d", i, w[i], v)
			}
		}
	}

	// Same contract for the uint32 path.
	if w := e.widenU64([]uint32{1, 2, 3, 4, 5}); len(w) != 5 {
		t.Fatalf("widenU64 len = %d, want 5", len(w))
	}
	backingU := unsafe.SliceData(e.wideU64)
	capacityU := cap(e.wideU64)
	for _, s := range [][]uint32{{7, 8}, {9, 10, 11}} {
		w := e.widenU64(s)
		if unsafe.SliceData(e.wideU64) != backingU || cap(e.wideU64) != capacityU {
			t.Fatalf("widenU64 reallocated scratch for len %d: not pooled", len(s))
		}
		for i, v := range s {
			if w[i] != uint64(v) {
				t.Fatalf("widenU64 value [%d] = %d, want %d", i, w[i], v)
			}
		}
	}

	// A widen LARGER than the current capacity must grow exactly once and then
	// stay pooled at the new size.
	e.widenI64(make([]int32, capacity*4))
	grown := unsafe.SliceData(e.wideI64)
	if e.widenI64(make([]int32, capacity*4)); unsafe.SliceData(e.wideI64) != grown {
		t.Fatal("widenI64 reallocated on a repeat at the grown size: not pooled after growth")
	}

	// Integration: the real QPack encode path must route []int32 / []uint32
	// through the pooled scratch (not an inline make), so the scratch is
	// populated after an encode. Guards against the call site being un-wired.
	type wired struct {
		A []int32  `qdf:"a"`
		B []uint32 `qdf:"b"`
	}
	enc := NewEncoderWith(OptQPack)
	if err := enc.EncodeValue(wired{A: make([]int32, 300), B: make([]uint32, 300)}); err != nil {
		t.Fatal(err)
	}
	if cap(enc.wideI64) < 300 {
		t.Fatalf("encode did not use the pooled int32 widen scratch (cap=%d): call site not wired", cap(enc.wideI64))
	}
	if cap(enc.wideU64) < 300 {
		t.Fatalf("encode did not use the pooled uint32 widen scratch (cap=%d): call site not wired", cap(enc.wideU64))
	}
}

// TestEncodeWidenScratchRoundTrip guards correctness: the pooled scratch must
// not corrupt the encoded values across repeated encodes of different-length
// slices on the same encoder.
func TestEncodeWidenScratchRoundTrip(t *testing.T) {
	type row struct {
		A []int32  `qdf:"a"`
		B []uint32 `qdf:"b"`
	}
	for _, n := range []int{1, 7, 64, 300, 50, 2} { // varying lengths reuse the scratch
		in := row{A: make([]int32, n), B: make([]uint32, n)}
		for i := range n {
			in.A[i] = int32(-i * 3)
			in.B[i] = uint32(i * 5)
		}
		buf, err := Marshal(in, OptQPack)
		if err != nil {
			t.Fatal(err)
		}
		var out row
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.A) != n || len(out.B) != n {
			t.Fatalf("n=%d len mismatch: %d %d", n, len(out.A), len(out.B))
		}
		for i := 0; i < n; i++ {
			if out.A[i] != in.A[i] || out.B[i] != in.B[i] {
				t.Fatalf("n=%d row %d: A %d!=%d  B %d!=%d", n, i, out.A[i], in.A[i], out.B[i], in.B[i])
			}
		}
	}
}
