package fsst

import "slices"

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
	for i := range b {
		lo |= uint64(b[i]) << (8 * i)
	}
	return symKey{lo, uint8(len(b))}
}

// counter is a flat open-addressed frequency table for symKeys. It replaces a
// map[symKey]int: no bucket chasing, an integer hash, and a hard capacity that
// drops new keys once full (frequent candidates are already counted). Reused
// across rounds via reset.
type counter struct {
	keys    []symKey
	cnt     []int32
	usedIdx []uint32 // occupied slot indices, so reset/iterate are O(distinct) not O(table)
	mask    uint32
}

func newCounter() *counter {
	n := 1 << counterLog
	return &counter{
		keys:    make([]symKey, n),
		cnt:     make([]int32, n),
		usedIdx: make([]uint32, 0, 2048),
		mask:    uint32(n - 1),
	}
}

func (c *counter) reset() {
	// Clear only the slots we touched — the table is 8192 wide but a single
	// training round fills far fewer, so a full clear(c.cnt) dominated the
	// per-round cost on small (probe-sample) inputs.
	for _, i := range c.usedIdx {
		c.cnt[i] = 0
	}
	c.usedIdx = c.usedIdx[:0]
}

// add increments the count for k, inserting it if absent and capacity remains.
func (c *counter) add(k symKey) {
	h := uint32(k.lo) ^ uint32(k.lo>>32)
	h = h*2654435761 + uint32(k.n)*40503
	i := h & c.mask
	for {
		if c.cnt[i] == 0 {
			if len(c.usedIdx)<<2 >= len(c.cnt)*3 { // load factor 0.75: stop admitting
				return
			}
			c.keys[i] = k
			c.cnt[i] = 1
			c.usedIdx = append(c.usedIdx, i)
			return
		}
		if c.keys[i] == k {
			if c.cnt[i] < 0x7fffffff { // saturate: 0 means empty, never wrap to it
				c.cnt[i]++
			}
			return
		}
		i = (i + 1) & c.mask
	}
}

type candidate struct {
	k    symKey
	gain int
}

// Builder holds reusable training scratch — the counter, candidate/keys buffers,
// and the table itself (whose symbol slice and first-byte index buckets retain
// their capacity across calls). The columnar probe estimates FSST per string
// column and the encoder re-trains per column, so without reuse each call
// re-allocated the counter and ~255 first-byte bucket slices (the dominant FSST
// encode allocation). Not safe for concurrent use; pool one per goroutine.
type Builder struct {
	c    *counter
	cand []candidate
	keys []symKey
	t    SymbolTable
}

func NewBuilder() *Builder {
	return &Builder{
		c:    newCounter(),
		cand: make([]candidate, 0, 2048),
		keys: make([]symKey, 0, maxSymbols),
	}
}

// Build trains a table from samples, reusing the Builder's scratch. The returned
// table ALIASES the Builder's internal table and is valid only until the next
// Build on the same Builder — a caller that retains the table (a pre-trained
// dictionary) must use BuildSymbolTable, which returns a fresh table.
//
// Deterministic: identical samples always yield an identical table.
func (b *Builder) Build(samples [][]byte) *SymbolTable {
	return b.BuildRounds(samples, buildRounds)
}

// BuildRounds is Build with an explicit refinement-round count. The columnar
// probe uses fewer rounds for its size ESTIMATE (a rough table is enough to
// decide columnar-vs-row-major), while the encoder uses the full buildRounds
// for the table it actually emits.
func (b *Builder) BuildRounds(samples [][]byte, rounds int) *SymbolTable {
	scan := sampleByBytes(samples, maxSampleBytes)
	b.t.reset() // empty: round 0 is all single-byte tokens
	for range rounds {
		b.c.reset()
		empty := len(b.t.symbols) == 0
		for _, s := range scan {
			tokenizeCount(&b.t, s, b.c, empty)
		}
		b.keys = topCandidates(b.c, b.cand[:0], b.keys[:0])
		b.t.fillFromKeys(b.keys)
	}
	return &b.t
}

