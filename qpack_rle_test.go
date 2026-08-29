package qdf

import (
	"math"
	"reflect"
	"testing"
)

func TestRLEUint64_RoundTrip(t *testing.T) {
	cases := [][]uint64{
		nil,
		{},
		{42},
		{42, 42, 42, 42, 42, 42, 42, 42, 42, 42},
		{200, 200, 200, 500, 500, 200, 200, 200, 404, 200},
		{0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0},
		{math.MaxUint64, math.MaxUint64, math.MaxUint64, 0, 0},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedRLEUint64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackRLE {
			t.Fatalf("expected tagPackRLE got %02x err=%v in=%v", tag, err, in)
		}
		dec.i++
		out, err := dec.readPackedRLEUint64Slice()
		if err != nil {
			t.Fatalf("decode: %v in=%v", err, in)
		}
		if len(in) == 0 {
			if len(out) != 0 {
				t.Fatalf("empty in, got %v", out)
			}
			continue
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch:\n in=%v\nout=%v", in, out)
		}
	}
}

func TestRLEInt64_RoundTrip(t *testing.T) {
	cases := [][]int64{
		{0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{-1, -1, -1, -1, -1, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
		{200, 200, 200, 500, 500, 200, 200, 200, 404, 200},
		{math.MaxInt64, math.MaxInt64, math.MinInt64, math.MinInt64, 0, 0, 0},
		{-7, -7, -7, 5, 5, 5, 5, -7, -7, 5},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedRLEInt64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackRLE {
			t.Fatalf("expected tagPackRLE got %02x err=%v in=%v", tag, err, in)
		}
		dec.i++
		out, err := dec.readPackedRLEInt64Slice()
		if err != nil {
			t.Fatalf("decode: %v in=%v", err, in)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch:\n in=%v\nout=%v", in, out)
		}
	}
}

// TestRLE_PickerHitsHighRepeat checks that pickU64Codec / pickI64Codec
// actually select qpackRLE when the input is run-heavy. Anchors the
// run-fraction threshold (probeRuns*2 <= probeN, i.e. avg run >= 2).
func TestRLE_PickerHitsHighRepeat(t *testing.T) {
	// 4 runs of length 32 each — avg run length 32, well above the
	// probe threshold and clear wire win over raw / FOR / Delta+FOR.
	u := make([]uint64, 128)
	for i := range 32 {
		u[i] = 200
	}
	for i := 32; i < 64; i++ {
		u[i] = 500
	}
	for i := 64; i < 96; i++ {
		u[i] = 200
	}
	for i := 96; i < 128; i++ {
		u[i] = 404
	}
	if got, _, _, _, _, _, _, _ := pickU64Codec(u); got != qpackRLE {
		t.Fatalf("u64 picker: got codec %d, want qpackRLE (%d)", got, qpackRLE)
	}

	s := make([]int64, 128)
	for i := range 32 {
		s[i] = -1
	}
	for i := 32; i < 64; i++ {
		s[i] = 0
	}
	for i := 64; i < 96; i++ {
		s[i] = 1
	}
	for i := 96; i < 128; i++ {
		s[i] = -2
	}
	if got, _, _, _, _, _, _, _ := pickI64Codec(s); got != qpackRLE {
		t.Fatalf("i64 picker: got codec %d, want qpackRLE (%d)", got, qpackRLE)
	}
}

// TestRLE_PickerSkipsHighEntropy makes sure the run-fraction probe
// rejects random-walk-like inputs. Catches a regression where RLE
// would beat raw on the size estimate but lose dramatically in the
// real-world worst case.
func TestRLE_PickerSkipsHighEntropy(t *testing.T) {
	// Wide-range values with no consecutive repeats — RLE would cost
	// 2 varuints per element, much worse than raw.
	u := make([]uint64, 64)
	for i := range u {
		u[i] = uint64(i*7919 + 1)
	}
	got, _, _, _, _, _, _, _ := pickU64Codec(u)
	if got == qpackRLE {
		t.Fatal("u64 picker: returned qpackRLE on high-entropy input")
	}
}

// TestRLE_ErrorCases exercises the body validator: zero run-length,
// truncated tail, and unknown kind byte all surface clean errors
// instead of panicking on a malformed buffer.
func TestRLE_ErrorCases(t *testing.T) {
	t.Run("zero_runLen", func(t *testing.T) {
		// tag, kind=u64, n=3, value=42, runLen=0 (illegal)
		buf := []byte{tagPackRLE, qpackKindUint64, 0x03, 0x2a, 0x00}
		dec := NewDecoderOnBuf(buf)
		dec.i++ // consume tag
		if _, err := dec.readPackedRLEUint64Slice(); err == nil {
			t.Fatal("expected error on zero runLen, got nil")
		}
	})
	t.Run("truncated_body", func(t *testing.T) {
		// tag, kind=u64, n=5, value=42 (no runLen follows)
		buf := []byte{tagPackRLE, qpackKindUint64, 0x05, 0x2a}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedRLEUint64Slice(); err == nil {
			t.Fatal("expected error on truncated body")
		}
	})
	t.Run("bad_kind", func(t *testing.T) {
		// kind = qpackKindFloat64 (illegal for RLE)
		buf := []byte{tagPackRLE, qpackKindFloat64, 0x01, 0x2a, 0x01}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedRLEUint64Slice(); err == nil {
			t.Fatal("expected ErrTypeMismatch on bad kind")
		}
	})
	t.Run("runLen_overflow_n", func(t *testing.T) {
		// n=3, but first runLen=10 → exceeds declared element count.
		buf := []byte{tagPackRLE, qpackKindUint64, 0x03, 0x2a, 0x0a}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedRLEUint64Slice(); err == nil {
			t.Fatal("expected error on runLen > n")
		}
	})
}

// TestRLEOverflowGuard verifies that a crafted wire input with runLen=MaxUint64
// cannot bypass the bounds guard via integer wrap-around when idx > 0.
//
// Attack: n=5, run1=(val=0, runLen=1) advances idx to 1, run2=(val=1, runLen=MaxUint64).
// Old addition guard: uint64(1)+MaxUint64 wraps to 0, and 0 > 5 is false → bypass.
// Then end = 1 + int(MaxUint64) = 0, inner fill is skipped, idx resets to 0.
// A third run (val=2, runLen=5) then fills all 5 output slots unchallenged → err=nil.
// Fixed subtraction guard: MaxUint64 > uint64(5-1)=4 → true → ErrInvalidLength.
func TestRLEOverflowGuard(t *testing.T) {
	// makeBypassBuf builds: header(n=5) + run1(0,1) + run2(1,MaxUint64) + run3(2,5).
	// With old guard run2 bypasses, idx resets to 0, run3 fills all slots → nil error.
	// With fixed guard run2 is rejected → ErrInvalidLength.
	makeBypassBuf := func(kind byte) []byte {
		var buf []byte
		buf = append(buf, tagPackRLE, kind)
		buf = appendUvarint(buf, 5)              // n=5
		buf = appendUvarint(buf, 0)              // run1 value
		buf = appendUvarint(buf, 1)              // run1 runLen → idx: 0→1
		buf = appendUvarint(buf, 1)              // run2 value
		buf = appendUvarint(buf, math.MaxUint64) // run2 runLen — wraps old addition guard
		buf = appendUvarint(buf, 2)              // run3 value (reached only if bypass succeeds)
		buf = appendUvarint(buf, 5)              // run3 runLen fills all 5 slots → err nil
		return buf
	}

	t.Run("uint64", func(t *testing.T) {
		buf := makeBypassBuf(qpackKindUint64)
		dec := NewDecoderOnBuf(buf)
		dec.i++ // consume tag; readPackedRLEUint64Slice expects kind onward
		_, err := dec.readPackedRLEUint64Slice()
		if err == nil {
			t.Fatal("expected ErrInvalidLength for MaxUint64 runLen overflow bypass, got nil")
		}
	})

	t.Run("int64", func(t *testing.T) {
		buf := makeBypassBuf(qpackKindInt64)
		dec := NewDecoderOnBuf(buf)
		dec.i++ // consume tag
		_, err := dec.readPackedRLEInt64Slice()
		if err == nil {
			t.Fatal("expected ErrInvalidLength for MaxUint64 runLen overflow bypass on int64, got nil")
		}
	})
}

// TestRLE_EndToEnd round-trips an []int64 through the full Marshal /
// Unmarshal API with QPack enabled, confirming that the picker fires
// and the wire decodes back. The fixture is shaped like a real HTTP-
// status column: long runs of 200 punctuated by occasional 4xx/5xx
// bursts; the picker should select RLE and produce wire well below
// the raw fallback.
func TestRLE_EndToEnd(t *testing.T) {
	type row struct {
		Status []int64 `qdf:"status"`
	}
	in := row{Status: make([]int64, 1024)}
	for i := range in.Status {
		switch {
		case i < 400:
			in.Status[i] = 200
		case i < 410:
			in.Status[i] = 500
		case i < 800:
			in.Status[i] = 200
		case i < 805:
			in.Status[i] = 404
		default:
			in.Status[i] = 200
		}
	}

	bRaw, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	bQPack, err := Marshal(in, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	// 5 runs total — body ≈ 5*(2-byte value + 1-2-byte runLen) plus
	// tag/kind/n header. Easily under 50 bytes vs ~2 KB raw.
	if len(bQPack) >= len(bRaw)/16 {
		t.Fatalf("RLE did not shrink as expected: raw=%d qpack=%d (want qpack < raw/16 on 5-run fixture)", len(bRaw), len(bQPack))
	}

	var out row
	if err := Unmarshal(bQPack, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in.Status, out.Status) {
		t.Fatal("round-trip mismatch")
	}
}
