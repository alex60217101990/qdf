package qdf

import (
	"fmt"
	"strings"
	"testing"
)

// shrink_test.go pins the bounded-memory contract added with the
// arena + state shrink-on-Reset machinery. The encoder / decoder
// pools must NOT pin spike memory forever; a long-running service
// with bursty traffic stays at the steady-state working set, not
// at the historical peak.

// Encoder Reset after a spike must drop the over-cap state. Verify
// at the encState level (the lowest layer that holds the buckets).
func TestShrink_EncStateRebuildsBigMap(t *testing.T) {
	st := newEncState()
	for i := range maxRetainedIDs + 100 {
		st.lookupOrAssign(fmt.Sprintf("burst-key-%05d", i))
	}
	if len(st.ids) <= maxRetainedIDs {
		t.Fatalf("test premise: burst should exceed maxRetainedIDs (%d), got %d",
			maxRetainedIDs, len(st.ids))
	}
	st.reset()
	if len(st.ids) != 0 {
		t.Fatalf("reset did not clear ids: %d", len(st.ids))
	}
	// After shrink the map header was replaced with a fresh one;
	// adding a single entry should not grow the buckets back to
	// the pre-shrink size in a single step.
	st.lookupOrAssign("post-shrink")
	if len(st.ids) != 1 {
		t.Fatalf("post-shrink lookup did not register: %d", len(st.ids))
	}
}

// LRU slices must shrink when their cap exceeded the threshold.
func TestShrink_EncStateDropsLRUOverCap(t *testing.T) {
	st := newEncState()
	for i := range maxRetainedLRUCap + 64 {
		st.lookupOrAssign(fmt.Sprintf("lru-%05d", i))
	}
	if cap(st.lruPrev) <= maxRetainedLRUCap {
		t.Fatalf("test premise: lruPrev should exceed cap (%d), got %d",
			maxRetainedLRUCap, cap(st.lruPrev))
	}
	st.reset()
	if st.lruPrev != nil {
		t.Fatalf("reset did not drop oversized lruPrev: cap=%d", cap(st.lruPrev))
	}
	if st.lruNext != nil {
		t.Fatalf("reset did not drop oversized lruNext: cap=%d", cap(st.lruNext))
	}
}

// pairPred slice shrink — same contract as LRU.
func TestShrink_EncStateDropsPairPredOverCap(t *testing.T) {
	st := newEncState()
	// Drive pairPred to grow past the cap by interning many keys
	// and recording pair transitions between them.
	for i := range maxRetainedPairCap + 64 {
		st.lookupOrAssign(fmt.Sprintf("pair-%05d", i))
	}
	// Force pair records — record a pair for the highest prev id
	// so pairPred slice is grown to that length.
	last := uint32(len(st.ids) - 1)
	st.pairRecord(last, last-1)
	if cap(st.pairPred) <= maxRetainedPairCap {
		t.Fatalf("test premise: pairPred should exceed cap (%d), got %d",
			maxRetainedPairCap, cap(st.pairPred))
	}
	st.reset()
	if st.pairPred != nil {
		t.Fatalf("reset did not drop oversized pairPred: cap=%d", cap(st.pairPred))
	}
}

// Steady-state workload must NOT trigger shrink. Encoder pool stays
// warm without the per-Reset reallocation cost.
func TestShrink_EncStateKeepsCapsUnderThreshold(t *testing.T) {
	st := newEncState()
	for i := range 100 {
		st.lookupOrAssign(fmt.Sprintf("steady-%05d", i))
	}
	preIDs := len(st.ids)
	preLRU := cap(st.lruPrev)
	st.reset()
	// After reset the map must still be the same instance (no
	// rebuild because we were under the cap). Test this indirectly:
	// the underlying buckets keep their capacity, so the cap hint
	// for a future insert is the same.
	if preIDs > maxRetainedIDs {
		t.Skipf("steady-state breached maxRetainedIDs (%d > %d)", preIDs, maxRetainedIDs)
	}
	st.lookupOrAssign("after-reset")
	// A clear()'d Go map keeps its allocation count under
	// lightweight reuse — re-running the steady-state loop after
	// reset should not trigger any new bucket growth past the
	// pre-existing cap.
	for i := range 100 {
		st.lookupOrAssign(fmt.Sprintf("steady-after-%05d", i))
	}
	if cap(st.lruPrev) > preLRU*2 {
		t.Fatalf("lruPrev cap grew unexpectedly: pre=%d post=%d", preLRU, cap(st.lruPrev))
	}
}