// BuildSymbolTable trains a fresh, independent table (for a retained dictionary
// or one-shot use). The wire stores the table, so table quality affects ratio
// only, never correctness.
func BuildSymbolTable(samples [][]byte) *SymbolTable {
	return NewBuilder().Build(samples)
}

// reset clears the table for reuse, keeping the symbol slice and first-byte
// bucket capacities so a subsequent fillFromKeys does not re-allocate.
func (t *SymbolTable) reset() {
	t.symbols = t.symbols[:0]
	for i := range t.byFirst {
		if t.byFirst[i] != nil {
			t.byFirst[i] = t.byFirst[i][:0]
		}
	}
}

// fillFromKeys rebuilds the table in place from packed candidate keys — each
// symbol's bytes written straight into its fixed array, no intermediate [][]byte.
func (t *SymbolTable) fillFromKeys(keys []symKey) {
	t.reset()
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
}

// sampleByBytes returns the leading prefix of samples whose total length first
// reaches budget, truncating the final element so the SCANNED bytes are bounded
// by budget regardless of any single element's size (a lone multi-MB string
// must not make training O(that string)). Deterministic; never mutates the
// caller's backing data (it shallow-copies the small header slice only when it
// has to trim).
func sampleByBytes(samples [][]byte, budget int) [][]byte {
	total := 0
	for i := range samples {
		total += len(samples[i])
		if total >= budget {
			over := total - budget
			if over > 0 && over < len(samples[i]) {
				trimmed := make([][]byte, i+1)
				copy(trimmed, samples[:i+1])
				trimmed[i] = samples[i][:len(samples[i])-over]
				return trimmed
			}
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
	for _, i := range c.usedIdx { // only occupied slots, not the whole 8192-wide table
		k := c.keys[i]
		if k.n == 0 || k.n > maxSymLen {
			continue
		}
		cand = append(cand, candidate{k, int(c.cnt[i]) * int(k.n)})
	}
	// High-cardinality columns produce thousands of candidates of which only 255
	// are kept; quickselect the top 255 in O(n) and sort just those, instead of
	// fully sorting the whole set (the dominant FSST training cost).
	if len(cand) > maxSymbols {
		selectTopK(cand, maxSymbols)
		cand = cand[:maxSymbols]
	}
	// slices.SortFunc (typed) avoids sort.Slice's reflection-based swapper.
	slices.SortFunc(cand, func(a, b candidate) int {
		if candLess(a, b) {
			return -1
		}
		if candLess(b, a) {
			return 1
		}
		return 0
	})
	for i := range cand {
		keys = append(keys, cand[i].k)
	}
	return keys
}

// candLess reports whether a outranks b: higher gain, then longer symbol, then
// smaller packed bytes. A strict total order on distinct keys (no ties).
func candLess(a, b candidate) bool {
	if a.gain != b.gain {
		return a.gain > b.gain
	}
	if a.k.n != b.k.n {
		return a.k.n > b.k.n
	}
	return a.k.lo < b.k.lo
}

// selectTopK partitions cand (Hoare quickselect, O(n) average) so the k
// highest-ranked candidates by candLess occupy cand[:k] (unordered). Used to
// avoid fully sorting a large candidate set when only the top maxSymbols are
// kept. The selected SET is deterministic for a given input.
func selectTopK(cand []candidate, k int) {
	lo, hi := 0, len(cand)-1
search:
	for lo < hi {
		pivot := cand[(lo+hi)/2]
		i, j := lo, hi
		for i <= j {
			for candLess(cand[i], pivot) {
				i++
			}
			for candLess(pivot, cand[j]) {
				j--
			}
			if i <= j {
				cand[i], cand[j] = cand[j], cand[i]
				i++
				j--
			}
		}
		switch {
		case k <= j:
			hi = j
		case k >= i:
			lo = i
		default:
			break search
		}
	}
}
