package qdf

import (
	"fmt"
	"strings"
	"testing"
)

// shrink_test.go pins the ADAPTIVE-retention contract of the encoder /
// decoder state pools (state.go reset()).
//
// The contract changed from "drop oversized backings on the very next
// reset" to "retain them while the workload stays large, release only
// after retainReleaseStreak consecutive small messages." This lets a
// STEADY high-cardinality / large-batch workload (AD / log / telemetry
// sync) amortize the table allocation instead of reallocating every
// message, while a one-off burst followed by quiet still returns to the
// steady-state working set (and sync.Pool's GC eviction bounds idle
// retention regardless).

// A single small lookup still registers after a burst+release cycle.
func TestShrink_EncStateRebuildsBigMap(t *testing.T) {
	st := newEncState()
	for i := range maxRetainedIDs + 100 {
		st.lookupOrAssign(fmt.Sprintf("burst-key-%05d", i))
	}
	if int(st.internLoad) <= maxRetainedIDs {
		t.Fatalf("test premise: burst should exceed maxRetainedIDs (%d), got %d",
			maxRetainedIDs, int(st.internLoad))
	}
	st.reset()
	if int(st.internLoad) != 0 {
		t.Fatalf("reset did not clear ids: %d", int(st.internLoad))
	}
	st.lookupOrAssign("post-shrink")
	if int(st.internLoad) != 1 {
		t.Fatalf("post-shrink lookup did not register: %d", int(st.internLoad))
	}
}

// Core amortization guarantee: under a SUSTAINED large workload the intern
// table is retained across resets (cleared in place), never rebuilt to the
// init size — so steady-state encoding pays no per-message table regrow.
func TestShrink_SteadyLargeRetainsInternTable(t *testing.T) {
	st := newEncState()
	var capAfterFirst int
	for cycle := range 20 {
		for i := range maxRetainedIDs + 200 { // large every cycle
			st.lookupOrAssign(fmt.Sprintf("c%02d-key-%05d", cycle, i))
		}
		st.reset()
		if cycle == 0 {
			capAfterFirst = cap(st.internTable)
		}
	}
	if cap(st.internTable) <= internTableInitSize {
		t.Fatalf("steady-large intern table was rebuilt to init size (no amortization): cap=%d", cap(st.internTable))
	}
	if cap(st.internTable) < capAfterFirst {
		t.Fatalf("steady-large intern table shrank under sustained load: first=%d last=%d",
			capAfterFirst, cap(st.internTable))
	}
}

// The encoder's LRU chain used to be retained and released here. It is gone:
// nothing on the encode side ever read it. State-ref ranks come from the MRU
// ring (encState.mruRank), and a ring miss emits the raw id rather than walking
// a chain — so the chain was maintained on every emission and consumed by no
// one. The decoder keeps its own, which decState still exercises.


// pairPred slice — same retain-then-release contract as LRU.
func TestShrink_EncStateReleasesPairPredAfterStreak(t *testing.T) {
	st := newEncState()
	for i := range maxRetainedPairCap + 64 {
		st.lookupOrAssign(fmt.Sprintf("pair-%05d", i))
	}
	last := uint32(int(st.internLoad) - 1)
	st.pairRecord(last, last-1)
	if cap(st.pairPred) <= maxRetainedPairCap {
		t.Fatalf("test premise: pairPred should exceed cap (%d), got %d",
			maxRetainedPairCap, cap(st.pairPred))
	}
	burstCap := cap(st.pairPred)

	st.reset()
	if cap(st.pairPred) != burstCap {
		t.Fatalf("pairPred dropped on first post-burst reset (should retain): burst=%d post=%d",
			burstCap, cap(st.pairPred))
	}
	for range retainReleaseStreak {
		st.reset()
	}
	if st.pairPred != nil {
		t.Fatalf("pairPred not released after %d small resets: cap=%d", retainReleaseStreak, cap(st.pairPred))
	}
}

// Steady-state SMALL workload must not grow caps unexpectedly (reuse in place).
func TestShrink_EncStateKeepsCapsUnderThreshold(t *testing.T) {
	st := newEncState()
	for i := range 100 {
		st.lookupOrAssign(fmt.Sprintf("steady-%05d", i))
	}
	preIDs := int(st.internLoad)
	preTable := cap(st.internTable)
	st.reset()
	if preIDs > maxRetainedIDs {
		t.Skipf("steady-state breached maxRetainedIDs (%d > %d)", preIDs, maxRetainedIDs)
	}
	st.lookupOrAssign("after-reset")
	for i := range 100 {
		st.lookupOrAssign(fmt.Sprintf("steady-after-%05d", i))
	}
	// The intern table is what this test now watches: the encoder's LRU chain
	// it used to check is gone, since nothing on the encode side read it.
	if cap(st.internTable) > preTable*2 {
		t.Fatalf("internTable cap grew unexpectedly: pre=%d post=%d", preTable, cap(st.internTable))
	}
}

