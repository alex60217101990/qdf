//go:build race

package qdf

// See race_norace.go for the contract; this file flips raceEnabled
// on when the race detector is linked.
const raceEnabled = true
