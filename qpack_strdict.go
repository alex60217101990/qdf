package qdf

import (
	"cmp"
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Dictionary-coded string columns for the columnar container. A string
// column whose values are drawn from a small set (enum-like dimensions —
// log level, service, region, status) is cheaper to store as a distinct
// table plus a bitpacked index per row than as M interned values, because
// each interned reference still costs at least a byte while an index costs
// ceil(log2 distinct) bits. Wire format documented at tagColStrDict.
//
// The codec is gated so it never grows the wire: it is chosen only when the
// bitpacked index body is smaller than the per-value floor of n bytes (the
// gathered-column fallback emits at least a 1-byte string/intern-ref tag per
// row and does NOT run-length-collapse consecutive identical rows). Because the
// distinct count is capped at 256, the index spends at most 8 bits/row, so a
// bitpacked index beats per-value refs for every column the high-cardinality
// probe does not already reject — including skewed (clustered) low-card columns,
// which an earlier run-count floor wrongly sent to the per-value path.

const (
	// qpackStrDictMaxDistinct caps the distinct count. 256 keeps the index
	// at <= 8 bits and the distinct probe bounded.
	qpackStrDictMaxDistinct = 256
	// qpackStrDictMinElems skips the probe for tiny columns where the table
	// overhead cannot pay off.
	qpackStrDictMinElems = 16
	// Cheap high-cardinality bail: if the first sample rows are mostly
	// distinct, the column is ID/text-like — abandon before the full scan.
	qpackStrDictSampleN      = 64
	qpackStrDictSampleMaxPct = 70 // distinct% over the sample above which we bail
)

// tryWriteStringColumnDict attempts to emit strs as a tagColStrDict block.
// It returns true when the dictionary form was written (and is strictly
// smaller than the per-value n-byte floor), false when the caller should fall
// back to per-value WriteString. Encode-side scratch is reused across
// columns to avoid per-column allocation.
func (e *Encoder) tryWriteStringColumnDict(strs []string) bool {
	n := len(strs)
	if n < qpackStrDictMinElems {
		return false
	}
	m := e.state.strDictMap
	if m == nil {
		m = make(map[string]uint32, qpackStrDictMaxDistinct)
		e.state.strDictMap = m
	} else {
		clear(m)
	}
	table := e.state.colDictTable.get()[:0]
	idx := e.state.colScratchU64.get()[:0]
	// gp tracks the longest byte prefix shared by EVERY distinct value, computed
	// for free as the table is built (no sort, no extra pass). It is a sort-free
	// signal for the front-coded table form: gp >= 2 guarantees the front-coded
	// table is strictly smaller than the plain one (it saves >= (d-1)*(gp-1)
	// bytes), so the encoder pays the sort ONLY when front-coding is sure to win
	// — a non-prefix-shared column skips it entirely (no CPU regression).
	gp := 0
	for i, s := range strs {
		id, ok := m[s]
		if !ok {
			if len(m) >= qpackStrDictMaxDistinct {
				e.state.colDictTable.set(table)
				e.state.colScratchU64.set(idx)
				return false
			}
			id = uint32(len(m))
			m[s] = id
			table = append(table, s)
			if len(table) == 1 {
				gp = len(s)
			} else {
				gp = commonPrefixLen(table[0][:gp], s)
			}
		}
		idx = append(idx, uint64(id))
		// High-cardinality bail: after the sample, if the column is mostly
		// distinct it will not dict-compress — stop before scanning the rest.
		if i+1 == qpackStrDictSampleN && len(m)*100 > qpackStrDictSampleN*qpackStrDictSampleMaxPct {
			e.state.colDictTable.set(table)
			e.state.colScratchU64.set(idx)
			return false
		}
	}
	e.state.colDictTable.set(table)
	e.state.colScratchU64.set(idx)

	d := len(table)
	if d < 2 {
		// A single distinct value never beats the per-value path (one interned
		// string + a state-repeat run). The gate below would reject it anyway,
		// but make the invariant explicit: the decoder relies on count >= 2 (so
		// bits >= 1, so a non-empty index body) to keep the row count bounded by
		// the buffer. A count==1 dict would carry no body and let a tiny input
		// claim a huge n.
		return false
	}
	bits := bitsForDistinct(d)
	bodyBytes := (n*bits + 7) >> 3
	// Never-worse gate. The distinct table bytes are paid by both forms (the
	// per-value fallback emits each distinct string once too), so they cancel;
	// compare only the dict's fixed overhead + index body against the per-value
	// floor. That floor is n, not the run count: the gathered-column fallback is
	// a per-value loop (writeStringColumn) that does NOT run-length-collapse
	// consecutive identical rows — each of the n rows emits at least a 1-byte
	// string/intern-ref tag, so the fallback costs >= n bytes after the table
	// cancels. (An earlier `runs` floor assumed RLE collapse that never happens;
	// it wrongly rejected dict for skewed low-card columns whose dominant value
	// clusters into few runs, bloating them ~2-3x.) Since distinct <= 256, the
	// index spends <= 8 bits/row, so a bitpacked index strictly beats 1-byte refs
	// for every d < 256 and ties (rejects) at d == 256 — exactly the n floor.
	overhead := 1 + uvarintLen(uint64(d)) + uvarintLen(uint64(n))
	if bodyBytes+overhead >= n {
		return false
	}

	// Table representation: plain (first-seen) vs front-coded (sorted, each entry
	// stored as shared-prefix-len + suffix). The per-row index body is
	// byte-identical between the two forms, so only the table bytes differ. The
	// front-coded form is attempted ONLY when the sort-free global-prefix signal
	// (gp) guarantees it wins: gp >= 2 ⇒ (d-1)*(gp-1) >= d-1 bytes saved, so the
	// encoder never sorts a column front-coding cannot help — no CPU regression
	// on non-prefix-shared data (which keeps the plain dictionary unchanged).
	e.writeHeader()
	out := e.buf
	useFC := false
	if gp >= 2 {
		plainTbl := 0
		for _, s := range table {
			plainTbl += uvarintLen(uint64(len(s))) + len(s)
		}
		var order, inv [qpackStrDictMaxDistinct]uint16
		for k := range d {
			order[k] = uint16(k)
		}
		slices.SortFunc(order[:d], func(a, b uint16) int { return cmp.Compare(table[a], table[b]) })
		fcTbl, prevS := 0, ""
		for k := range d {
			s := table[order[k]]
			pfx := commonPrefixLen(prevS, s)
			fcTbl += uvarintLen(uint64(pfx)) + uvarintLen(uint64(len(s)-pfx)) + (len(s) - pfx)
			inv[order[k]] = uint16(k)
			prevS = s
		}
		if fcTbl < plainTbl { // guaranteed by gp >= 2; the explicit never-larger gate
			useFC = true
			out = append(out, tagColStrDictFC)
			out = appendUvarint(out, uint64(d))
			prevS = ""
			for k := range d {
				s := table[order[k]]
				pfx := commonPrefixLen(prevS, s)
				out = appendUvarint(out, uint64(pfx))
				out = appendUvarint(out, uint64(len(s)-pfx))
				out = append(out, s[pfx:]...)
				prevS = s
			}
			for i := range idx { // remap row indices into the sorted table
				idx[i] = uint64(inv[idx[i]])
			}
		}
	}
	if !useFC {
		// Per-row index codec: flat ceil(log2 d)-bit pack (tagColStrDict) vs the
		// QPack integer picker (tagColStrDictQ). The picker's chosen-codec byte
		// cost (qCost, which already includes the QPack block framing) is compared
		// against the flat body; the QPack form is emitted only when strictly
		// smaller, so the column never grows. The picker captures run-length /
		// dictionary structure a skewed index has but the flat pack cannot exploit.
		// QPack is paired only with the plain table: a front-coded table fires on
		// high-card sorted-prefix data whose index is near-uniform, where the
		// picker cannot win, so the FC branch keeps the flat index.
		codec, mn, forBits, first, minDelta, deltaBits, pforBits, qCost := pickU64Codec(idx)
		// The !qpackConstantOverCap guard keeps qCost honest: when n exceeds the
		// standalone cap, emitQPackUint64 downgrades a constant-body codec (RLE /
		// Dict / const-FOR) to raw, so the emitted block would be ~8n bytes, not
		// qCost — choosing DictQ then could grow the wire past the flat index. It is
		// currently unreachable here (n <= maxColumnarElems == qpackMaxStandaloneCount
		// for every columnar/gathered caller), but gating on it decouples the
		// never-larger guarantee from that constant coincidence.
		if qCost < bodyBytes && !qpackConstantOverCap(n, codec, forBits, deltaBits, pforBits) {
			out = append(out, tagColStrDictQ)
			out = appendUvarint(out, uint64(d))
			for _, s := range table {
				out = appendUvarint(out, uint64(len(s)))
				out = append(out, s...)
			}
			// emitQPackUint64 appends the index block (with its own row count) to
			// e.buf; writeHeader is idempotent so the mid-block call is a no-op for
			// the header already emitted above.
			e.buf = out
			e.emitQPackUint64(idx, codec, mn, forBits, first, minDelta, deltaBits, pforBits)
			return true
		}
		out = append(out, tagColStrDict)
		out = appendUvarint(out, uint64(d))
		for _, s := range table {
			out = appendUvarint(out, uint64(len(s)))
			out = append(out, s...)
		}
	}
	out = appendUvarint(out, uint64(n))
	if bits > 0 {
		start := len(out)
		out = append(out, make([]byte, bodyBytes)...)
		body := out[start : start+bodyBytes]
		var chunk [64]uint64
		for i := 0; i < n; i += len(chunk) {
			end := min(i+len(chunk), n)
			bitpack.PackChunk(body, idx[i:end], bits, i)
		}
	}
	e.buf = out
	return true
}

// dictSampleHighCard reports whether a leading sample of strs is mostly distinct
// (> qpackStrDictSampleMaxPct), using a fixed stack array and a linear scan — no
// map, no allocation. It mirrors the in-loop high-cardinality bail of
// tryWriteStringColumnDict so a mostly-distinct column can exit before the hash
// map is built. The linear O(sample²) compares are cheap because distinct values
// (random IDs) diverge in their first bytes, so most comparisons fail fast.
func dictSampleHighCard(strs []string) bool {
	sampleN := min(len(strs), qpackStrDictSampleN)
	var seen [qpackStrDictSampleN]string
	distinct := 0
	for i := range sampleN {
		s := strs[i]
		fresh := true
		for j := 0; j < distinct; j++ {
			if seen[j] == s {
				fresh = false
				break
			}
		}
		if fresh {
			seen[distinct] = s
			distinct++
		}
	}
	return distinct*100 > sampleN*qpackStrDictSampleMaxPct
}

// commonPrefixLen returns the length of the longest byte prefix shared by a and
// b. Used by the front-coded string-dictionary codec.
func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

// readStringColumn decodes a string column of n values written by
// writeStringColumn: a tagColStrDict dictionary block, a tagColStrFSST block,
// or n per-value strings.
func (d *Decoder) readStringColumn(n int) ([]string, error) {
	if n > 0 && d.i < len(d.buf) && d.buf[d.i] == tagColStrConst {
		return d.readStringColumnConst(n)
	}
	// FSST allocates its own output slab+slice, so dispatch before the per-value
	// make to avoid a dead []string allocation on the FSST path.
	if n > 0 && d.i < len(d.buf) && d.buf[d.i] == tagColStrFSST {
		return d.readStringColumnFSST(n)
	}
	if n > 0 && d.i < len(d.buf) && d.buf[d.i] == tagColStrRaw {
		return d.readStringColumnRaw(n)
	}
	if n > 0 && d.i < len(d.buf) && d.buf[d.i] == tagColStrAlpha {
		return d.readStringColumnAlpha(n)
	}
	out := d.colStrScratch(n)
	if n > 0 && d.i < len(d.buf) && (d.buf[d.i] == tagColStrDict || d.buf[d.i] == tagColStrDictFC || d.buf[d.i] == tagColStrDictQ) {
		var (
			table []string
			idx   []uint32
			err   error
		)
		switch d.buf[d.i] {
		case tagColStrDictFC:
			table, idx, err = d.readStringColumnDictFC(n)
		case tagColStrDictQ:
			table, idx, err = d.readStringColumnDictQ(n)
		default:
			table, idx, err = d.readStringColumnDict(n)
		}
		if err != nil {
			return nil, err
		}
		for i := range n {
			out[i] = table[idx[i]]
		}
		return out, nil
	}
	for i := range n {
		sb, err := d.readStringBytes()
		if err != nil {
			return nil, err
		}
		out[i] = d.materializeStr(sb) // aliases input under noCopy / arena
	}
	return out, nil
}

// colDictTableScratch / colDictIdxScratch return reused per-column dict scratch.
// Both are transient — the only caller (readStringColumn) copies table[idx[i]]
// into the column result before the next column read — so reusing them across
// columns is safe, and they are distinct buffers from the result scratch
// (colScratchStr), so the table->result copy never self-overwrites.
func (d *Decoder) colDictTableScratch(count int) []string {
	st := d.colStateDec()
	if cap(st.colDictTableScr) < count {
		st.colDictTableScr = make([]string, count)
	}
	st.colDictTableScr = st.colDictTableScr[:count]
	return st.colDictTableScr
}

func (d *Decoder) colDictIdxScratch(n int) []uint32 {
	st := d.colStateDec()
	if cap(st.colDictIdxScr) < n {
		st.colDictIdxScr = make([]uint32, n)
	}
	st.colDictIdxScr = st.colDictIdxScr[:n]
	return st.colDictIdxScr
}

// readStringColumnDict decodes a tagColStrDict block (tag at d.i) into a
// distinct table and a per-row index slice of length n. Each row's string
// is table[idx[i]]; the distinct strings are allocated once and shared by
// every row that references them. n is the row count the caller expects
// from the columnar header; the block's own count must match.
func (d *Decoder) readStringColumnDict(n int) (table []string, idx []uint32, err error) {
	d.i++ // consume tagColStrDict
	c64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if c64 < 2 || c64 > qpackStrDictMaxDistinct {
		// The encoder never emits a single-distinct (count==1) dictionary —
		// per-value wins, so the never-worse gate rejects it. Requiring
		// count >= 2 here is therefore lossless for valid streams and, crucially,
		// guarantees bits >= 1 so the index body is non-empty and the row count n
		// is bounded by the remaining buffer below. A count==1 dict would carry
		// no body and let a tiny hostile input drive a huge n / allocation.
		return nil, nil, ErrBadTag
	}
	count := int(c64)
	table = d.colDictTableScratch(count)
	for i := range count {
		l64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, nil, ErrInvalidLength
		}
		d.i += nr
		if l64 > uint64(len(d.buf)-d.i) {
			return nil, nil, ErrShortBuffer
		}
		l := int(l64)
		table[i] = d.materializeStr(d.buf[d.i : d.i+l]) // aliases input under noCopy / arena
		d.i += l
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, nil, ErrTypeMismatch
	}
	// count >= 2 ⇒ bits >= 1 ⇒ the index body is non-empty, so this check
	// bounds n by the remaining buffer BEFORE any n-sized allocation; a tiny
	// input can no longer drive a large idx/tmp/output slice.
	bits := bitsForDistinct(count)
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bits) {
		return nil, nil, ErrShortBuffer
	}
	bodyBytes := (n*bits + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx = d.colDictIdxScratch(n)
	// Reuse the shared transient unpack scratch (as readPackedDictUint64Slice
	// does): tmp holds the bit-unpacked dictionary indices, fully written by
	// Unpack below before any read, and only mapped into idx — never aliased
	// into the returned slices. count >= 2 ⇒ bits >= 1 (checked above), so
	// Unpack always writes all n slots; no stale-tail under-fill is possible.
	if cap(d.deltaScratch) < n {
		d.deltaScratch = make([]uint64, n)
	}
	tmp := d.deltaScratch[:n]
	bitpack.Unpack(tmp, body, bits)
	for i, v := range tmp {
		if v >= c64 {
			return nil, nil, ErrBadTag
		}
		idx[i] = uint32(v)
	}
	return table, idx, nil
}

