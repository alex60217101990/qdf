//go:build !race

package qdf

// raceEnabled is the compile-time runtime check used by tests that
// rely on testing.AllocsPerRun. The race detector evicts sync.Pool
// per-P caches more aggressively than the non-race runtime, so a
// pooled encoder that reuses its buffer at 0 allocations in
// production reports a small constant overhead under -race. Tests
// that gate on a tight alloc budget skip themselves when raceEnabled
// is true rather than masking the assertion behind a build tag.
const raceEnabled = false
