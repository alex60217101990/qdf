package qdf

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func pforTestSlicesU64() [][]uint64 {
	r := rand.New(rand.NewSource(1))
	mk := func(n int, base uint64, spikePct int, spike uint64) []uint64 {
		s := make([]uint64, n)
		for i := range s {
			if spikePct > 0 && r.Intn(100) < spikePct {
				s[i] = spike + uint64(r.Intn(1000))
			} else {
				s[i] = base + uint64(r.Intn(16))
			}
		}
		return s
	}
	return [][]uint64{
		{}, {5}, {5, 5, 5},
		mk(100, 1000, 1, 1<<40), // rare huge spikes
		mk(1000, 200, 3, 1<<20), // 3% spikes
		mk(257, 0, 0, 0),        // no spikes
	}
}

func pforTestSlicesI64() [][]int64 {
	r := rand.New(rand.NewSource(2))
	mk := func(n int, base int64, spikePct int, spike int64) []int64 {
		s := make([]int64, n)
		for i := range s {
			if spikePct > 0 && r.Intn(100) < spikePct {
				s[i] = spike + int64(r.Intn(1000))
			} else {
				s[i] = base + int64(r.Intn(16))
			}
		}
		return s
	}
	return [][]int64{
		mk(100, -1000, 1, 1<<40),    // negative min, rare huge spikes
		mk(1000, -50, 3, -(1 << 20)), // 3% negative spikes
		mk(257, 7, 0, 0),             // no spikes
		mk(512, -(1 << 30), 2, 1<<45),
	}
}

func TestPFor_RoundTripI64(t *testing.T) {
	for ci, s := range pforTestSlicesI64() {
		mn, mx := minMaxI64(s)
		forBits := bitsForDelta(uint64(mx) - uint64(mn))
		b, _, ok := pforPlanSigned(s, mn, forBits)
		if !ok {
			continue
		}
		var e Encoder
		e.writePackedPForInt64Slice(s, mn, b)
		var d Decoder
		d.buf = e.buf
		d.i = 0
		if err := d.readHeader(); err != nil {
			t.Fatalf("case %d header: %v", ci, err)
		}
		if d.buf[d.i] != tagPackPFor {
			t.Fatalf("case %d: expected tagPackPFor", ci)
		}
		d.i++
		got, err := d.readPackedPForInt64Slice()
		if err != nil {
			t.Fatalf("case %d decode: %v", ci, err)
		}
		if len(got) != len(s) {
			t.Fatalf("case %d len %d != %d", ci, len(got), len(s))
		}
		for i := range s {
			if got[i] != s[i] {
				t.Fatalf("case %d i=%d: got %d want %d", ci, i, got[i], s[i])
			}
		}
	}
}

func TestPFor_RoundTripU64(t *testing.T) {
	for ci, s := range pforTestSlicesU64() {
		if len(s) == 0 {
			continue
		}
		mn, mx := minMaxU64(s)
		forBits := bitsForDelta(mx - mn)
		b, _, ok := pforPlanUnsigned(s, mn, forBits)
		if !ok {
			continue // PFOR not applicable for this slice
		}
		var e Encoder
		e.writePackedPForUint64Slice(s, mn, b)
		var d Decoder
		d.buf = e.buf
		d.i = 0
		if err := d.readHeader(); err != nil {
			t.Fatalf("case %d header: %v", ci, err)
		}
		if d.buf[d.i] != tagPackPFor {
			t.Fatalf("case %d: expected tagPackPFor", ci)
		}
		d.i++
		got, err := d.readPackedPForUint64Slice()
		if err != nil {
			t.Fatalf("case %d decode: %v", ci, err)
		}
		if len(got) != len(s) {
			t.Fatalf("case %d len %d != %d", ci, len(got), len(s))
		}
		for i := range s {
			if got[i] != s[i] {
				t.Fatalf("case %d i=%d: got %d want %d", ci, i, got[i], s[i])
			}
		}
	}
}

func FuzzPForRoundTrip(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add(make([]byte, 256))
	f.Fuzz(func(t *testing.T, raw []byte) {
		n := len(raw) / 8
		if n == 0 {
			return
		}
		s := make([]uint64, n)
		for i := range s {
			s[i] = binary.LittleEndian.Uint64(raw[i*8:])
		}
		mn, mx := minMaxU64(s)
		forBits := bitsForDelta(mx - mn)
		b, _, ok := pforPlanUnsigned(s, mn, forBits)
		if !ok {
			return
		}
		var e Encoder
		e.writePackedPForUint64Slice(s, mn, b)
		var d Decoder
		d.buf = e.buf
		d.i = 0
		if err := d.readHeader(); err != nil {
			t.Fatalf("header: %v", err)
		}
		d.i++ // tag
		got, err := d.readPackedPForUint64Slice()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got) != len(s) {
			t.Fatalf("len %d != %d", len(got), len(s))
		}
		for i := range s {
			if got[i] != s[i] {
				t.Fatalf("i=%d got %d want %d", i, got[i], s[i])
			}
		}
	})
}
