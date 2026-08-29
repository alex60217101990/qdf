package internarena

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// Round-trip on a small set: every id resolves back to its
// original payload bytes.
func TestArena_PutGetRoundTrip(t *testing.T) {
	a := &Arena{}
	cases := []string{
		"", "a", "ab", "hello", "the quick brown fox",
		"привет", "こんにちは", "🌍🚀",
		strings.Repeat("x", 1000),
	}
	ids := make([]uint32, len(cases))
	for i, s := range cases {
		ids[i] = a.Put(s)
	}
	for i, s := range cases {
		got := a.Get(ids[i])
		if string(got) != s {
			t.Fatalf("[%d] put %q got %q", i, s, got)
		}
	}
}

// Many small puts across multiple chunks. Verifies chunk-growth +
// loc-encoding (chunk_idx packing) survive a non-trivial run.
func TestArena_ManyPutsAcrossChunks(t *testing.T) {
	a := &Arena{}
	const N = 1024
	want := make([]string, N)
	ids := make([]uint32, N)
	for i := range N {
		want[i] = fmt.Sprintf("entry-%05d-%s", i, strings.Repeat("z", i%32))
		ids[i] = a.Put(want[i])
	}
	if len(a.chunks) < 2 {
		t.Fatalf("expected at least 2 chunks after %d puts, got %d", N, len(a.chunks))
	}
	for i := range N {
		if got := string(a.Get(ids[i])); got != want[i] {
			t.Fatalf("[%d] put %q got %q", i, want[i], got)
		}
	}
}

// Reset rolls cursors back to chunks[0]. Subsequent Puts reuse the
// existing first chunk. No new chunk should appear until the total
// post-Reset volume exceeds cap(chunks[0]).
func TestArena_ResetReusesChunks(t *testing.T) {
	a := &Arena{}
	for i := range 100 {
		a.Put(fmt.Sprintf("preset-%d", i))
	}
	preChunks := len(a.chunks)
	if preChunks == 0 {
		t.Fatal("expected chunks after warmup")
	}
	a.Reset()
	if a.Len() != 0 {
		t.Fatalf("Reset did not clear locs: %d", a.Len())
	}
	// Re-fill with smaller volume than chunks[0]; chunks slice must
	// not grow.
	for i := range 10 {
		a.Put(fmt.Sprintf("post-%d", i))
	}
	if len(a.chunks) != preChunks {
		t.Fatalf("Reset re-allocated chunks: pre=%d post=%d", preChunks, len(a.chunks))
	}
}

// Reset followed by a volume that exceeds the first chunk walks
// into the second existing chunk before allocating a new one.
func TestArena_ResetWalksExistingChunks(t *testing.T) {
	a := &Arena{}
	// Fill enough to allocate three chunks (4 KiB + 8 KiB + 16 KiB).
	for i := range 4000 {
		a.Put(fmt.Sprintf("entry-%05d-%s", i, strings.Repeat("z", 8)))
	}
	preChunks := len(a.chunks)
	if preChunks < 3 {
		t.Fatalf("expected ≥3 chunks for the test, got %d", preChunks)
	}
	a.Reset()
	for i := range 4000 {
		a.Put(fmt.Sprintf("entry-%05d-%s", i, strings.Repeat("z", 8)))
	}
	if len(a.chunks) != preChunks {
		t.Fatalf("post-reset chunk count grew: pre=%d post=%d", preChunks, len(a.chunks))
	}
}

// A payload that overflows the default initial chunk must trigger a
// grow path that allocates a chunk large enough to hold it.
func TestArena_LargePayloadForcesGrowth(t *testing.T) {
	a := &Arena{}
	big := strings.Repeat("x", 8*1024) // 8 KiB > initialChunkBytes
	id := a.Put(big)
	got := a.Get(id)
	if string(got) != big {
		t.Fatalf("large payload corrupted: len=%d", len(got))
	}
}

// Get must return an aliased slice that reflects the underlying
// chunk bytes. Mutating the returned slice MUST mutate the arena's
// storage (proves alias, not copy). This is documented "do-not-do"
// behavior but the test guarantees the aliasing contract.
func TestArena_GetReturnsAlias(t *testing.T) {
	a := &Arena{}
	id := a.Put("mutable")
	got := a.Get(id)
	got[0] = 'M'
	again := a.Get(id)
	if string(again) != "Mutable" {
		t.Fatalf("Get does not alias: %q", again)
	}
}

