package qdf

import "testing"

// TestOOM_ConstantSliceOverCapRoundTrips pins that the encoder never produces a
// constant/empty-body codec the decoder would reject: above qpackMaxStandaloneCount
// the emitter falls back to raw (proportional body) so a large constant slice
// still round-trips, while hostile tiny-header inputs are still capped on decode.
// Regression for the encode/decode cap asymmetry the cap-tightening introduced.
func TestOOM_ConstantSliceOverCapRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates a >128 MiB slice; skipped under -short")
	}
	n := qpackMaxStandaloneCount + 1000 // just over the cap → must NOT use a constant codec
	s := make([]int64, n)
	for i := range s {
		s[i] = 42
	}
	buf, err := Marshal(s, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out []int64
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("round-trip broken: encoder produced %d bytes the decoder rejects: %v", len(buf), err)
	}
	if len(out) != n || out[0] != 42 || out[n-1] != 42 {
		t.Fatalf("decoded wrong: len=%d", len(out))
	}
}

// TestQPackConstantOverCap_Logic unit-tests the guard that decides when a chosen
// codec must fall back to raw (no large allocation needed).
func TestQPackConstantOverCap_Logic(t *testing.T) {
	capN := qpackMaxStandaloneCount
	cases := []struct {
		n            int
		codec        qpackCodec
		forB, dB, pB int
		want         bool
	}{
		{capN, qpackFor, 0, 0, 0, false},     // at cap: ok
		{capN + 1, qpackFor, 0, 0, 0, true},  // over cap, constant FOR: redirect
		{capN + 1, qpackFor, 5, 0, 0, false}, // over cap but bits>0: proportional body, keep
		{capN + 1, qpackDeltaFor, 0, 0, 0, true},
		{capN + 1, qpackDeltaFor, 0, 7, 0, false},
		{capN + 1, qpackPFor, 0, 0, 0, true},
		{capN + 1, qpackPFor, 0, 0, 9, false},
		{capN + 1, qpackRLE, 0, 0, 0, true},  // long-run RLE: empty-ish body
		{capN + 1, qpackDict, 0, 0, 0, true}, // single-value dict
		{capN + 1, qpackRaw, 0, 0, 0, false}, // already raw
	}
	for i, c := range cases {
		if got := qpackConstantOverCap(c.n, c.codec, c.forB, c.dB, c.pB); got != c.want {
			t.Errorf("case %d: qpackConstantOverCap(%d,%d,bits %d/%d/%d)=%v want %v",
				i, c.n, c.codec, c.forB, c.dB, c.pB, got, c.want)
		}
	}
}

// TestOOM_ConstantCountCapTightened pins that the standalone constant-codec
// element-count ceiling is tight enough to prevent a multi-GB allocation from a
// ~14-byte header. A constant codec (bitsPer == 0) carries an EMPTY body, so the
// per-element byte bound cannot apply and only the absolute ceiling defends the
// make(). The ceiling must be on the order of maxColumnarElems (1<<24, 128 MiB
// for int64), NOT 1<<30 (8 GiB). `mid` sits above the cap but far below the old
// 8 GiB ceiling, so it asserts the bound was actually tightened. Reader-level
// assertions keep the test safe: a correct reader rejects BEFORE the make.
func TestOOM_ConstantCountCapTightened(t *testing.T) {
	const mid = uint64(1) << 25 // 33.5M elements ⇒ 268 MiB int64 make if accepted

	if qpackMaxStandaloneCount > maxColumnarElems {
		t.Fatalf("qpackMaxStandaloneCount=%d must be <= maxColumnarElems=%d to cap the constant-codec make",
			qpackMaxStandaloneCount, maxColumnarElems)
	}

	t.Run("for", func(t *testing.T) {
		buf := []byte{qpackKindUint64, 0x00, 0x00} // kind, bits=0, min=0
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, _, _, _, _, err := d.readPackedForHeader(qpackKindUint64); err == nil {
			t.Fatal("FOR bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("delta", func(t *testing.T) {
		buf := []byte{qpackKindUint64, 0x00, 0x00, 0x00} // kind, bits=0, first=0, minDelta=0
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, _, _, _, _, _, err := d.readPackedDeltaForHeader(qpackKindUint64); err == nil {
			t.Fatal("Delta-FOR bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("pfor", func(t *testing.T) {
		buf := []byte{qpackKindInt64}
		buf = appendUvarint(buf, mid) // n
		buf = append(buf, 0x00, 0x00) // b=0, min=0
		d := &Decoder{buf: buf}
		if _, err := d.readPackedPForInt64Slice(); err == nil {
			t.Fatal("PFor b=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("dict", func(t *testing.T) {
		// 1 distinct value ⇒ bitsForDistinct(1)==0 ⇒ empty index body.
		buf := []byte{qpackKindInt64}
		buf = appendUvarint(buf, 1) // 1 distinct value
		buf = appendUvarint(buf, 0) // table[0]
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, err := d.readPackedDictInt64Slice(); err == nil {
			t.Fatal("Dict bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("rle", func(t *testing.T) {
		buf := []byte{qpackKindUint64}
		buf = appendUvarint(buf, mid) // claimed element count, body is a tiny run
		d := &Decoder{buf: buf}
		if _, err := d.readPackedRLEHeader(qpackKindUint64); err == nil {
			t.Fatal("RLE accepted a 268 MiB standalone count")
		}
	})
}

// TestOOM_Const32BitSliceNeverLargerThanNative pins that a constant []int32 /
// []uint32 over qpackMaxStandaloneCount does not inflate to the int64/uint64-raw
// 8 B/elem fallback inside emitQPack*: above the cap a constant-body codec is
// rejected, and for a 32-bit slice the native 4 B/elem raw is the real
// never-larger floor. Regression for the constant-over-cap widened-path size
// blow-up (twice 4 B/elem before the fix).
func TestOOM_Const32BitSliceNeverLargerThanNative(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates a >64 MiB slice; skipped under -short")
	}
	n := qpackMaxStandaloneCount + 1000 // over cap → constant codec must redirect

	t.Run("int32", func(t *testing.T) {
		s := make([]int32, n)
		for i := range s {
			s[i] = 7
		}
		buf, err := Marshal(s, OptBalanced)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if len(buf) > 5*n { // native floor ~4n; reject the 8n int64-raw blow-up
			t.Fatalf("never-larger violated: %d bytes for %d int32 (native ~%d)", len(buf), n, 4*n)
		}
		var out []int32
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("round-trip: %v", err)
		}
		if len(out) != n || out[0] != 7 || out[n-1] != 7 {
			t.Fatalf("decoded wrong len=%d", len(out))
		}
	})

	t.Run("uint32", func(t *testing.T) {
		s := make([]uint32, n)
		for i := range s {
			s[i] = 9
		}
		buf, err := Marshal(s, OptBalanced)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if len(buf) > 5*n {
			t.Fatalf("never-larger violated: %d bytes for %d uint32 (native ~%d)", len(buf), n, 4*n)
		}
		var out []uint32
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("round-trip: %v", err)
		}
		if len(out) != n || out[0] != 9 || out[n-1] != 9 {
			t.Fatalf("decoded wrong len=%d", len(out))
		}
	})
}
