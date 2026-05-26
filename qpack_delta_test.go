package qdf

import (
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func TestDeltaForUint64_RoundTrip(t *testing.T) {
	cases := [][]uint64{
		nil,
		{42},
		{42, 42, 42, 42, 42},            // const => minDelta=0, bits=0
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, // strict +1
		{1_700_000_000, 1_700_000_001, 1_700_000_003},             // unix ns-ish
		{100, 200, 150, 250, 300, 200, 400},                       // mixed direction
		{math.MaxUint64 - 10, math.MaxUint64 - 5, math.MaxUint64}, // near MaxUint64
	}
	for _, in := range cases {
		first, minD, bp := computeDeltaStatsU64(in)
		if bp > qpackForMaxBits {
			continue
		}
		enc := NewEncoder(Fast)
		enc.writePackedDeltaForUint64Slice(in, first, minD, bp)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackDeltaFor {
			t.Fatalf("expected tagPackDeltaFor got %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedDeltaForUint64Slice()
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
			t.Fatalf("mismatch:\n in=%v\nout=%v", in, out)
		}
	}
}

func TestDeltaForInt64_RoundTrip(t *testing.T) {
	cases := [][]int64{
		{0, 1, 2, 3, 4},
		{-5, -4, -3, -2, -1, 0, 1, 2},
		{-1000, -500, 0, 500, 1000},
		{math.MaxInt64 - 100, math.MaxInt64 - 50, math.MaxInt64},
		{math.MinInt64, math.MinInt64 + 1, math.MinInt64 + 2}, // signed wrap
	}
	for _, in := range cases {
		first, minD, bp := computeDeltaStatsI64(in)
		if bp > qpackForMaxBits {
			continue
		}
		enc := NewEncoder(Fast)
		enc.writePackedDeltaForInt64Slice(in, first, minD, bp)
		dec := NewDecoderOnBuf(enc.buf)
		tag, err := dec.peekTag()
		if err != nil || tag != tagPackDeltaFor {
			t.Fatalf("tag %02x err=%v", tag, err)
		}
		dec.i++
		out, err := dec.readPackedDeltaForInt64Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("mismatch want=%v got=%v", in, out)
		}
	}
}

func TestDeltaForUint64_SkipIntegration(t *testing.T) {
	in := []uint64{1700, 1701, 1702, 1703, 1704}
	first, minD, bp := computeDeltaStatsU64(in)
	enc := NewEncoder(Fast)
	enc.writePackedDeltaForUint64Slice(in, first, minD, bp)
	enc.WriteInt(11)
	dec := NewDecoderOnBuf(enc.buf)
	if err := dec.Skip(); err != nil {
		t.Fatal(err)
	}
	v, err := dec.ReadInt()
	if err != nil || v != 11 {
		t.Fatalf("after skip v=%d err=%v", v, err)
	}
}

func TestDeltaForSizeBetter(t *testing.T) {
	// Strict monotonic: deltas are 1 -> bitsPer=0 (minDelta=maxDelta=1).
	in := make([]uint64, 1024)
	for i := range in {
		in[i] = 1_700_000_000 + uint64(i)
	}
	first, minD, bp := computeDeltaStatsU64(in)
	encDelta := NewEncoder(Fast)
	encDelta.writePackedDeltaForUint64Slice(in, first, minD, bp)
	encFor := NewEncoder(Fast)
	mnFor, mxFor := minMaxU64(in)
	encFor.writePackedForUint64Slice(in, mnFor, bitsForDelta(mxFor-mnFor))
	encRaw := NewEncoder(Fast)
	encRaw.SetQPack(true)
	encRaw.writePackedUint64Slice(in)
	t.Logf("delta=%d FOR=%d RAW=%d  delta-vs-raw=%.1fx delta-vs-for=%.1fx",
		len(encDelta.buf), len(encFor.buf), len(encRaw.buf),
		float64(len(encRaw.buf))/float64(len(encDelta.buf)),
		float64(len(encFor.buf))/float64(len(encDelta.buf)))
	if len(encDelta.buf) >= len(encFor.buf) {
		t.Fatalf("delta+FOR not smaller than plain FOR on monotonic data: %d vs %d", len(encDelta.buf), len(encFor.buf))
	}
}

func makeMonotonicU64(n int, base, jitter uint64) []uint64 {
	s := make([]uint64, n)
	rng := rand.New(rand.NewSource(99))
	v := base
	for i := range s {
		v += 1 + uint64(rng.Int63n(int64(jitter)+1))
		s[i] = v
	}
	return s
}

func BenchmarkQPackDelta_Uint64(b *testing.B) {
	for _, cfg := range []struct {
		n   int
		jit uint64
	}{
		{1024, 0},      // strict +1 -> bits=0
		{1024, 7},      // ~3-bit jitter
		{16384, 1023},  // ~10-bit jitter
		{16384, 65535}, // ~16-bit jitter
	} {
		in := makeMonotonicU64(cfg.n, 1_700_000_000, cfg.jit)
		first, mnD, bp := computeDeltaStatsU64(in)
		_ = first
		tag := strconv.Itoa(cfg.n) + "/jit" + strconv.FormatUint(cfg.jit, 10) + "/" + strconv.Itoa(bp) + "b"

		encRaw := NewEncoder(Fast)
		encRaw.SetQPack(true)
		encRaw.writePackedUint64Slice(in)

		mn, mx := minMaxU64(in)
		encFor := NewEncoder(Fast)
		encFor.writePackedForUint64Slice(in, mn, bitsForDelta(mx-mn))

		encDelta := NewEncoder(Fast)
		encDelta.writePackedDeltaForUint64Slice(in, in[0], mnD, bp)

		b.Logf("[%s] raw=%d for=%d delta=%d  raw/delta=%.1fx for/delta=%.1fx", tag,
			len(encRaw.buf), len(encFor.buf), len(encDelta.buf),
			float64(len(encRaw.buf))/float64(len(encDelta.buf)),
			float64(len(encFor.buf))/float64(len(encDelta.buf)))

		b.Run("encode/delta/"+tag, func(b *testing.B) {
			enc := NewEncoder(Fast)
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				enc.Reset()
				f, mD, bits := computeDeltaStatsU64(in)
				enc.writePackedDeltaForUint64Slice(in, f, mD, bits)
			}
		})

		deltaBuf := append([]byte(nil), encDelta.buf...)
		b.Run("decode/delta/"+tag, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(cfg.n * 8))
			for b.Loop() {
				dec := NewDecoderOnBuf(deltaBuf)
				_, _ = dec.peekTag()
				dec.i++
				_, _ = dec.readPackedDeltaForUint64Slice()
			}
		})
	}
}