// Decoder values slice — retain-then-release, driven by len(d.values).
func TestShrink_DecStateReleasesValuesAfterStreak(t *testing.T) {
	d := newDecState()
	for i := range maxRetainedIDs + 64 {
		d.append(fmt.Appendf(nil, "dec-val-%05d", i))
	}
	if cap(d.values) <= maxRetainedIDs {
		t.Fatalf("test premise: values cap should exceed %d, got %d",
			maxRetainedIDs, cap(d.values))
	}
	burstCap := cap(d.values)

	d.reset()
	if cap(d.values) != burstCap {
		t.Fatalf("values dropped on first post-burst reset (should retain): burst=%d post=%d",
			burstCap, cap(d.values))
	}
	for range retainReleaseStreak {
		d.reset()
	}
	if d.values != nil {
		t.Fatalf("decoder values not released after %d small resets: cap=%d", retainReleaseStreak, cap(d.values))
	}
	if d.lruLink != nil {
		t.Fatalf("decoder lruLink not released after streak")
	}
}

// Integration view: a SAME encState through burst-grow → retain → quiet →
// release. Memory is held across the first quiet reset (amortization) and
// returned only after the streak elapses.
func TestShrink_BurstThenSteadyState(t *testing.T) {
	st := newEncState()
	for i := range maxRetainedIDs + 64 {
		st.lookupOrAssign(fmt.Sprintf("burst-%05d-%s", i, strings.Repeat("z", 16)))
	}
	st.pairRecord(uint32(int(st.internLoad)-1), 0)

	burstPairCap := cap(st.pairPred)
	burstArena := st.arena.BytesUsed()
	t.Logf("burst peak: ids=%d pairCap=%d arenaUsed=%d KiB",
		int(st.internLoad), burstPairCap, burstArena/1024)

	// First post-burst reset retains the grown pair backing.
	st.reset()
	if burstPairCap > maxRetainedPairCap && cap(st.pairPred) != burstPairCap {
		t.Fatalf("pairPred dropped on first post-burst reset: burst=%d post=%d", burstPairCap, cap(st.pairPred))
	}

	// Quiet phase: small messages for the full streak → release.
	for range retainReleaseStreak {
		for i := range 100 {
			st.lookupOrAssign(fmt.Sprintf("steady-%05d", i))
		}
		st.reset()
	}

	if cap(st.pairPred) >= burstPairCap && burstPairCap > maxRetainedPairCap {
		t.Fatalf("pairPred cap did not shrink: burst=%d post=%d",
			burstPairCap, cap(st.pairPred))
	}
	if used := st.arena.BytesUsed(); burstArena > internarenaRetainBytes() && used > internarenaRetainBytes() {
		t.Fatalf("arena did not shrink past its watermark: burst=%dB post=%dB", burstArena, used)
	}
}

// No-regression guard: a SMALL steady workload (internLoad always under the
// soft cap) must behave exactly as before the adaptive-retention change — the
// table is reused in place across resets, never reallocated and never released
// to the init size. The retain/release branches added by the change only fire
// for over-cap arrays, so for small messages the policy is inert.
func TestShrink_SmallSteadyIsInert(t *testing.T) {
	st := newEncState()
	const perMsg = 200 // well under maxRetainedIDs
	fill := func(cycle int) {
		for i := range perMsg {
			st.lookupOrAssign(fmt.Sprintf("c%02d-small-%04d", cycle, i))
		}
	}
	// Warm up to the steady-state table size.
	fill(0)
	st.reset()
	fill(1)
	warmCap := cap(st.internTable)
	if warmCap > maxRetainedIDs*2 {
		t.Fatalf("test premise: small workload should stay under cap, got internTable cap=%d", warmCap)
	}

	// Many cycles: cap must stay put — no realloc churn, no drop-to-init,
	// no streak-driven release (the arrays never exceeded the cap).
	for c := range 30 {
		st.reset()
		fill(c + 2)
		if got := cap(st.internTable); got != warmCap {
			t.Fatalf("cycle %d: internTable cap changed (realloc/drop churn): warm=%d got=%d", c, warmCap, got)
		}
	}
	if st.internTable == nil {
		t.Fatal("small steady workload wrongly released under-cap backings")
	}
}

// Helper exported only for this test — internarena.DefaultRetainBytes
// is the public const; alias here for readability inside the package.
func internarenaRetainBytes() int { return internarenaDefaultRetainBytes }

// Keep this in sync with internarena.DefaultRetainBytes; copied here
// so the shrink test can compare without re-exporting the const.
const internarenaDefaultRetainBytes = 256 * 1024

// TestColScratchStrClearedOnReset pins that the columnar string scratch is
// cleared across its FULL backing on a (retained) pool reset, not just up to
// len. colScratchStr is resliced via [:n] / [:0], so a column with fewer rows
// than an earlier one leaves a high-water tail of live string headers; without
// a full clear those pin the prior message's strings (or the caller's struct
// strings on the encoder) from GC for the pooled state's lifetime.
func TestColScratchStrClearedOnReset(t *testing.T) {
	live := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	t.Run("decoder", func(t *testing.T) {
		d := &decState{}
		d.colScratchStr = live[:2:5] // len 2, cap 5: tail [2:5] holds live headers
		d.reset()
		for i, s := range d.colScratchStr[:cap(d.colScratchStr)] {
			if s != "" {
				t.Fatalf("decState retained string header at [%d]=%q after reset", i, s)
			}
		}
	})

	t.Run("encoder", func(t *testing.T) {
		e := &encState{}
		e.colScratchStr = live[:2:5]
		e.reset()
		for i, s := range e.colScratchStr[:cap(e.colScratchStr)] {
			if s != "" {
				t.Fatalf("encState retained string header at [%d]=%q after reset", i, s)
			}
		}
	})
}