// readStringColumnDictQ decodes a tagColStrDictQ block (tag at d.i): a plain
// distinct table (identical layout to tagColStrDict) followed by a QPack-coded
// per-row index instead of a flat bitpack. The index block carries its own row
// count; the decoded length must equal n (the columnar header count) and every
// index must address a table slot. The QPack reader bounds its own allocation by
// the remaining buffer, so a hostile inner count cannot force an oversized slice.
func (d *Decoder) readStringColumnDictQ(n int) (table []string, idx []uint32, err error) {
	d.i++ // consume tagColStrDictQ
	c64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if c64 < 2 || c64 > qpackStrDictMaxDistinct {
		// count >= 2 mirrors tagColStrDict: the encoder never emits a single-
		// distinct dictionary (per-value wins), so this is lossless for valid
		// streams while rejecting a hostile count.
		return nil, nil, ErrBadTag
	}
	count := int(c64)
	table = d.colDictTableScratch(count)
	for i := range count {
		l64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, nil, ErrInvalidLength
		}
		d.i += nr
		if l64 > uint64(len(d.buf)-d.i) {
			return nil, nil, ErrShortBuffer
		}
		l := int(l64)
		table[i] = d.materializeStr(d.buf[d.i : d.i+l]) // aliases input under noCopy / arena
		d.i += l
	}
	// QPack index block: peek the codec tag, decode, validate length and range.
	if d.i >= len(d.buf) {
		return nil, nil, ErrShortBuffer
	}
	idx64, err := d.readQPackUint64(d.buf[d.i])
	if err != nil {
		return nil, nil, err
	}
	if len(idx64) != n {
		return nil, nil, ErrTypeMismatch
	}
	idx = d.colDictIdxScratch(n)
	for i, v := range idx64 {
		if v >= c64 {
			return nil, nil, ErrBadTag
		}
		idx[i] = uint32(v)
	}
	return table, idx, nil
}

