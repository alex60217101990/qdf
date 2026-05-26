package qdf

import (
	"testing"
)

// Allocation-budget tests. testing.AllocsPerRun measures the average
// number of heap allocations across many calls. Hot paths sit under a
// strict budget; a PR that pushes them over the budget fails CI
// instead of slipping into a release.
//
// The budgets are an upper bound, not a target. They are intentionally
// a little loose so a one-off compiler tweak does not flake the suite.
// Tighten them if a real improvement lands.

type allocSmall struct {
	ID   int    `qdf:"id"`
	Name string `qdf:"name"`
	Age  int    `qdf:"age"`
}

func assertAllocs(t *testing.T, name string, budget float64, fn func()) {
	t.Helper()
	if raceEnabled {
		// sync.Pool per-P caches drain more aggressively under
		// -race, so tight pool-reuse budgets get a 1–2 alloc bump
		// that is not present in production. Skip with a clear
		// message; CI runs the suite under both modes and the
		// non-race pass enforces the real budget.
		t.Skipf("alloc budgets are not measured under -race (sync.Pool churn instrumentation)")
	}
	got := testing.AllocsPerRun(200, fn)
	if got > budget {
		t.Fatalf("%s: %.1f allocs/op, budget %.1f", name, got, budget)
	}
	t.Logf("%s: %.1f allocs/op (budget %.1f)", name, got, budget)
}

func TestAllocBudget_MarshalSpeed(t *testing.T) {
	v := allocSmall{ID: 42, Name: "alice", Age: 30}
	assertAllocs(t, "Marshal(struct, OptSpeed)", 4, func() {
		_, _ = Marshal(v, OptSpeed)
	})
}

func TestAllocBudget_MarshalTSpeed(t *testing.T) {
	v := allocSmall{ID: 42, Name: "alice", Age: 30}
	assertAllocs(t, "MarshalT[struct](OptSpeed)", 3, func() {
		_, _ = MarshalT(v, OptSpeed)
	})
}

func TestAllocBudget_MarshalBalanced(t *testing.T) {
	v := allocSmall{ID: 42, Name: "alice", Age: 30}
	// Dense pays one extra allocation for the intern-table map ops on
	// top of the Speed baseline.
	assertAllocs(t, "Marshal(struct, OptBalanced)", 7, func() {
		_, _ = Marshal(v, OptBalanced)
	})
}

func TestAllocBudget_MarshalQPack(t *testing.T) {
	type vec struct {
		IDs []uint64 `qdf:"ids"`
	}
	v := vec{IDs: []uint64{1, 2, 3, 4, 5, 6, 7, 8}}
	assertAllocs(t, "Marshal(numeric slice, OptQPack)", 4, func() {
		_, _ = Marshal(v, OptQPack)
	})
}

func TestAllocBudget_Unmarshal(t *testing.T) {
	v := allocSmall{ID: 42, Name: "alice", Age: 30}
	buf, _ := Marshal(v, OptSpeed)
	assertAllocs(t, "Unmarshal(struct)", 4, func() {
		var out allocSmall
		_ = Unmarshal(buf, &out)
	})
}

func TestAllocBudget_AppendMarshalPooledBuffer(t *testing.T) {
	// Caller-owned buffer that is large enough — encoder must not
	// reallocate; pool returns 0 extra allocations once warm.
	v := allocSmall{ID: 42, Name: "alice", Age: 30}
	dst := make([]byte, 0, 256)
	// Warm-up.
	for range 4 {
		dst, _ = AppendMarshal(dst[:0], v, OptSpeed)
	}
	assertAllocs(t, "AppendMarshal(warm dst, OptSpeed)", 2, func() {
		dst, _ = AppendMarshal(dst[:0], v, OptSpeed)
	})
	_ = dst
}

func TestAllocBudget_StringInternBalanced(t *testing.T) {
	// Repeated string emissions in Dense mode should NOT allocate per
	// reference — the second occurrence resolves through the intern
	// table without copying the payload.
	v := struct {
		Vals []string `qdf:"vals"`
	}{Vals: []string{"region-eu-west-1", "region-eu-west-1", "region-eu-west-1"}}
	assertAllocs(t, "Marshal(repeated strings, OptBalanced)", 7, func() {
		_, _ = Marshal(v, OptBalanced)
	})
}

func TestAllocBudget_BoolSliceBitpack(t *testing.T) {
	// Bitpacked []bool via OptQPack must keep the cost at the per-call
	// overhead — no per-bit allocation.
	v := struct {
		B []bool `qdf:"b"`
	}{B: make([]bool, 1024)}
	for i := range v.B {
		v.B[i] = i%3 == 0
	}
	assertAllocs(t, "Marshal([]bool 1024, OptQPack)", 4, func() {
		_, _ = Marshal(v, OptQPack)
	})
}
