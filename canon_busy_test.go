package qdf

import "testing"

// A canonical (OptCanonical) map encode that errors mid-loop on an `any` value
// must still release the canonKeysBusy re-entrancy latch — otherwise a pooled
// encoder is pinned to the fresh-allocation fallback forever. The generated
// fast paths now release via defer (matching the reflect path). Regression for
// the maps_fast_generated canonical-encode error path.
func TestCanonKeysBusyReleasedOnError(t *testing.T) {
	e := NewEncoderWith(OptCanonical | OptDense)
	// chan has no codec → encodeReflect errors inside the sorted-key loop.
	if err := e.EncodeValue(map[string]any{"k": make(chan int)}); err == nil {
		t.Fatal("expected an error encoding a chan map value")
	}
	if e.state != nil && e.state.canonKeysBusy {
		t.Fatal("canonKeysBusy leaked true after a mid-loop error return")
	}
	// A subsequent canonical map encode must still succeed (and reuse the pooled
	// scratch, not be wedged on the busy latch).
	if err := e.EncodeValue(map[string]any{"b": 2, "a": 1}); err != nil {
		t.Fatalf("follow-up canonical encode failed: %v", err)
	}
}