// readStringColumnDictFC decodes a tagColStrDictFC block (tag at d.i): a sorted,
// front-coded distinct table plus a per-row index slice of length n. Each table
// entry is reconstructed as prev[:sharedPrefixLen] + suffix; all distinct strings
// are materialised into ONE slab (a single allocation for the whole table) and
// returned as views into it. Bounds mirror readStringColumnDict.
func (d *Decoder) readStringColumnDictFC(n int) (table []string, idx []uint32, err error) {
	d.i++ // consume tagColStrDictFC
	c64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if c64 < 2 || c64 > qpackStrDictMaxDistinct {
		// count >= 2 keeps bits >= 1 (non-empty index body) so n is buffer-bounded.
		return nil, nil, ErrBadTag
	}
	count := int(c64)

	// Reconstruct the sorted front-coded table into one growing slab: entry k is
	// prev[:prefixLen] + suffix. starts/lens record each entry's region in the
	// FINAL slab (offsets are stable across the appends' reallocations).
	var starts, lens [qpackStrDictMaxDistinct]int
	slab := make([]byte, 0, 64)
	prevStart, prevLen := 0, 0
	for i := range count {
		p64, k := readUvarint(d.buf[d.i:])
		if k <= 0 {
			return nil, nil, ErrInvalidLength
		}
		d.i += k
		s64, k := readUvarint(d.buf[d.i:])
		if k <= 0 {
			return nil, nil, ErrInvalidLength
		}
		d.i += k
		// The shared prefix cannot exceed the previous entry's length (entry 0
		// has prevLen 0 → prefix 0); the suffix is bounded by the remaining buffer.
		if p64 > uint64(prevLen) || s64 > uint64(len(d.buf)-d.i) {
			return nil, nil, ErrBadTag
		}
		p, s := int(p64), int(s64)
		// Cap cumulative slab growth: the prefix is copied from the previous entry
		// (no wire cost), so a hostile table (entry 0 suffix S, then N entries with
		// prefix=S, suffix=0) amplifies to N*S with no per-entry guard catching it.
		// The uncompressed distinct table cannot legitimately exceed the columnar
		// byte ceiling.
		if int64(len(slab))+int64(p)+int64(s) > int64(maxColumnarBytes) {
			return nil, nil, ErrInvalidLength
		}
		start := len(slab)
		slab = append(slab, slab[prevStart:prevStart+p]...) // shared prefix from prev
		slab = append(slab, d.buf[d.i:d.i+s]...)            // suffix from wire
		d.i += s
		starts[i], lens[i] = start, p+s
		prevStart, prevLen = start, p+s
	}
	table = d.colDictTableScratch(count)
	for i := range count {
		table[i] = unsafestr.String(slab[starts[i] : starts[i]+lens[i]]) // view into the owned slab
	}

	// Index body: identical layout to tagColStrDict. Bound n by the remaining
	// buffer (bits >= 1 since count >= 2) BEFORE the n-sized allocations.
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, nil, ErrTypeMismatch
	}
	bits := bitsForDistinct(count)
	rem := uint64(len(d.buf) - d.i)
	if n64 > rem*8/uint64(bits) {
		return nil, nil, ErrShortBuffer
	}
	bodyBytes := (n*bits + 7) >> 3
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	idx = d.colDictIdxScratch(n)
	if cap(d.deltaScratch) < n {
		d.deltaScratch = make([]uint64, n)
	}
	tmp := d.deltaScratch[:n]
	bitpack.Unpack(tmp, body, bits)
	for i, v := range tmp {
		if v >= c64 {
			return nil, nil, ErrBadTag
		}
		idx[i] = uint32(v)
	}
	return table, idx, nil
}
