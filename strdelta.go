package qdf

import (
	"sync/atomic"

	"github.com/alex60217101990/qdf/internal/bumparena"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Front-delta string values (tagStrDelta) for the row-major struct path.
//
// The columnar path has its own front-delta codec (qpack_strfrontdelta.go): a
// block with an anchor interval, because a column is contiguous and a reader
// may want to seek into it. This one is per value, because row-major
// interleaves fields — there is no column to anchor and no seek to preserve,
// so the form carries no header at all.
//
// The prefix compare is shared with that codec (frontDeltaCommonPrefix): a
// 64-bit XOR is zero exactly when eight bytes match, and TrailingZeros64>>3
// turns the first differing bit into the first differing byte. Measured 3x
// faster than a byte loop on this codebase's string lengths.

// strDeltaEmitted counts tagStrDelta emissions across the process. Acceptance
// tests assert on it rather than on wire size, because a size assertion here is
// vacuous: interning alone shrinks these payloads, so a smaller wire proves
// nothing about whether this form ran. That is precisely how the columnar codec
// on this branch looked healthy while never running once.
var strDeltaEmitted atomic.Int64

// strDeltaCount gates the two counters below. They exist so acceptance tests can
// assert how many values took the delta form — a wire-size assertion is vacuous
// here, since interning alone shrinks these payloads. But an atomic increment on
// every eligible string is not free: it was 14% of writeStringField's profile on
// the RTB encode, spent entirely on instrumentation.
//
// A package-level bool costs a predictable branch instead. Only tests set it.
var strDeltaCount bool

// strDeltaProbes counts prefix comparisons performed. The gate's whole purpose
// is to keep this well below the number of eligible values on data the delta
// cannot help, and only a counter shows that — strDeltaEmitted stays at zero
// with or without a gate there.
var strDeltaProbes atomic.Int64

const (
	// strDeltaProbeN values with no win mute a field. Small enough that a
	// hostile field pays the comparison only briefly.
	strDeltaProbeN = 32
	// strDeltaRearmN values later, probing resumes. The gate may cost CPU and
	// must never cost wire, so it cannot latch: a field whose data turns
	// compressible part-way through is recovered instead of forfeited. Worst
	// case is one wasted comparison per 512 values, ~0.2%.
	strDeltaRearmN = 512
	// The delta is emitted only when its cost is below this fraction of the
	// form it replaces: num/den = 1/2 means it must halve the value.
	strDeltaMinGainNum = 1
	strDeltaMinGainDen = 2
)

// strDeltaGate is the per-field probe state. Encoder-side only: the wire stays
// self-describing, so there is no decoder mirror and nothing to desynchronise.
type strDeltaGate struct {
	seen  uint16
	wins  uint16
	muted bool
}

// writeStringField writes one struct string field.
//
// It differs from WriteString in exactly one place: when the value is a FIRST
// SIGHTING — the case WriteString spells as tagInternStr — the delta against
// this field's previous value may be smaller, and then it is written instead.
// Both forms register the value and continue the state chain identically, so
// the choice is a plain byte count between two interchangeable encodings.
//
// An already-interned value is not considered at all. emitStateRef spends one
// or two bytes there and this form needs at least three, so there is nothing to
// win — and a repeated value, where pfx == len(s) makes the delta look
// cheapest, is exactly where a careless comparison would lose bytes.
func (e *Encoder) writeStringField(s string, base *string, g *strDeltaGate) {
	st := e.state
	// An empty base means this is the field's FIRST value — there is no previous
	// row to resemble, the common prefix is zero, and the delta loses by
	// construction. Checking that first costs one compare and skips the
	// eligibility test, the prefix scan and the cost arithmetic.
	//
	// It is not a micro-optimisation for the first row of a batch; it is the
	// whole cost for a struct encoded ONCE. A payload that is a single struct
	// rather than a slice of them has no second row for any field, so every
	// value took the full delta path to reach a conclusion that was fixed in
	// advance — measured as the entire regression on the Flat, Nested and Deep16
	// encodes, which are exactly that shape.
	if *base == "" {
		e.WriteString(s)
		*base = s
		return
	}
	if s == *base {
		// A consecutive repeat. The intern path spends ONE byte on it
		// (tagStateRepeat) and the delta cannot go below three, so the delta
		// must not be offered — but with the prefix compare running before the
		// intern lookup it would score pfx == len(s), a cost of ~3 bytes, and
		// win the threshold against the FIRST-SIGHTING cost it is compared
		// with. That is a wire regression on exactly the low-cardinality
		// fields this codec is supposed to leave alone.
		//
		// The compare is also cheaper than what it replaces: a full-length
		// prefix scan of two identical strings.
		e.WriteString(s)
		return
	}
	if !e.strDeltaEligible(s) {
		e.WriteString(s)
		*base = s
		return
	}

	// The prefix comparison runs BEFORE the intern lookup, and that ordering is
	// the whole optimisation.
	//
	// A delta wins on values that are nearly unique — that is what makes them
	// resemble the row above without repeating it. Such a value is almost never
	// referenced again, so hashing it and inserting it into the intern table is
	// work neither side gets anything back for: a full-string hash and an insert
	// on the encoder, an append and its retention on the decoder. The prefix
	// compare, in contrast, stops at the first differing word, so it is cheap
	// exactly where it is about to lose.
	//
	// So measure first. When the delta wins, emit it and skip the intern
	// machinery entirely; only a value the delta declines pays for the lookup.
	// A value the intern table already holds costs one or two bytes there, which
	// no delta can undercut — its cheapest form is three. Ask WITHOUT
	// installing: installing is what this codec avoids for the values it does
	// claim, and doing it here to answer a question would give that up.
	//
	// The equality fast path above catches only a value identical to the base.
	// This catches one identical to any earlier value of the field, which the
	// prefix compare would otherwise let win the threshold whenever it happens
	// to share a long prefix with the base.
	id, interned, keyHash := st.lookupOnly(s)
	if interned {
		e.emitStateRef(id)
		*base = s
		return
	}

	if !g.muted {
		if strDeltaCount {
			strDeltaProbes.Add(1)
		}
		p := frontDeltaCommonPrefix(*base, s)
		internCost := 1 + uvarintLen(uint64(len(s))) + len(s)
		// The delta must not merely win, it must win BIG: every emission adds
		// bytes the decoder has to materialise, because a delta value is
		// contiguous nowhere and cannot alias the read buffer the way an
		// interned one does. Saving two bytes of wire is not worth a full copy;
		// halving the value is.
		win := strDeltaCostAt(p, s)*strDeltaMinGainDen < internCost*strDeltaMinGainNum
		g.seen++
		if win {
			g.wins++
		}
		if g.seen >= strDeltaProbeN {
			g.muted = g.wins == 0
			g.seen, g.wins = 0, 0
		}
		if win {
			e.buf = appendStrDeltaAt(e.buf, p, s)
			if strDeltaCount {
				strDeltaEmitted.Add(1)
			}
			// Inline semantics: the value never entered the intern table, so a
			// following tagStateRepeat must not resurrect whatever ID was last
			// on the chain. The decoder drops it on every inline read; the
			// encoder drops it here, or the two disagree about what "previous"
			// means — a desync, not a bigger wire.
			st.lastID = lruInvalidID
			*base = s
			return
		}
	} else {
		g.seen++
		if g.seen >= strDeltaRearmN {
			g.muted, g.seen, g.wins = false, 0, 0
		}
	}

	// The delta declined (or the field is muted): the value takes the path it
	// would have taken without this feature at all.
	// Absent, and lookupOnly already hashed it: install with that hash rather
	// than paying for a second one.
	id = st.assignHashed(s, keyHash)
	e.buf = append(e.buf, tagInternStr)
	e.buf = appendUvarint(e.buf, uint64(len(s)))
	e.buf = appendString(e.buf, s)
	if st.lastID != lruInvalidID && e.pairPred {
		st.pairRecord(st.lastID, id)
	}
	st.lastID = id
	*base = s
}

// strDeltaEligible reports whether the delta form may be considered at all:
// Dense, a live and unsuspended state table, and a string long enough to be
// interned. Outside that the existing path runs verbatim, so nothing about
// non-Dense encoding changes.
func (e *Encoder) strDeltaEligible(s string) bool {
	st := e.state
	return st != nil && !e.stateSuspended && e.opts.Has(OptDense) &&
		len(s) >= e.minIntern && int(st.internLoad) < e.maxStateEntries
}

// strDeltaCost is the exact byte count appendStrDelta will write. The caller
// compares it against the first-sighting form — 1 + uvarintLen(len(s)) + len(s)
// — and emits the delta only when it is strictly smaller, which is what makes
// the form never worse by construction rather than by a gate.
func strDeltaCost(base, s string) int {
	return strDeltaCostAt(frontDeltaCommonPrefix(base, s), s)
}

// strDeltaCostAt is the cost for a prefix already computed. The caller measures
// once and writes with the same number: computing it twice — once to price the
// form and once to emit it — doubled the encoder's only real work.
func strDeltaCostAt(p int, s string) int {
	return 1 + uvarintLen(uint64(p)) + uvarintLen(uint64(len(s)-p)) + (len(s) - p)
}

// appendStrDelta encodes s against base into a fresh slice. The encoder writes
// through appendStrDeltaAt with a prefix it has already measured; this form
// exists for callers that have neither, and computes the prefix itself.
func appendStrDelta(base, s string) []byte {
	return appendStrDeltaAt(nil, frontDeltaCommonPrefix(base, s), s)
}

func appendStrDeltaAt(buf []byte, p int, s string) []byte {
	buf = append(buf, tagStrDelta)
	buf = appendUvarint(buf, uint64(p))
	buf = appendUvarint(buf, uint64(len(s)-p))
	return appendString(buf, s[p:])
}

// readStrDelta decodes one tagStrDelta value against base and returns the
// reconstructed string, the bytes consumed, and an error.
//
// Every field is validated before a byte is copied. A hostile wire can claim a
// prefix longer than the base — which would slice past the end of a string the
// reader does not own — or a middle longer than the buffer. Both are rejected
// here rather than deeper, where the copy would already have happened.
func readStrDelta(buf []byte, base string) (string, int, error) {
	return readStrDeltaInto(buf, base, nil)
}

// readStrDeltaInto is readStrDelta with the output cut from st's chunk arena
// when st is non-nil, which is what keeps decode allocations flat.
func readStrDeltaInto(buf []byte, base string, st *decState) (string, int, error) {
	if len(buf) == 0 || buf[0] != tagStrDelta {
		return "", 0, ErrBadTag
	}
	i := 1
	p64, n := readUvarint(buf[i:])
	if n <= 0 {
		return "", 0, ErrShortBuffer
	}
	i += n
	m64, n := readUvarint(buf[i:])
	if n <= 0 {
		return "", 0, ErrShortBuffer
	}
	i += n
	if p64 > uint64(len(base)) {
		return "", 0, ErrInvalidLength
	}
	if m64 > uint64(len(buf)-i) {
		return "", 0, ErrShortBuffer
	}
	p, m := int(p64), int(m64)
	var out []byte
	if st != nil {
		out = st.strDeltaAlloc(p + m)
	} else {
		out = make([]byte, p+m)
	}
	copy(out, base[:p])
	copy(out[p:], buf[i:i+m])
	// The bytes are ours — cut from the arena or freshly made — so the header
	// can alias them instead of paying a second copy. Copying here would undo
	// the arena entirely.
	return unsafestr.String(out), i + m, nil
}

// --- per-field base storage -------------------------------------------------
//
// tagStrDelta codes against the previous value of the SAME field, so the base
// has to outlive the row: each row is its own encodeStruct call. It therefore
// lives on the pooled state, beside the shape bindings that key it.
//
// The encoder keys by *typeDesc, which it always has. The decoder cannot: a
// field the target struct does not declare has no typeDesc, and that field's
// base still has to advance or the next value of it decodes against a stale
// one. So the decoder keys by wire shape ID instead.

// strDeltaBases returns this type's per-field base slice, growing it if the
// type is seen with more fields than before.
//
// The one-entry cache is the whole point: a slice of one struct type — the case
// that matters — hits it on every row, so the lookup is a pointer compare
// rather than a scan.
func (e *encState) strDeltaBases(td *typeDesc, nFields int) ([]string, []strDeltaGate) {
	// TWO cache slots, not one. A nested struct alternates between the parent's
	// type and the child's on every field, so a single slot misses every time
	// and falls through to the linear scan below — measured at 37 ns on a 222 ns
	// nested encode, 16.7% of it. Two slots turn that alternation into a hit.
	if e.lastDeltaTd == td && len(e.lastDeltaBases) >= nFields {
		return e.lastDeltaBases, e.lastDeltaGates
	}
	if e.prevDeltaTd == td && len(e.prevDeltaBases) >= nFields {
		// Promote so a run on this type keeps hitting the first slot.
		e.lastDeltaTd, e.prevDeltaTd = e.prevDeltaTd, e.lastDeltaTd
		e.lastDeltaBases, e.prevDeltaBases = e.prevDeltaBases, e.lastDeltaBases
		e.lastDeltaGates, e.prevDeltaGates = e.prevDeltaGates, e.lastDeltaGates
		return e.lastDeltaBases, e.lastDeltaGates
	}
	for i, t := range e.strDeltaTd {
		if t == td {
			b, g := e.strDeltaBase[i], e.strDeltaGate[i]
			if len(b) < nFields {
				b = append(b, make([]string, nFields-len(b))...)
				g = append(g, make([]strDeltaGate, nFields-len(g))...)
				e.strDeltaBase[i], e.strDeltaGate[i] = b, g
			}
			e.prevDeltaTd, e.prevDeltaBases, e.prevDeltaGates = e.lastDeltaTd, e.lastDeltaBases, e.lastDeltaGates
			e.lastDeltaTd, e.lastDeltaBases, e.lastDeltaGates = td, b, g
			return b, g
		}
	}
	b := make([]string, nFields)
	g := make([]strDeltaGate, nFields)
	e.strDeltaTd = append(e.strDeltaTd, td)
	e.strDeltaBase = append(e.strDeltaBase, b)
	e.strDeltaGate = append(e.strDeltaGate, g)
	e.prevDeltaTd, e.prevDeltaBases, e.prevDeltaGates = e.lastDeltaTd, e.lastDeltaBases, e.lastDeltaGates
	e.lastDeltaTd, e.lastDeltaBases, e.lastDeltaGates = td, b, g
	return b, g
}

// strDeltaBases returns the per-field base slice for a wire shape ID.
func (d *decState) strDeltaBases(shapeID uint32, nFields int) []string {
	if int(shapeID) >= len(d.strDeltaBase) {
		grow := int(shapeID) + 1 - len(d.strDeltaBase)
		d.strDeltaBase = append(d.strDeltaBase, make([][]string, grow)...)
	}
	b := d.strDeltaBase[shapeID]
	if len(b) < nFields {
		b = append(b, make([]string, nFields-len(b))...)
		d.strDeltaBase[shapeID] = b
	}
	return b
}

// strDeltaResetEnc clears the encoder's bases in place. The slices hold string
// headers from the previous message; a bare truncation would keep them live in
// the tail and pin that message's memory for the lifetime of the pooled state —
// the same hazard decState.reset documents for stringValues.
// strDeltaResetEnc clears the per-field state for the next message while
// KEEPING the per-type slices themselves.
//
// Dropping the type bindings would make the next message reallocate a base and
// a gate slice for every struct type it touches — measured at +10 allocs/op on
// a small nested payload, which never amortizes because a small payload is
// encoded once. typeDesc pointers are process-global and stay valid, so the
// bindings survive; only the values in them must not.
//
// Bounded: a process that encodes an unbounded variety of types would otherwise
// accumulate one entry per type forever. Past the cap the table is dropped
// whole rather than grown, matching how the intern table releases after a
// streak of small messages.
func (e *encState) strDeltaResetEnc() {
	if len(e.strDeltaTd) > maxRetainedDeltaTypes {
		clear(e.strDeltaTd)
		e.strDeltaTd = e.strDeltaTd[:0]
		e.strDeltaBase = e.strDeltaBase[:0]
		e.strDeltaGate = e.strDeltaGate[:0]
		e.lastDeltaTd, e.lastDeltaBases, e.lastDeltaGates = nil, nil, nil
		e.prevDeltaTd, e.prevDeltaBases, e.prevDeltaGates = nil, nil, nil
		return
	}
	for i := range e.strDeltaBase {
		clear(e.strDeltaBase[i])
		clear(e.strDeltaGate[i])
	}
	// The one-entry cache still points at a live slice, so it stays: a stream of
	// the same type hits it on the first field of the next message too.
}

// maxRetainedDeltaTypes caps how many struct types keep their per-field state
// across a pooled reset.
const maxRetainedDeltaTypes = 64

func (d *decState) strDeltaResetDec() {
	// A fresh allocator, not Bump.Reset: the old block's bytes are referenced by
	// the intern table and by strings handed to the previous caller.
	d.strDeltaBump = bumparena.New()
	for i := range d.strDeltaBase {
		clear(d.strDeltaBase[i])
	}
}

// strDeltaTagAdvancesBase reports whether a value opening with this tag would
// be read through readStringBytes — and so would advance the current field's
// delta base.
//
// It exists for fields the target struct does not declare. The encoder advances
// a field's base on every value it writes for that field, so the decoder must
// advance it too; calling Skip() instead leaves the base a row behind and the
// next delta on that field rebuilds against the wrong prefix.
//
// State-ref forms are included: they resolve to strings and the encoder counted
// them. Binary forms are included as well, and that is deliberate — a []byte
// field never has a delta written for it, so its base is never consulted and a
// spurious advance costs nothing, whereas guessing wrong in the other direction
// corrupts a string field.
func strDeltaTagAdvancesBase(b byte) bool {
	switch {
	case b >= tagFixstr && b <= tagFixstr|tagFixstrMask:
		return true
	case b == tagStr8, b == tagStr16, b == tagStr32:
		return true
	case b == tagBin8, b == tagBin16, b == tagBin32:
		return true
	case b == tagInternStr, b == tagInternBin, b == tagStateRef:
		return true
	case b == tagStateRepeat, b == tagStateMTF, b == tagStatePair:
		return true
	case b == tagStrDelta:
		return true
	}
	return false
}

// strDeltaAlloc returns n uninitialised bytes owned by the decoder state.
//
// A delta value is base[:pfx] + mid, contiguous nowhere: unlike tagInternStr —
// whose bytes are a sub-slice of the read buffer and cost nothing — it must be
// materialised. A make() per value doubled decode allocations on the access-log
// profile (30.9k -> 60.7k) and cost most of the decode regression.
//
// It goes through the same bumparena the decoder already uses for arena-backed
// strings rather than a second allocator beside it.
func (d *decState) strDeltaAlloc(n int) []byte { return d.strDeltaBump.Alloc(n) }