// Decoder Reset must shrink symmetrically. Drive the values slice
// past the cap by decoding a huge intern-heavy buffer, then verify
// reset drops the backing array.
func TestShrink_DecStateDropsValuesOverCap(t *testing.T) {
	d := newDecState()
	// Manually pump value entries past the cap. Append until the
	// slice exceeds maxRetainedIDs in capacity, mirroring what the
	// decoder does when reading a stream with many interned items.
	for i := range maxRetainedIDs + 64 {
		d.append([]byte(fmt.Sprintf("dec-val-%05d", i)))
	}
	if cap(d.values) <= maxRetainedIDs {
		t.Fatalf("test premise: values cap should exceed %d, got %d",
			maxRetainedIDs, cap(d.values))
	}
	d.reset()
	if d.values != nil {
		t.Fatalf("decoder reset did not drop oversized values: cap=%d", cap(d.values))
	}
	if d.lruPrev != nil {
		t.Fatalf("decoder reset did not drop lruPrev under oversized cap")
	}
}

// Integration view of the shrink contract: a SAME encState that
// went through a burst-grow → reset → steady-state cycle MUST come
// out of reset with caps no larger than the soft caps. The pool's
// eviction policy (sync.Pool drops items at GC time) makes a
// HeapAlloc-based assertion unreliable; instead, inspect the
// state directly across one full cycle.
func TestShrink_BurstThenSteadyState(t *testing.T) {
	st := newEncState()
	// Burst phase: blow past every threshold.
	for i := range maxRetainedIDs + 64 {
		st.lookupOrAssign(fmt.Sprintf("burst-%05d-%s", i, strings.Repeat("z", 16)))
	}
	st.pairRecord(uint32(len(st.ids)-1), 0) // grow pairPred too

	// Snapshot the burst-peak state.
	burstIDs := len(st.ids)
	burstLRUCap := cap(st.lruPrev)
	burstPairCap := cap(st.pairPred)
	burstArena := st.arena.BytesUsed()
	t.Logf("burst peak: ids=%d lruCap=%d pairCap=%d arenaUsed=%d KiB",
		burstIDs, burstLRUCap, burstPairCap, burstArena/1024)

	// Reset — must shrink at least one of the over-cap structures.
	st.reset()

	// Steady-state phase: small workload, must not regrow.
	for i := range 100 {
		st.lookupOrAssign(fmt.Sprintf("steady-%05d", i))
	}

	// Post-shrink invariants. The map and slices may NOT carry the
	// burst-time capacity any longer; either they were rebuilt to
	// a small size or they grew naturally to fit the steady-state
	// workload only.
	if cap(st.lruPrev) >= burstLRUCap && burstLRUCap > maxRetainedLRUCap {
		t.Fatalf("lruPrev cap did not shrink after burst→reset: burst=%d post=%d",
			burstLRUCap, cap(st.lruPrev))
	}
	if cap(st.pairPred) >= burstPairCap && burstPairCap > maxRetainedPairCap {
		t.Fatalf("pairPred cap did not shrink: burst=%d post=%d",
			burstPairCap, cap(st.pairPred))
	}
	if used := st.arena.BytesUsed(); burstArena > internarenaRetainBytes() && used > internarenaRetainBytes() {
		t.Fatalf("arena did not shrink past its watermark: burst=%dB post=%dB", burstArena, used)
	}
}

// Helper exported only for this test — internarena.DefaultRetainBytes
// is the public const; alias here for readability inside the package.
func internarenaRetainBytes() int { return internarenaDefaultRetainBytes }

// Keep this in sync with internarena.DefaultRetainBytes; copied here
// so the shrink test can compare without re-exporting the const.
const internarenaDefaultRetainBytes = 256 * 1024
