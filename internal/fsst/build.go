package fsst

import "sort"

const (
	buildRounds  = 5    // refinement iterations
	maxSampleStr = 8192 // cap strings scanned per round (bounded encode CPU)
)

// BuildSymbolTable learns a SymbolTable from samples. Deterministic: the same
// samples always produce the same table (stable candidate ordering, fixed
// stride sample, no RNG). The wire stores the table, so table quality affects
// compression ratio only — never correctness.
func BuildSymbolTable(samples [][]byte) *SymbolTable {
	scan := sampleStride(samples, maxSampleStr)
	t := newSymbolTable(nil) // empty: everything is an escape on round 0
	for round := 0; round < buildRounds; round++ {
		counts := make(map[string]int, 1024)
		for _, s := range scan {
			tokenizeCount(t, s, counts)
		}
		t = buildFromCounts(counts)
	}
	return t
}

// sampleStride returns at most maxN strings chosen at a fixed stride across
// samples (deterministic, covers the whole corpus rather than just the head).
func sampleStride(samples [][]byte, maxN int) [][]byte {
	if len(samples) <= maxN {
		return samples
	}
	out := make([][]byte, 0, maxN)
	step := len(samples) / maxN
	if step < 1 {
		step = 1
	}
	for i := 0; i < len(samples) && len(out) < maxN; i += step {
		out = append(out, samples[i])
	}
	return out
}

// tokenizeCount greedily tokenizes s with the current table and counts each
// emitted symbol and each adjacent-symbol pair (concatenated, ≤8 bytes).
func tokenizeCount(t *SymbolTable, s []byte, counts map[string]int) {
	i := 0
	var prev []byte
	for i < len(s) {
		_, n := t.match(s[i:])
		var cur []byte
		if n == 0 {
			cur = s[i : i+1]
			i++
		} else {
			cur = s[i : i+n]
			i += n
		}
		counts[string(cur)]++
		if prev != nil && len(prev)+len(cur) <= maxSymLen {
			counts[string(prev)+string(cur)]++
		}
		prev = cur
	}
}

// buildFromCounts selects the top-255 candidates by gain = freq*len, with a
// deterministic total order (gain desc, len desc, bytes asc).
func buildFromCounts(counts map[string]int) *SymbolTable {
	type cand struct {
		b    string
		gain int
	}
	cs := make([]cand, 0, len(counts))
	for b, c := range counts {
		if len(b) == 0 || len(b) > maxSymLen {
			continue
		}
		cs = append(cs, cand{b, c * len(b)})
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].gain != cs[j].gain {
			return cs[i].gain > cs[j].gain
		}
		if len(cs[i].b) != len(cs[j].b) {
			return len(cs[i].b) > len(cs[j].b)
		}
		return cs[i].b < cs[j].b
	})
	raw := make([][]byte, 0, maxSymbols)
	for _, c := range cs {
		if len(raw) >= maxSymbols {
			break
		}
		raw = append(raw, []byte(c.b))
	}
	return newSymbolTable(raw)
}
