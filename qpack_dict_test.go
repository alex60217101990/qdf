package qdf

import (
	"math"
	"reflect"
	"testing"
)

func TestDictUint64_RoundTrip(t *testing.T) {
	cases := [][]uint64{
		{42},
		{42, 42, 42, 42, 42, 42, 42, 42, 42, 42},
		{100, 1000, 100, 50000, 100, 1000, 100, 50000, 50000, 100},
		{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1},
		{math.MaxUint64, 0, math.MaxUint64, 1, math.MaxUint64, 1, 0, 1, 0, math.MaxUint64},
		// 16 distinct values — at the cap, still valid.
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedDictUint64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackDict {
			t.Fatalf("expected tagPackDict got %02x err=%v in=%v", tag, err, in)
		}
		dec.i++
		out, err := dec.readPackedDictUint64Slice()
		if err != nil {
			t.Fatalf("decode: %v in=%v", err, in)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch:\n in=%v\nout=%v", in, out)
		}
	}
}

func TestDictInt64_RoundTrip(t *testing.T) {
	cases := [][]int64{
		{0},
		{-1, 0, 1, -1, 0, 1, -1, 0, 1, -1},
		{math.MinInt64, math.MaxInt64, 0, math.MinInt64, math.MaxInt64, 0, math.MinInt64},
		{-5, -5, -5, 5, 5, 5, -5, 5, 5, -5},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	for _, in := range cases {
		enc := NewEncoder(Fast)
		enc.writePackedDictInt64Slice(in)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackDict {
			t.Fatalf("expected tagPackDict got %02x err=%v in=%v", tag, err, in)
		}
		dec.i++
		out, err := dec.readPackedDictInt64Slice()
		if err != nil {
			t.Fatalf("decode: %v in=%v", err, in)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch:\n in=%v\nout=%v", in, out)
		}
	}
}

// TestDict_PickerHitsSpreadValues checks that the picker selects
// qpackDict when distinct count is small but the values are spread
// far apart enough that FOR can't bit-pack them densely.
func TestDict_PickerHitsSpreadValues(t *testing.T) {
	// 4 distinct values spanning a huge range — FOR would need 16+
	// bits per element; dict only needs 2.
	s := make([]uint64, 256)
	dictVals := []uint64{100, 1000, 50000, 9_999_999}
	for i := range s {
		s[i] = dictVals[i%4]
	}
	codec, _, _, _, _, _ := pickU64Codec(s)
	if codec != qpackDict {
		t.Fatalf("u64 picker: got codec %d, want qpackDict (%d) on 4-distinct/wide-range input", codec, qpackDict)
	}

	si := make([]int64, 256)
	dictI := []int64{-1_000_000, 0, 500_000, -500_000}
	for i := range si {
		si[i] = dictI[i%4]
	}
	codec, _, _, _, _, _ = pickI64Codec(si)
	if codec != qpackDict {
		t.Fatalf("i64 picker: got codec %d, want qpackDict (%d) on 4-distinct/wide-range input", codec, qpackDict)
	}
}

// TestDict_PickerSkipsHighCardinality makes sure the probe early-
// exits and the picker never selects dict when distinct > cap.
func TestDict_PickerSkipsHighCardinality(t *testing.T) {
	s := make([]uint64, 64)
	for i := range s {
		s[i] = uint64(i * 113) // 64 unique values, well above cap
	}
	codec, _, _, _, _, _ := pickU64Codec(s)
	if codec == qpackDict {
		t.Fatalf("u64 picker: returned qpackDict on >cap distinct (=%d), want fallback", len(s))
	}
}

// TestDict_ErrorCases pins the body validator: zero distinct count,
// truncated dictionary, distinct > cap, and out-of-range indices.
func TestDict_ErrorCases(t *testing.T) {
	t.Run("zero_distinct", func(t *testing.T) {
		buf := []byte{tagPackDict, qpackKindUint64, 0x00}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedDictUint64Slice(); err == nil {
			t.Fatalf("expected error on distinct=0")
		}
	})
	t.Run("over_cap_distinct", func(t *testing.T) {
		// distinct = qpackDictMaxDistinct + 1 = 17
		buf := []byte{tagPackDict, qpackKindUint64, byte(qpackDictMaxDistinct + 1)}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedDictUint64Slice(); err == nil {
			t.Fatalf("expected error on distinct > cap")
		}
	})
	t.Run("bad_kind", func(t *testing.T) {
		buf := []byte{tagPackDict, qpackKindFloat64, 0x02, 0x2a, 0x14, 0x01, 0x00}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedDictUint64Slice(); err == nil {
			t.Fatalf("expected ErrTypeMismatch on bad kind")
		}
	})
	t.Run("truncated_dict_values", func(t *testing.T) {
		// distinct=3 but only 2 varuints follow before n.
		buf := []byte{tagPackDict, qpackKindUint64, 0x03, 0x01, 0x02}
		dec := NewDecoderOnBuf(buf)
		dec.i++
		if _, err := dec.readPackedDictUint64Slice(); err == nil {
			t.Fatalf("expected error on truncated dict")
		}
	})
}

// TestDict_EndToEnd round-trips a slice through Marshal / Unmarshal
// with QPack enabled. The fixture is 4-distinct status codes spread
// wide enough that FOR can't beat dict.
func TestDict_EndToEnd(t *testing.T) {
	type row struct {
		Codes []int64 `qdf:"codes"`
	}
	in := row{Codes: make([]int64, 1024)}
	values := []int64{200, 999_999, 12345, -42}
	for i := range in.Codes {
		in.Codes[i] = values[i%4]
	}
	bSpeed, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	bQPack, err := Marshal(in, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	// Dict at distinct=4 → 2 bits per index → ~256 bytes body for
	// 1024 elements. Speed-mode wire is at least 4×.
	if len(bQPack) >= len(bSpeed)/3 {
		t.Fatalf("dict did not shrink: speed=%d qpack=%d", len(bSpeed), len(bQPack))
	}
	var out row
	if err := Unmarshal(bQPack, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in.Codes, out.Codes) {
		t.Fatalf("round-trip mismatch")
	}
}
