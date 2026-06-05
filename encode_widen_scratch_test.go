package qdf

import "testing"

// TestEncodeWidenScratchPooled is the RED test for pooling the int32→int64 /
// uint32→uint64 widening scratch in the QPack slice encoders. Each []int32 /
// []uint32 field currently allocates a fresh make([]int64,len) / make([]uint64,
// len) before the codec picker runs; on a reused encoder that scratch must be
// pooled so a steady-state encode of many narrow-int slices is allocation-free.
func TestEncodeWidenScratchPooled(t *testing.T) {
	// One- vs five-narrow-int-slice payloads. With the widening scratch pooled,
	// both cost the SAME number of allocations per encode (only the fixed
	// AppendMarshal overhead) — every per-field make([]int64/[]uint64,len) is
	// gone. Comparing the two counts is robust to the constant overhead and to
	// the race detector's extra bookkeeping allocs (an absolute bound is not).
	type row1 struct {
		A []int32 `qdf:"a"`
	}
	type row5 struct {
		A []int32  `qdf:"a"`
		B []int32  `qdf:"b"`
		C []uint32 `qdf:"c"`
		D []int32  `qdf:"d"`
		E []uint32 `qdf:"e"`
	}
	seqI := func(n int) []int32 {
		s := make([]int32, n)
		for i := range s {
			s[i] = int32(i)
		}
		return s
	}
	seqU := func(n int) []uint32 {
		s := make([]uint32, n)
		for i := range s {
			s[i] = uint32(i)
		}
		return s
	}
	v1 := row1{A: seqI(500)}
	v5 := row5{A: seqI(500), B: seqI(500), C: seqU(500), D: seqI(500), E: seqU(500)}

	measure := func(v any) float64 {
		dst, err := AppendMarshal(nil, v, OptQPack) // warm scratch + grow dst
		if err != nil {
			t.Fatal(err)
		}
		return testing.AllocsPerRun(50, func() {
			var e error
			if dst, e = AppendMarshal(dst[:0], v, OptQPack); e != nil {
				t.Fatal(e)
			}
		})
	}
	a1, a5 := measure(v1), measure(v5)
	// Four extra narrow-int slices in row5 must add ZERO allocations once the
	// widening scratch is pooled (pre-fix this gap was 4).
	if a5 > a1 {
		t.Fatalf("5-slice encode = %.0f allocs/op vs 1-slice = %.0f: per-field widening not pooled (gap %.0f)", a5, a1, a5-a1)
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
