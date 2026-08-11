package qdf

import "testing"

// The firing counters are gated in production, so every test that asserts on
// them has to turn them on. TestMain would be the obvious home, but this package
// already has one; an init in a _test.go file is compiled only into the test
// binary and reaches every test without one.
func init() { strDeltaCount = true }

// A guard that is never enabled makes every counter assertion vacuous, which is
// the exact failure mode the counters exist to prevent.
func TestStrDeltaCountersAreEnabledInTests(t *testing.T) {
	if !strDeltaCount {
		t.Fatal("the firing counters are off in the test binary — every assertion on them is vacuous")
	}
}