// pack / unpack symmetry across the full range we encode.
func TestArena_PackUnpack(t *testing.T) {
	cases := []struct {
		chunk  uint16
		offset uint32
		length uint16
	}{
		{0, 0, 0},
		{1, 100, 16},
		{0xFFFF, 0xFFFFFFFF, 0xFFFF},
		{42, 1024, 64},
	}
	for _, c := range cases {
		loc := pack(c.chunk, c.offset, c.length)
		ci, off, ln := unpack(loc)
		if ci != c.chunk || off != c.offset || ln != c.length {
			t.Fatalf("pack/unpack mismatch: in=(%d,%d,%d) out=(%d,%d,%d)",
				c.chunk, c.offset, c.length, ci, off, ln)
		}
	}
}

// BytesUsed accounting tracks per-chunk used bytes across growth.
func TestArena_BytesUsedAccounting(t *testing.T) {
	a := &Arena{}
	total := 0
	for i := range 200 {
		s := fmt.Sprintf("entry-%05d", i)
		a.Put(s)
		total += len(s)
	}
	if got := a.BytesUsed(); got != total {
		t.Fatalf("BytesUsed mismatch: got=%d want=%d chunks=%d", got, total, len(a.chunks))
	}
}

// BytesPut is an O(1) running counter of payload bytes stored since the last
// Reset, used as the adaptive-retention trigger. It must equal the exact sum of
// Put lengths, zero after Reset, and — unlike BytesUsed — must NOT count the
// capacity of a chunk that grow() skipped because a payload did not fit.
func TestArena_BytesPut(t *testing.T) {
	a := &Arena{}
	total := 0
	for i := range 200 {
		s := fmt.Sprintf("entry-%05d", i)
		a.Put(s)
		total += len(s)
	}
	if got := a.BytesPut(); got != total {
		t.Fatalf("BytesPut=%d want=%d", got, total)
	}
	a.Reset()
	if got := a.BytesPut(); got != 0 {
		t.Fatalf("BytesPut after Reset=%d want=0", got)
	}

	// Skipped-chunk case: a fresh arena's chunk[0] is 4 KiB; a single payload
	// larger than that forces grow() to walk past (skip) chunk[0] and allocate a
	// big chunk. BytesPut must report ONLY the real payload, while BytesUsed
	// over-counts the skipped chunk[0]'s full capacity.
	a2 := &Arena{}
	a2.Put("warm") // lands in chunk[0] (4 KiB)
	a2.Reset()     // retains chunk[0]; cursor back to 0
	big := make([]byte, 10*1024)
	a2.Put(string(big)) // 10 KiB: skips 4 KiB chunk[0], allocates a bigger one
	if got := a2.BytesPut(); got != len(big) {
		t.Fatalf("BytesPut with skipped chunk=%d want=%d (must not count skipped capacity)", got, len(big))
	}
	if a2.BytesUsed() <= a2.BytesPut() {
		t.Fatalf("expected BytesUsed (%d) to over-count vs BytesPut (%d) when a chunk was skipped", a2.BytesUsed(), a2.BytesPut())
	}
}

// Randomized round-trip: fuzz-style coverage that mixed lengths /
// orderings cannot corrupt loc indexing across chunk growth.
func TestArena_RandomRoundTrip(t *testing.T) {
	a := &Arena{}
	rnd := rand.New(rand.NewSource(1))
	const N = 5000
	keys := make([]string, N)
	ids := make([]uint32, N)
	for i := range N {
		ln := rnd.Intn(64) + 1
		buf := make([]byte, ln)
		for j := range buf {
			buf[j] = byte('a' + rnd.Intn(26))
		}
		keys[i] = string(buf)
		ids[i] = a.Put(keys[i])
	}
	// Read out of order.
	for _, j := range rnd.Perm(N) {
		got := a.Get(ids[j])
		if !bytes.Equal(got, []byte(keys[j])) {
			t.Fatalf("[%d] put %q got %q", j, keys[j], got)
		}
	}
}

