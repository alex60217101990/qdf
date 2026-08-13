package qdf

import (
	"fmt"
	"testing"
)

// The discriminator's only hard requirement: equal strings MUST agree. A
// disagreement would inflate the distinct count and change a codec decision.
// (The reverse is not required — a collision costs one full compare, which the
// scan still performs.)
func TestStrDiscriminatorAgreesOnEqualStrings(t *testing.T) {
	for _, s := range []string{
		"", "a", "ab", "abcdefg", "abcdefgh", "abcdefghi",
		"com.acme.0000.worker.service.000",
		"D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)",
	} {
		if a, b := strDiscriminator(s), strDiscriminator(string([]byte(s))); a != b {
			t.Errorf("%q: two equal strings keyed %d and %d", s, a, b)
		}
	}
}

// It must actually separate the shape it exists for: values sharing a long
// prefix and differing only near the end. A head-only key would collide on all
// of these, which is the case that made the old scan walk the whole prefix.
func TestStrDiscriminatorSeparatesSharedPrefixes(t *testing.T) {
	const stem = "com.acme.platform.worker.service."
	seen := make(map[uint64]string, 512)
	for i := range 512 {
		s := stem + fmt.Sprintf("%06d", i)
		k := strDiscriminator(s)
		if prev, dup := seen[k]; dup {
			t.Fatalf("prefix-sharing values collided: %q and %q both key %d", prev, s, k)
		}
		seen[k] = s
	}
}

// The distinct count itself must be unchanged. Asserted against a plain
// reference implementation over inputs built to stress equality: repeats,
// shared prefixes, shared suffixes, and equal lengths.
func TestDictSampleHighCardMatchesAReference(t *testing.T) {
	ref := func(strs []string) bool {
		n := min(len(strs), qpackStrDictSampleN)
		var seen []string
		for i := range n {
			fresh := true
			for _, p := range seen {
				if p == strs[i] {
					fresh = false
					break
				}
			}
			if fresh {
				seen = append(seen, strs[i])
			}
		}
		return len(seen)*100 > n*qpackStrDictSampleMaxPct
	}
	cases := [][]string{
		{},
		{"a"},
		{"a", "a", "a"},
		{"a", "b", "c"},
	}
	// Shared prefix, differing tail — the codec's own territory.
	var pre []string
	for i := range 100 {
		pre = append(pre, "com.acme.platform.worker."+fmt.Sprintf("%06d", i%7))
	}
	cases = append(cases, pre)
	// Shared SUFFIX, differing head — the mirror case, which a tail-only key
	// would collide on if the head were not folded in.
	var suf []string
	for i := range 100 {
		suf = append(suf, fmt.Sprintf("%06d", i%7)+".worker.platform.acme.com")
	}
	cases = append(cases, suf)
	// Equal length, all distinct.
	var eq []string
	for i := range 100 {
		eq = append(eq, fmt.Sprintf("%032x", uint64(i)*11400714819323198485))
	}
	cases = append(cases, eq)
	for i, c := range cases {
		if got, want := dictSampleHighCard(c), ref(c); got != want {
			t.Errorf("case %d (%d values): got %v want %v", i, len(c), got, want)
		}
	}
}
