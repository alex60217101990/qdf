//go:build race

package tans

// raceEnabled gates compute-only exhaustive tests: they contain no
// concurrency, so the race detector adds cost (5-10x) but no coverage,
// and the macos-latest CI runner blows the 10m go-test timeout.
const raceEnabled = true
