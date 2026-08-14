package qdf

import "testing"

// canonKeysBusy is a re-entrancy flag, and the reflect key gathers release it
// explicitly rather than with defer. A panic through a StreamEncoder — which is
// not discarded the way a pooled encoder is — would leave it set, and every
// later canonical map encode would allocate a fresh key slice forever.
//
// Reset is the backstop, so assert Reset clears it.
func TestResetClearsCanonKeysBusy(t *testing.T) {
	e := NewEncoderWith(OptBalanced | OptCanonical)
	e.EnsureHeader()
	if e.state == nil {
		e.state = newEncState()
	}
	e.state.canonKeysBusy = true
	e.state.reset()
	if e.state.canonKeysBusy {
		t.Fatal("reset left canonKeysBusy set — a stream that panicked mid-gather " +
			"would allocate a fresh key slice on every later canonical map encode")
	}
}

// strDeltaBase is indexed by WIRE shape id, so the payload chooses its length.
// Without a ceiling, one message declaring many shapes pins a []decFieldState
// per shape on the pooled decoder until the pool is collected.
func TestResetBoundsStrDeltaBase(t *testing.T) {
	d := &decState{}
	d.strDeltaBase = make([][]decFieldState, maxRetainedShapeCap+1)
	// The release branch is what drops retained capacity; reach it the same way
	// the pool does, by declaring a run of small messages.
	d.retainStreak = retainReleaseStreak
	d.reset()
	if len(d.strDeltaBase) > maxRetainedShapeCap {
		t.Fatalf("reset kept %d shape rows (ceiling %d) — a single wide message "+
			"pins them for the life of the pooled decoder",
			len(d.strDeltaBase), maxRetainedShapeCap)
	}
}

// And the ceiling must NOT fire for an ordinary table, or a steady workload
// reallocates its per-shape rows on every message.
func TestResetKeepsASmallStrDeltaBase(t *testing.T) {
	d := &decState{}
	d.strDeltaBase = make([][]decFieldState, 4)
	d.strDeltaBase[0] = make([]decFieldState, 3)
	d.retainStreak = retainReleaseStreak
	d.reset()
	if len(d.strDeltaBase) != 4 {
		t.Fatalf("reset dropped a %d-row table that is well under the %d ceiling",
			len(d.strDeltaBase), maxRetainedShapeCap)
	}
}
