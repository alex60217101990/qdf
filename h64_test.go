package qdf

import (
	"fmt"
	"testing"
)

// TestH64Distribution guards the intern-table hash against the
// catastrophic collision mode that a pure prefix+suffix sampler has:
// high-cardinality keys that share a fixed leading and trailing 8 bytes
// but differ only in the middle (URLs, paths, formatted IDs) must not
// all collapse to one bucket. internHash routes keys longer than
// internHashFastMax to maphash (which hashes every byte) precisely to
// prevent that; this test fails if that length gate is removed or h64 is
// used directly on long keys.
func TestH64Distribution(t *testing.T) {
	dupRate := func(keys []string) float64 {
		seen := make(map[uint64]int, len(keys))
		for _, k := range keys {
			seen[internHash(k)]++
		}
		dup := 0
		for _, c := range seen {
			if c > 1 {
				dup += c - 1
			}
		}
		return float64(dup) / float64(len(keys))
	}

	const n = 5000
	corpora := map[string][]string{}

	// Worst case: identical 8B prefix ("trace-00") + identical 8B suffix
	// ("suffixZZ"), differing only in the middle.
	mid := make([]string, n)
	for i := range mid {
		mid[i] = fmt.Sprintf("trace-00%08d-suffixZZ", i)
	}
	corpora["middle-only"] = mid

	// URL-shaped: shared scheme/host prefix and shared trailing segment.
	urls := make([]string, n)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://api.example.com/v1/users/%d/profile", i)
	}
	corpora["url-shaped"] = urls

	// Short labels with varied tails (the common intern case).
	short := make([]string, n)
	for i := range short {
		short[i] = fmt.Sprintf("node-%05d", i)
	}
	corpora["short-tail"] = short

	for name, keys := range corpora {
		if r := dupRate(keys); r > 0.01 {
			t.Errorf("%s: collision rate %.4f exceeds 1%% — h64 distribution degraded", name, r)
		}
	}
}
