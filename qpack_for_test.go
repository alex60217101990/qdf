package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func TestBitPackUnpackRoundTrip(t *testing.T) {
	for _, bitsPer := range []int{1, 2, 3, 5, 7, 8, 11, 13, 16, 17, 23, 32, 33, 47, 56} {
		mask := uint64(1)<<uint(bitsPer) - 1
		for _, n := range []int{0, 1, 7, 8, 9, 63, 64, 65, 1000} {
			vals := make([]uint64, n)
			rng := rand.New(rand.NewSource(int64(bitsPer*1000 + n)))
			for i := range vals {
				vals[i] = rng.Uint64() & mask
			}
			body := make([]byte, (n*bitsPer+7)>>3)
			bitPackU64LE(body, vals, bitsPer)
			out := make([]uint64, n)
			bitUnpackU64LE(out, body, bitsPer)
			if !reflect.DeepEqual(vals, out) {
				t.Fatalf("bits=%d n=%d mismatch\n want=%v\n got=%v", bitsPer, n, vals[:min(8, n)], out[:min(8, n)])
			}
		}
	}
}

func TestForUint64_RoundTrip(t *testing.T) {
	cases := [][]uint64{
		nil,
		{42},
		{42, 42, 42, 42}, // const => bits=0
		{0, 1, 2, 3, 4, 5},
		{1000, 1003, 1005, 1007, 1010},
		{math.MaxUint64, math.MaxUint64 - 1, math.MaxUint64 - 7}, // small delta at top of range
	}
	for _, in := range cases {
		mn := uint64(0)
		mx := uint64(0)
		if len(in) > 0 {
			mn, mx = minMaxU64(in)
		}
		bitsPer := bitsForDelta(mx - mn)
		if bitsPer > qpackForMaxBits {
			continue
		}
		enc := NewEncoder(Fast)
		enc.writePackedForUint64Slice(in, mn, bitsPer)
		dec := NewDecoderOnBuf(enc.buf)
		// consume the tag, then call codec helper
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackFor {
			t.Fatalf("expected tagPackFor got %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedForUint64Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(in) == 0 {
			if len(out) != 0 {
				t.Fatalf("empty want, got %v", out)
			}
			continue
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch: want %v got %v", in, out)
		}
	}
}

func TestForInt64_RoundTrip(t *testing.T) {
	cases := [][]int64{
		{-5, -4, -3, -2, -1},
		{-1000, -999, -998, -997},
		{0, 0, 0, 0},
		{-1 << 30, -1<<30 + 1, -1<<30 + 2, -1<<30 + 3},
		{1, 2, 3, math.MaxInt32},
	}
	for _, in := range cases {
		mn, mx := minMaxI64(in)
		delta := uint64(mx) - uint64(mn)
		bitsPer := bitsForDelta(delta)
		if bitsPer > qpackForMaxBits {
			continue
		}
		enc := NewEncoder(Fast)
		enc.writePackedForInt64Slice(in, mn, bitsPer)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackFor {
			t.Fatalf("tag %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedForInt64Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch want=%v got=%v", in, out)
		}
	}
}

func TestForUint64_SkipIntegration(t *testing.T) {
	in := []uint64{100, 101, 102, 103, 104, 105, 106}
	mn, mx := minMaxU64(in)
	enc := NewEncoder(Fast)
	enc.writePackedForUint64Slice(in, mn, bitsForDelta(mx-mn))
	enc.WriteInt(7)
	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatal(err)
	}
	v, err := dec.ReadInt()
	if err != nil || v != 7 {
		t.Fatalf("after skip: v=%d err=%v", v, err)
	}
}

func TestForSizeBetter(t *testing.T) {
	// Compare FOR encoded size vs raw size on clustered data.
	in := make([]uint64, 1024)
	for i := range in {
		in[i] = 1_000_000 + uint64(i&0xFFF)
	}
	mn, mx := minMaxU64(in)
	bitsPer := bitsForDelta(mx - mn)
	if bitsPer > 16 {
		t.Fatalf("unexpected bits=%d", bitsPer)
	}
	encRaw := NewEncoder(Fast)
	encRaw.SetQPack(true)
	encRaw.writePackedUint64Slice(in)
	encFor := NewEncoder(Fast)
	encFor.writePackedForUint64Slice(in, mn, bitsPer)
	if len(encFor.buf) >= len(encRaw.buf) {
		t.Fatalf("FOR not smaller: FOR=%d RAW=%d", len(encFor.buf), len(encRaw.buf))
	}
	t.Logf("FOR=%d  RAW=%d  ratio=%.2fx", len(encFor.buf), len(encRaw.buf), float64(len(encRaw.buf))/float64(len(encFor.buf)))
}

func TestZigzag64(t *testing.T) {
	values := []int64{0, -1, 1, -2, 2, math.MinInt64, math.MaxInt64, -1234567890, 1234567890}
	for _, v := range values {
		if got := zigzagDecode64(zigzagEncode64(v)); got != v {
			t.Fatalf("zigzag %d != %d", got, v)
		}
	}
}

func makeClusteredU64(n int, base, span uint64) []uint64 {
	s := make([]uint64, n)
	rng := rand.New(rand.NewSource(11))
	for i := range s {
		s[i] = base + uint64(rng.Int63n(int64(span)))
	}
	return s
}

func BenchmarkQPackFor_Uint64(b *testing.B) {
	for _, cfg := range []struct {
		n    int
		base uint64
		span uint64
	}{
		{1024, 1_000_000, 1 << 12},  // 12 bits
		{16384, 1_000_000, 1 << 16}, // 16 bits
		{16384, 0, 1 << 32},         // 32 bits
	} {
		in := makeClusteredU64(cfg.n, cfg.base, cfg.span)
		mn, mx := minMaxU64(in)
		bp := bitsForDelta(mx - mn)
		tag := strconv.Itoa(cfg.n) + "/" + strconv.Itoa(bp) + "b"
		b.Run("encode/raw/"+tag, func(b *testing.B) {
			enc := NewEncoder(Fast)
			enc.SetQPack(true)
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				enc.Reset()
				enc.writePackedUint64Slice(in)
			}
		})
		b.Run("encode/for/"+tag, func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				enc.Reset()
				enc.writePackedForUint64Slice(in, mn, bp)
			}
		})
		// Size comparison print.
		encRaw := NewEncoder(Fast)
		encRaw.SetQPack(true)
		encRaw.writePackedUint64Slice(in)
		encFor := NewEncoder(Fast)
		encFor.writePackedForUint64Slice(in, mn, bp)
		b.Logf("size raw=%d for=%d  ratio=%.2fx", len(encRaw.buf), len(encFor.buf), float64(len(encRaw.buf))/float64(len(encFor.buf)))

		// Decode benches
		rawBuf := append([]byte(nil), encRaw.buf...)
		forBuf := append([]byte(nil), encFor.buf...)
		b.Run("decode/raw/"+tag, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(rawBuf)
				_, _ = dec.peekTag()
				dec.i++
				_, _ = dec.readPackedUint64Slice()
			}
		})
		b.Run("decode/for/"+tag, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(forBuf)
				_, _ = dec.peekTag()
				dec.i++
				_, _ = dec.readPackedForUint64Slice()
			}
		})
	}
}
