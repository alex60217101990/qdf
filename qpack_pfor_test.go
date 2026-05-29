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
		mk(100, -1000, 1, 1<<40),     // negative min, rare huge spikes
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

type pforProbe struct {
	V []uint64
}

// containsTagKind reports whether b carries an integer-slice payload framed by
// the two-byte structural header {tag, kind}. A bare byte-scan would yield
// false positives against arbitrary data bytes.
func containsTagKind(b []byte, tag, kind byte) bool {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == tag && b[i+1] == kind {
			return true
		}
	}
	return false
}

func TestPFor_PickerChoosesByData(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	// Outlier-heavy: mostly tiny values, ~1% huge spikes -> plain FOR forced wide.
	spikes := make([]uint64, 1024)
	for i := range spikes {
		if r.Intn(100) < 1 {
			spikes[i] = 1<<40 + uint64(r.Intn(1000))
		} else {
			spikes[i] = uint64(r.Intn(16))
		}
	}
	// Clean: uniform small values -> FOR already optimal, PFOR must not fire.
	clean := make([]uint64, 1024)
	for i := range clean {
		clean[i] = 1000 + uint64(r.Intn(16))
	}

	// Inspect the raw wire; exclude rANS (it wraps the body and hides the tag).
	opts := OptBalanced &^ OptRANS
	encSpikes, err := Marshal(pforProbe{V: spikes}, opts)
	if err != nil {
		t.Fatal(err)
	}
	encClean, err := Marshal(pforProbe{V: clean}, opts)
	if err != nil {
		t.Fatal(err)
	}

	if !containsTagKind(encSpikes, tagPackPFor, qpackKindUint64) {
		t.Errorf("outlier-heavy data: expected PFOR tag on wire, not found")
	}
	if containsTagKind(encClean, tagPackPFor, qpackKindUint64) {
		t.Errorf("clean data: PFOR should lose to FOR, but PFOR tag present")
	}

	// Round-trips regardless of codec choice.
	for name, enc := range map[string][]byte{"spikes": encSpikes, "clean": encClean} {
		var got pforProbe
		if err := Unmarshal(enc, &got); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
	}
	var gotS pforProbe
	if err := Unmarshal(encSpikes, &gotS); err != nil {
		t.Fatal(err)
	}
	for i := range spikes {
		if gotS.V[i] != spikes[i] {
			t.Fatalf("spikes round-trip mismatch at %d: %d != %d", i, gotS.V[i], spikes[i])
		}
	}
}

// TestPFor_CorpusGate is the measure-first gate: on a realistic latency column
// (microsecond latencies with ~1% large spikes) PFOR must beat plain FOR by a
// meaningful margin, and the codec must round-trip. The never-worse picker
// guarantees no other workload regresses; this asserts the win actually exists.
func TestPFor_CorpusGate(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	const n = 1024
	s := make([]uint64, n)
	for i := range s {
		if r.Intn(1000) < 10 { // ~1% spikes
			s[i] = 50_000 + uint64(r.Intn(950_000)) // 50ms..1s spike (in us)
		} else {
			s[i] = uint64(r.Intn(500)) // sub-millisecond latency
		}
	}
	mn, mx := minMaxU64(s)
	forBits := bitsForDelta(mx - mn)
	forSize := qpackForSizeUnsigned(n, forBits, mn)
	b, pforCost, ok := pforPlanUnsigned(s, mn, forBits)
	if !ok {
		t.Fatalf("PFOR not applicable (forBits=%d)", forBits)
	}
	t.Logf("latency_spikes_1024: FOR=%d B (bits=%d), PFOR=%d B (bits=%d) -> %.1f%% smaller",
		forSize, forBits, pforCost, b, 100*(1-float64(pforCost)/float64(forSize)))
	if pforCost >= forSize*9/10 {
		t.Fatalf("PFOR not >=10%% smaller than FOR: PFOR=%d FOR=%d", pforCost, forSize)
	}
	// Round-trip the actual encoded payload.
	var e Encoder
	e.writePackedPForUint64Slice(s, mn, b)
	var d Decoder
	d.buf = e.buf
	if err := d.readHeader(); err != nil {
		t.Fatal(err)
	}
	d.i++ // tag
	got, err := d.readPackedPForUint64Slice()
	if err != nil {
		t.Fatal(err)
	}
	for i := range s {
		if got[i] != s[i] {
			t.Fatalf("round-trip mismatch at %d", i)
		}
	}
}
