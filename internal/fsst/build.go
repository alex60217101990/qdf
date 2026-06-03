package fsst

import "sort"

const (
	buildRounds = 3 // refinement iterations

	// maxSampleBytes caps the bytes of input scanned per training round. FSST
	// table quality saturates well before a whole large column is consumed, so
	// bounding the sample keeps encode cost flat regardless of column size.
	maxSampleBytes = 1 << 13 // 8 KiB

	// counterLog sizes the flat candidate counter (1<<counterLog slots). Large
	// enough to hold the distinct symbols + adjacent pairs of an 8 KiB sample
	// at a < 0.75 load factor; once full it stops admitting new keys (the
	// frequent ones are already present), bounding memory and time.
	counterLog = 13
)

// symKey is a fixed-size, comparable key for a candidate symbol of up to 8
// bytes — counting it never materializes a string.
type symKey struct {
	lo uint64 // up to 8 bytes, packed little-endian
	n  uint8  // length 1..8
}

func packKey(b []byte) symKey {
	var lo uint64
	for i := 0; i < len(b); i++ {
		lo |= uint64(b[i]) << (8 * i)
	}
	return symKey{lo, uint8(len(b))}
}

// counter is a flat open-addressed frequency table for symKeys. It replaces a
// map[symKey]int: no bucket chasing, an integer hash, and a hard capacity that
// drops new keys once full (frequent candidates are already counted). Reused
// across rounds via reset.
type counter struct {
	keys []symKey
	cnt  []int32
	used int
	mask uint32
}

func newCounter() *counter {
	n := 1 << counterLog
	return &counter{keys: make([]symKey, n), cnt: make([]int32, n), mask: uint32(n - 1)}
}

func (c *counter) reset() {
	clear(c.cnt) // cnt==0 marks a slot empty; stale keys are ignored
	c.used = 0
}

// add increments the count for k, inserting it if absent and capacity remains.
func (c *counter) add(k symKey) {
	h := uint32(k.lo) ^ uint32(k.lo>>32)
	h = h*2654435761 + uint32(k.n)*40503
	i := h & c.mask
	for {
		if c.cnt[i] == 0 {
			if c.used<<2 >= len(c.cnt)*3 { // load factor 0.75: stop admitting
				return
			}
			c.keys[i] = k
			c.cnt[i] = 1
			c.used++
			return
		}
		if c.keys[i] == k {
			c.cnt[i]++
			return
		}
		i = (i + 1) & c.mask
	}
}

// BuildSymbolTable learns a SymbolTable from samples. Deterministic: identical
// samples always yield an identical table (no RNG; total-order selection). The
// wire stores the table, so table quality affects ratio only, never correctness.
func BuildSymbolTable(samples [][]byte) *SymbolTable {
	scan := sampleByBytes(samples, maxSampleBytes)
	t := newSymbolTable(nil) // empty: round 0 is all single-byte tokens
	c := newCounter()
	cand := make([]candidate, 0, 2048) // reused across rounds
	keys := make([]symKey, 0, maxSymbols)
	for round := 0; round < buildRounds; round++ {
		c.reset()
		empty := len(t.symbols) == 0
		for _, s := range scan {
			tokenizeCount(t, s, c, empty)
		}
		keys = topCandidates(c, cand[:0], keys[:0])
		t = newSymbolTableFromKeys(keys)
	}
	return t
}

type candidate struct {
	k    symKey
	gain int
}

// newSymbolTableFromKeys builds a table directly from packed candidate keys,
// writing each symbol's bytes straight into its fixed array — no intermediate
// [][]byte and no per-symbol allocation.
func newSymbolTableFromKeys(keys []symKey) *SymbolTable {
	t := &SymbolTable{}
	for _, k := range keys {
		if k.n == 0 || k.n > maxSymLen || len(t.symbols) >= maxSymbols {
			continue
		}
		var s symbol
		s.len = k.n
		for i := 0; i < int(k.n); i++ {
			s.bytes[i] = byte(k.lo >> (8 * i))
		}
		s.val = k.lo // packKey already packed exactly k.n bytes
		if k.n >= maxSymLen {
			s.mask = ^uint64(0)
		} else {
			s.mask = (uint64(1) << (8 * k.n)) - 1
		}
		code := uint8(len(t.symbols))
		t.symbols = append(t.symbols, s)
		t.byFirst[s.bytes[0]] = append(t.byFirst[s.bytes[0]], code)
	}
	t.buildIndex()
	return t
}

// sampleByBytes returns the leading prefix of samples whose total length first
// reaches budget (deterministic, bounded).
func sampleByBytes(samples [][]byte, budget int) [][]byte {
	total := 0
	for i := range samples {
		total += len(samples[i])
		if total >= budget {
			return samples[:i+1]
		}
	}
	return samples
}

// tokenizeCount greedily tokenizes s with the current table and counts each
// emitted symbol and each adjacent-symbol pair. The pair is a contiguous slice
// of s (the previous token immediately precedes the current one), so counting
// never concatenates or allocates. When empty is set (round 0, no symbols yet)
// every token is a single byte, so the per-position match scan is skipped.
func tokenizeCount(t *SymbolTable, s []byte, c *counter, empty bool) {
	i := 0
	prevStart, prevLen := 0, 0
	for i < len(s) {
		n := 1
		if !empty {
			if _, m := t.match(s[i:]); m != 0 {
				n = m
			}
		}
		c.add(packKey(s[i : i+n]))
		if prevLen != 0 && prevLen+n <= maxSymLen {
			c.add(packKey(s[prevStart : i+n])) // prev+cur, contiguous
		}
		prevStart, prevLen = i, n
		i += n
	}
}

// topCandidates selects the top-255 candidate keys by gain = freq*len, with a
// deterministic total order (gain desc, len desc, packed-bytes asc). The cand
// and keys scratch slices are caller-owned and reused across rounds.
func topCandidates(c *counter, cand []candidate, keys []symKey) []symKey {
	for i := range c.cnt {
		if c.cnt[i] == 0 {
			continue
		}
		k := c.keys[i]
		if k.n == 0 || k.n > maxSymLen {
			continue
		}
		cand = append(cand, candidate{k, int(c.cnt[i]) * int(k.n)})
	}
	sort.Slice(cand, func(i, j int) bool {
		if cand[i].gain != cand[j].gain {
			return cand[i].gain > cand[j].gain
		}
		if cand[i].k.n != cand[j].k.n {
			return cand[i].k.n > cand[j].k.n
		}
		return cand[i].k.lo < cand[j].k.lo
	})
	for i := range cand {
		if len(keys) >= maxSymbols {
			break
		}
		keys = append(keys, cand[i].k)
	}
	return keys
}