// Burst growth + Reset under the default soft cap must drop the
// spike chunks so the arena's resident memory stays bounded.
func TestArena_ResetShrinksAfterBurst(t *testing.T) {
	a := &Arena{}
	// Burst: enough payload to grow past DefaultRetainBytes.
	huge := strings.Repeat("x", 128*1024) // 128 KiB single payload
	for range 4 {
		a.Put(huge)
	}
	if len(a.chunks) < 2 {
		t.Fatalf("test premise: burst should produce ≥2 chunks, got %d", len(a.chunks))
	}
	burstChunks := len(a.chunks)
	burstCap := 0
	for _, c := range a.chunks {
		burstCap += cap(c)
	}
	if burstCap <= DefaultRetainBytes {
		t.Fatalf("test premise: burst should exceed DefaultRetainBytes (%d), got %d",
			DefaultRetainBytes, burstCap)
	}

	a.Reset()
	if len(a.chunks) != 1 {
		t.Fatalf("Reset did not drop spike chunks: pre=%d post=%d", burstChunks, len(a.chunks))
	}
	// After shrink, total chunk cap must be small (≤ chunks[0]).
	if cap(a.chunks[0]) > DefaultRetainBytes {
		t.Fatalf("chunks[0] itself exceeds retain cap: %d", cap(a.chunks[0]))
	}
}

// Steady-state encoder workload stays below the soft cap; Reset
// must NOT drop any chunks in that regime.
func TestArena_ResetKeepsChunksUnderThreshold(t *testing.T) {
	a := &Arena{}
	for i := range 1000 {
		a.Put(fmt.Sprintf("steady-state-%05d", i))
	}
	preChunks := len(a.chunks)
	preCap := 0
	for _, c := range a.chunks {
		preCap += cap(c)
	}
	if preCap > DefaultRetainBytes {
		t.Skipf("steady-state workload exceeded retain cap (%d > %d) — skipping shrink-keep test",
			preCap, DefaultRetainBytes)
	}
	a.Reset()
	if len(a.chunks) != preChunks {
		t.Fatalf("Reset dropped chunks under threshold: pre=%d post=%d", preChunks, len(a.chunks))
	}
}

// ResetWithLimit(0) disables the cap entirely.
func TestArena_ResetWithLimitZeroNoShrink(t *testing.T) {
	a := &Arena{}
	huge := strings.Repeat("x", 128*1024)
	for range 4 {
		a.Put(huge)
	}
	pre := len(a.chunks)
	a.ResetWithLimit(0)
	if len(a.chunks) != pre {
		t.Fatalf("ResetWithLimit(0) dropped chunks: pre=%d post=%d", pre, len(a.chunks))
	}
}

// After a shrink the arena must still be operational — subsequent
// Puts return valid ids that resolve back to original bytes.
func TestArena_OperationalAfterShrink(t *testing.T) {
	a := &Arena{}
	huge := strings.Repeat("x", 128*1024)
	for range 4 {
		a.Put(huge)
	}
	a.Reset()

	// Reuse the shrunk arena.
	const N = 100
	want := make([]string, N)
	ids := make([]uint32, N)
	for i := range N {
		want[i] = fmt.Sprintf("post-shrink-%d", i)
		ids[i] = a.Put(want[i])
	}
	for i := range N {
		if got := string(a.Get(ids[i])); got != want[i] {
			t.Fatalf("[%d] post-shrink Get mismatch: got=%q want=%q", i, got, want[i])
		}
	}
}

// BenchmarkArena_Put vs strings.Clone — the headline replacement.
// strings.Clone heap-allocates one block per call; the arena bulks
// the payload into a slab and only allocates on chunk growth.
func BenchmarkArena_Put(b *testing.B) {
	const s = "service-billing-region-eu-west-1-host-ip-10-0-42"
	a := &Arena{}
	b.ReportAllocs()
	b.SetBytes(int64(len(s)))
	for b.Loop() {
		_ = a.Put(s)
		// Reset every 1024 entries to keep the arena bounded.
		if a.Len() >= 1024 {
			a.Reset()
		}
	}
}

func BenchmarkStringsClone(b *testing.B) {
	const s = "service-billing-region-eu-west-1-host-ip-10-0-42"
	b.ReportAllocs()
	b.SetBytes(int64(len(s)))
	sink := ""
	for b.Loop() {
		sink = strings.Clone(s)
	}
	_ = sink
}

// BenchmarkArena_PutVarLen exercises a more realistic spread: mixed
// short / medium / long payloads, alternating, with periodic Reset.
func BenchmarkArena_PutVarLen(b *testing.B) {
	corpus := []string{
		"prod",
		"eu-west-1",
		"service-billing",
		"ip-10-0-42",
		"a longer key-ish payload that crosses a few cache lines maybe",
		strings.Repeat("x", 200),
	}
	a := &Arena{}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		_ = a.Put(corpus[i%len(corpus)])
		i++
		if a.Len() >= 1024 {
			a.Reset()
			i = 0
		}
	}
}
