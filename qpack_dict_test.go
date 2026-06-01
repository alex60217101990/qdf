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
	codec, _, _, _, _, _, _, _ := pickU64Codec(s)
	if codec != qpackDict {
		t.Fatalf("u64 picker: got codec %d, want qpackDict (%d) on 4-distinct/wide-range input", codec, qpackDict)
	}

	si := make([]int64, 256)
	dictI := []int64{-1_000_000, 0, 500_000, -500_000}
	for i := range si {
		si[i] = dictI[i%4]
	}
	codec, _, _, _, _, _, _, _ = pickI64Codec(si)
	if codec != qpackDict {
		t.Fatalf("i64 picker: got codec %d, want qpackDict (%d) on 4-distinct/wide-range input", codec, qpackDict)
	}
}

// TestDict_PickerSkipsHighCardinality makes sure the probe early-
// exits and the picker never selects dict when distinct > cap.
func TestDict_PickerSkipsHighCardinality(t *testing.T) {
	s := make([]uint64, 256)
	for i := range s {
		s[i] = uint64(i * 113) // 256 unique values, well above the cap
	}
	codec, _, _, _, _, _, _, _ := pickU64Codec(s)
	if codec == qpackDict {
		t.Fatalf("u64 picker: returned qpackDict on >cap distinct (=%d), want fallback", len(s))
	}
}

// TestDict_PickerHitsMidCardinality covers the 17..64-distinct band the
// cap=64 bump unlocked: wide values (FOR would need 30 bits) but only a
// few dozen distinct (6 index bits) — dict must win on size.
func TestDict_PickerHitsMidCardinality(t *testing.T) {
	const distinct = 40
	dictVals := make([]uint64, distinct)
	for i := range dictVals {
		dictVals[i] = uint64(i) * 25_000_000 // spread across ~30 bits
	}
	s := make([]uint64, 1024)
	for i := range s {
		s[i] = dictVals[i%distinct]
	}
	codec, _, _, _, _, _, _, _ := pickU64Codec(s)
	if codec != qpackDict {
		t.Fatalf("u64 picker: got codec %d, want qpackDict (%d) on %d-distinct/wide-range input", codec, qpackDict, distinct)
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
		// distinct = qpackDictMaxDistinct + 1 (just over the cap)
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

// BenchmarkPickU64Codec_Probe measures the picker's probe cost across
// cardinality bands. The high-card case is the cap-bump regression
// guard: the distinct probe must bail at the (cap+1)th value, so encode
// time stays flat regardless of qpackDictMaxDistinct. The mid-card case
// exercises the 17..64-distinct band the cap=64 bump newly admits.
func BenchmarkPickU64Codec_Probe(b *testing.B) {
	mk := func(distinct int, wide bool) []uint64 {
		s := make([]uint64, 1024)
		for i := range s {
			v := uint64(i % distinct)
			if wide {
				v *= 25_000_000
			}
			s[i] = v
		}
		return s
	}
	cases := []struct {
		name string
		s    []uint64
	}{
		{"highcard_random", func() []uint64 {
			s := make([]uint64, 1024)
			x := uint64(1)
			for i := range s {
				x = x*6364136223846793005 + 1442695040888963407 // LCG, ~all distinct
				s[i] = x
			}
			return s
		}()},
		{"midcard40_wide", mk(40, true)},
		{"lowcard8_wide", mk(8, true)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _, _, _, _, _, _ = pickU64Codec(c.s)
			}
		})
	}
}

// TestDict_MidCardZeroRoundTrip exercises the 17..64-distinct band with
// a 0 in the dictionary, pinning the open-addressed index map (an empty
// slot is also zero, so the 0 key must still resolve to its own index).
// Round-trips through the real Marshal / Unmarshal path.
func TestDict_MidCardZeroRoundTrip(t *testing.T) {
	type urow struct {
		Codes []uint64 `qdf:"codes"`
	}
	for _, distinct := range []int{17, 33, 64} {
		vals := make([]uint64, distinct)
		for i := range vals {
			vals[i] = uint64(i) * 1_000_003 // includes 0 at i==0, wide spread
		}
		s := make([]uint64, 500)
		for i := range s {
			s[i] = vals[i%distinct]
		}
		if codec, _, _, _, _, _, _, _ := pickU64Codec(s); codec != qpackDict {
			t.Fatalf("distinct=%d: picker chose %d, want dict", distinct, codec)
		}
		b, err := Marshal(urow{Codes: s}, OptQPack)
		if err != nil {
			t.Fatalf("distinct=%d: marshal: %v", distinct, err)
		}
		var out urow
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("distinct=%d: unmarshal: %v", distinct, err)
		}
		if !reflect.DeepEqual(s, out.Codes) {
			t.Fatalf("distinct=%d: roundtrip mismatch", distinct)
		}
	}

	// Signed mirror: 40 distinct spanning negatives and including 0.
	type irow struct {
		Codes []int64 `qdf:"codes"`
	}
	dictI := make([]int64, 40)
	for i := range dictI {
		dictI[i] = int64(i-20) * 777_777 // spans negatives, includes 0
	}
	si := make([]int64, 600)
	for i := range si {
		si[i] = dictI[i%len(dictI)]
	}
	if codec, _, _, _, _, _, _, _ := pickI64Codec(si); codec != qpackDict {
		t.Fatalf("i64 mid-card: picker chose %d, want dict", codec)
	}
	b, err := Marshal(irow{Codes: si}, OptQPack)
	if err != nil {
		t.Fatalf("i64 marshal: %v", err)
	}
	var out irow
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("i64 unmarshal: %v", err)
	}
	if !reflect.DeepEqual(si, out.Codes) {
		t.Fatalf("i64 roundtrip mismatch")
	}
}
