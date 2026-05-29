package qdf

import (
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
