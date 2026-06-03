package qdf

import (
	"hash/maphash"
	"strings"

	"github.com/alex60217101990/qdf/internal/internarena"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// internHashSeed is the shared maphash seed for the flat intern
// table. Per-process random; values are stable across goroutines
// but differ across binary invocations (no semantic impact — the
// intern table is per-encoder and reset between Marshals).
var internHashSeed = maphash.MakeSeed()

// internSlot is one entry in the flat hash table that replaces the
// old map[string]uint32 intern dictionary. Profiling on telemetry
// workloads showed Go's map[string]uint32 spending ~17 ns/op in
// mapaccess2_faststr (string hash + bucket walk + memequal).
// Open-addressing on a contiguous []internSlot with the hash
// precomputed and stored alongside lets the hot path read one
// cache line, compare the hash, then `==` the key — saving ~5 ns
// per lookup at the cost of one extra 8 B slot field.
//
// hash == 0 reserves the "empty slot" sentinel; computed hashes
// that fall on 0 are bumped to 1 before storage.
//
//	hash 8 B  +  key 16 B  +  id 4 B  +  pad 4 B  =  32 B  →  2 slots / cache line
type internSlot struct {
	hash uint64
	key  string
	id   uint32
	_    uint32
}

// Initial intern table size. Doubles when load > 0.5 so the
// linear-probing chain stays short. 64 covers the common
// telemetry / config payloads (a few dozen distinct values) with
// no resize at all.
const internTableInitSize = 64

// Intern table backing Dense mode. The encoder maintains a string→ID
// map; the decoder maintains the matching ID-ordered list of byte
// slices. IDs are assigned in encode order starting at 0. There is no
// eviction — callers bound lifetime by resetting or recycling.

// pairPredK is the wire bound on the rank varuint emitted after
// tagStatePair. The current predictor stores top-1 only (K=1) so a
// hit always emits rank=0 and the decoder rejects any rank ≥ 1 as
// malformed. The constant stays around so the wire-side validation
// reads the same on both sides.
const pairPredK = 1

// Pair predictor storage: []uint32 indexed by prev intern ID. The
// stored value is `succ+1` so an empty slot is zero, which lets
// reset() use the runtime `clear()` builtin (a memclr) instead of a
// per-element loop. Sentinel choice trades 1 bit of representable
// range we never use (max intern id ≤ maxStateEntries < ^uint32(0)-1)
// for a cleaner reset path.
//
// Top-1 vs the previous K=4 ring: 4 bytes per slot down from 17, so
// at default maxStateEntries=16384 the predictor's residual capacity
// drops from ~280 KiB to 64 KiB per encoder (-77 %). Hit rate on
// strictly cyclic workloads (A→{B,C,D,E,B,C,…}) falls to zero, but
// on the common stable-transition case (A→B→A→B) top-1 catches every
// hit the ring did. Real telemetry workloads sit close to the
// stable case.
const pairPredEmpty uint32 = 0

// mruRingSize is the side-cache covering recent emit history for fast
// MTF rank discovery. Scanning a contiguous 128-entry uint16 ring is
// dominated by sequential L1 hits (cache-prefetcher friendly) and
// completes in roughly N×1.5ns. The canonical LRU linked list still
// holds every ID; the ring just shortcuts the rank-walk on the hot
// path. 128 covers the full 1-byte rank space (uvarintLen(rank)==1
// for rank ≤ 127), which is exactly the range where MTF beats the
// raw 2-byte state-ref encoding most ids land in.
//
// uint16 not uint32: the encoder caps maxStateEntries at 1<<14 (=
// 16 384), well under the uint16 range. Halving the slot width packs
// the whole ring into 256 B (4 × 64 B cache lines) instead of 512 B
// — a sequential scan reads half as many lines and finishes faster
// on the same prefetcher budget. mruEmpty (0xFFFF) is reserved as
// the "no id here" sentinel; the runtime id space stays below it.
//
// Power of two so the modulo collapses to a single AND.
const (
	mruRingSize        = 128
	mruRingMask        = mruRingSize - 1
	mruEmpty    uint16 = 0xFFFF
)

type encState struct {
	// Hot scalars first so they share a single cache line with the
	// adjacent mruHead and the map header. lastID + lruHead + mruHead
	// are touched on every state-ref emit; co-locating them with
	// mruRing keeps the per-emit footprint at 1-2 cache lines.
	lastID     uint32
	lruHead    uint32
	mruHead    uint32
	internLoad uint32 // number of occupied slots in internTable

	// mruRing is a side-cache of the last mruRingSize state-ref emit
	// IDs. Each new emit pushes its ID at mruRing[mruHead&mask] and
	// bumps mruHead. Rank discovery scans backwards from mruHead: a
	// hit at offset r means r OTHER ids were emitted after this one,
	// which is exactly the LRU chain rank. Storing the ring as uint16
	// (id space fits) cuts the scan footprint to 256 B / 4 cache
	// lines.
	mruRing [mruRingSize]uint16

	// internTable is a flat open-addressed hash table replacing the
	// old map[string]uint32. See internSlot for layout. Hot —
	// touched on every WriteString. Linear probing keeps the access
	// pattern sequential and predictable; load is kept below 0.5 by
	// doubling on growth so probe chains stay short (typically 1).
	internTable []internSlot

	// Move-to-front LRU over intern IDs. lruHead is the ID at rank 0
	// (most recently emitted state-ref or freshly interned). lruLink
	// packs the previous + next ids for each chain slot into a single
	// uint32 (low 16 bits = prev, high 16 bits = next). IDs are
	// bounded by maxStateEntries (1 << 14) so they fit in 16 bits
	// with 0xFFFF reserved as the "no neighbour" sentinel. Packing
	// halves the cache lines a lruMoveFront has to touch (one array
	// instead of two) while keeping the unlink/insert update O(1).
	lruLink []uint32

	// Pair predictor: top-1 successor per prev intern ID. Slot stores
	// `succ+1` so zero = empty; this keeps Reset() on a memclr fast
	// path while preserving the ability to predict succ==0 cleanly.
	// See the pairPredEmpty / pairPredK constants and the
	// pairLookup / pairRecord methods for the lookup contract.
	pairPred []uint32

	// Shape table for tagMapShape. shapes is indexed by id-1 (id 0 is
	// reserved on the wire to mean "declare"). shapeBindings is a
	// small linear-scan registry of (typeDesc → wire ID). Typical
	// streams emit a handful of struct types, so a slice beats a map
	// on lookup cost AND keeps the race-detector instrumentation off
	// the hot path. lastShapeTd / lastShapeID memoise the
	// most-recent successful shape lookup; shapeCount is the running
	// total of declared shape IDs.
	lastShapeTd   *typeDesc
	lastShapeID   uint32
	shapeCount    uint32
	shapeBindings []shapeBinding

	// Columnar shape table (tagColStruct). Separate from shapeBindings
	// because columnar shapes carry field kinds. Keyed by structural
	// identity (names + kinds) since the same struct type always produces
	// the same columnar shape.
	colShapeNames [][]string
	colShapeKinds [][]colKind
	// Pooled transpose scratch, reused across columns and across calls.
	colScratchI64  []int64
	colScratchU64  []uint64
	colScratchF64  []float64
	colScratchBool []bool
	colScratchStr  []string // gathered string column values
	colDictTable   []string // distinct table for the string-dict codec
	colMaskScratch []byte   // presence bitmap for nullable columns
	// FSST codec scratch, reused across columns (same lifetime as colDictTable).
	fsstScratch []byte // compressed bytes for all rows, concatenated
	fsstLens    []int  // per-row compressed lengths
	// strDictMap maps a string column's distinct values to dense indices
	// while the string-dict codec decides/encodes. Reused (cleared) per
	// column to avoid a per-column map allocation.
	strDictMap map[string]uint32

	// arena owns the byte storage that backs every intern key the
	// encoder allocates — accessed only on intern miss, kept at the
	// end so the hot fields above share earlier cache lines.
	arena internarena.Arena
}

// shapeBinding is a (typeDesc → wire shape ID) pair. Stored in a
// linear-scan slice on the encoder state so the hot path adds no map
// access (and no allocation under -race).
type shapeBinding struct {
	td *typeDesc
	id uint32
}

const lruInvalidID = ^uint32(0)

func newEncState() *encState {
	// arena is zero-value initialised here — its slab is lazily
	// allocated on first Put (see internarena.Arena.Put).
	e := &encState{
		internTable: make([]internSlot, internTableInitSize),
		lruHead:     lruInvalidID,
		lastID:      lruInvalidID,
	}
	// Prime ring with sentinels so a scan never matches id 0 by
	// accident before the ring has been written.
	for i := range e.mruRing {
		e.mruRing[i] = mruEmpty
	}
	return e
}

// Soft caps on per-encoder state retention across Reset(). A single
// payload that pushes any of these past their threshold is dropped
// rather than pinned to the pooled encoder forever — long-running
// services with bursty traffic keep a bounded resident set instead
// of growing to peak and staying there.
//
// Numbers picked so that a "typical" telemetry batch (≤ a few
// thousand interned strings, ≤ a few thousand state-ref ids) stays
// under every cap; only outlier payloads trigger the shrink.
const (
	maxRetainedIDs      = 4096
	maxRetainedLRUCap   = 4096
	maxRetainedPairCap  = 4096
	maxRetainedShapeCap = 1024
)

func (e *encState) reset() {
	// Intern table shrink. The flat hash table doubles when load
	// exceeds 0.5; long-running services that occasionally take a
	// burst payload would otherwise pin the high-water-mark table
	// to the pool forever. Drop oversized backing arrays here; in
	// the steady-state path the table fits below the cap and only
	// gets memcleared (cheap), which keeps the slot pages warm.
	//
	// Order: clear / rebuild BEFORE arena.Reset. The slot.key
	// fields alias arena bytes; arena.Reset rolls cursors back and
	// the next Put overwrites the prior payload area, so any
	// surviving aliased key would read garbage.
	if cap(e.internTable) > maxRetainedIDs*2 {
		e.internTable = make([]internSlot, internTableInitSize)
	} else {
		for i := range e.internTable {
			e.internTable[i] = internSlot{}
		}
	}
	e.internLoad = 0
	e.arena.Reset() // Arena has its own watermark, see internarena.

	e.lastID = lruInvalidID
	e.lruHead = lruInvalidID
	if cap(e.lruLink) > maxRetainedLRUCap {
		e.lruLink = nil
	} else {
		e.lruLink = e.lruLink[:0]
	}

	// pairPred slice: clear in place if under the cap (memclr-fast
	// because the empty sentinel is zero), drop the backing array
	// otherwise.
	if cap(e.pairPred) > maxRetainedPairCap {
		e.pairPred = nil
	} else {
		clear(e.pairPred)
	}

	e.shapeCount = 0
	if cap(e.shapeBindings) > maxRetainedShapeCap {
		e.shapeBindings = nil
	} else {
		e.shapeBindings = e.shapeBindings[:0]
	}
	e.lastShapeTd = nil
	e.lastShapeID = 0

	if cap(e.colShapeNames) > maxRetainedShapeCap {
		e.colShapeNames = nil
		e.colShapeKinds = nil
	} else {
		e.colShapeNames = e.colShapeNames[:0]
		e.colShapeKinds = e.colShapeKinds[:0]
	}
	if cap(e.colScratchI64) > maxRetainedIDs {
		e.colScratchI64, e.colScratchU64, e.colScratchF64, e.colScratchBool = nil, nil, nil, nil
	}
	// FSST compressed-bytes scratch can grow to the largest string column seen;
	// drop it past the retention cap so the pool does not pin multi-MB buffers
	// (mirrors the numeric scratch policy above).
	if cap(e.fsstScratch) > maxRetainedIDs {
		e.fsstScratch = nil
		e.fsstLens = nil
	}

	// Ring side-cache: re-prime with sentinels so post-reset emits
	// can't false-match a stale id 0.
	for i := range e.mruRing {
		e.mruRing[i] = mruEmpty
	}
	e.mruHead = 0
}

// shapeForType returns the wire shape ID bound to t in this encoder's
// state, or 0 if none. Pair with shapeBindType after a declaration.
//
//go:nosplit
func (e *encState) shapeForType(t *typeDesc) uint32 {
	if e.lastShapeTd == t && e.lastShapeID != 0 {
		return e.lastShapeID
	}
	for i := range e.shapeBindings {
		if e.shapeBindings[i].td == t {
			id := e.shapeBindings[i].id
			e.lastShapeTd = t
			e.lastShapeID = id
			return id
		}
	}
	return 0
}

func (e *encState) shapeBindType(t *typeDesc, id uint32) {
	e.shapeBindings = append(e.shapeBindings, shapeBinding{td: t, id: id})
	e.lastShapeTd = t
	e.lastShapeID = id
}

// shapeDeclareEnc reserves the next sequential wire ID and returns
// it. Caller emits the keys on the wire; this side only tracks the
// count to keep IDs aligned with the decoder.
func (e *encState) shapeDeclareEnc() uint32 {
	e.shapeCount++
	return e.shapeCount
}

// pairLookup reports whether the top-1 predicted successor of prev
// is curr. The wire emits a rank byte after tagStatePair that is
// always 0 in the top-1 design — callers hand-write the literal
// instead of consuming a rank return value.
//
//go:nosplit
func (e *encState) pairLookup(prev, curr uint32) bool {
	if int(prev) >= len(e.pairPred) {
		return false
	}
	return e.pairPred[prev] == curr+1
}

// pairEnsure grows the predictor slice so prev is a valid index. New
// slots default to pairPredEmpty (zero) via the runtime's append
// zero-fill — no extra initialisation needed.
//
//go:nosplit
func (e *encState) pairEnsure(prev uint32) {
	for uint32(len(e.pairPred)) <= prev {
		e.pairPred = append(e.pairPred, pairPredEmpty)
	}
}

// pairRecord installs curr as the top-1 successor of prev. Always
// overwrites — the predictor remembers the most recent transition
// only.
//
//go:nosplit
func (e *encState) pairRecord(prev, curr uint32) {
	e.pairEnsure(prev)
	e.pairPred[prev] = curr + 1
}

// mruPush records id as the newest entry in the side-cache ring.
// Overwrites the slot at mruHead and advances the head. The ring is
// power-of-two sized so the modulo collapses to an AND. Caller
// guarantees id < mruEmpty (maxStateEntries cap ensures this).
//
//go:nosplit
func (e *encState) mruPush(id uint32) {
	e.mruRing[e.mruHead&mruRingMask] = uint16(id)
	e.mruHead++
}

// mruRank scans the ring from newest to oldest looking for id. If
// found at offset r from the head, r equals the current LRU chain
// rank (since every state-ref emit is recorded in the ring in
// order). Returns (rank, true) on hit, (0, false) when id is not in
// the last mruRingSize emissions — in which case the caller falls
// back to the raw state-ref encoding (chain rank is necessarily
// ≥ mruRingSize and would need a multi-byte varuint anyway).
//
// Hand-unrolled 4-way: profiling on telemetry workloads showed the
// scalar loop at ~17 % flat (top hotspot post the May 2026
// series). The unroll amortises the back-edge branch, lets the CPU
// issue 4 independent loads per iteration, and keeps the typical
// low-rank early-exit semantics. Falls back to scalar for the
// final partial iteration when mruRingSize is not a multiple of 4
// (it is at 128, but the guard keeps the function correct under
// future ring-size changes).
//
//go:nosplit
func (e *encState) mruRank(id uint32) (uint32, bool) {
	// IDs above the uint16 representable range can never appear in
	// the uint16 ring; short-circuit so an oversized id (only
	// reachable if maxStateEntries was bumped) never false-hits the
	// mruEmpty sentinel.
	if id >= uint32(mruEmpty) {
		return 0, false
	}
	target := uint16(id)
	h := e.mruHead - 1 // newest emission lives at h after this offset
	r := uint32(0)
	for ; r+3 < mruRingSize; r += 4 {
		if e.mruRing[(h-r)&mruRingMask] == target {
			return r, true
		}
		if e.mruRing[(h-r-1)&mruRingMask] == target {
			return r + 1, true
		}
		if e.mruRing[(h-r-2)&mruRingMask] == target {
			return r + 2, true
		}
		if e.mruRing[(h-r-3)&mruRingMask] == target {
			return r + 3, true
		}
	}
	for ; r < mruRingSize; r++ {
		if e.mruRing[(h-r)&mruRingMask] == target {
			return r, true
		}
	}
	return 0, false
}

// lruLinkInvalid encodes (prev=0xFFFF, next=0xFFFF) — an isolated
// slot with no neighbours. Used as the append default when growing
// the lruLink slice.
const lruLinkInvalid uint32 = 0xFFFF | (0xFFFF << 16)
const lruLink16Invalid uint32 = 0xFFFF // 16-bit sentinel masked into a uint32

//go:nosplit
func linkPrev(link uint32) uint32 { return link & 0xFFFF }

//go:nosplit
func linkNext(link uint32) uint32 { return link >> 16 }

//go:nosplit
func setLinkPrev(link, prev uint32) uint32 { return (link &^ 0xFFFF) | (prev & 0xFFFF) }

//go:nosplit
func setLinkNext(link, next uint32) uint32 { return (link & 0xFFFF) | ((next & 0xFFFF) << 16) }

// lruAddFresh inserts a brand-new ID (just assigned) at the head of
// the LRU. Caller must have ensured id == len(ids)-1 (i.e. ids assigns
// sequentially starting from 0). Also records the emit in the MRU
// ring so the rank-discovery side-cache reflects the new chain head.
func (e *encState) lruAddFresh(id uint32) {
	for uint32(len(e.lruLink)) <= id {
		e.lruLink = append(e.lruLink, lruLinkInvalid)
	}
	head := e.lruHead
	// id.prev = invalid, id.next = head
	if head == lruInvalidID {
		e.lruLink[id] = lruLinkInvalid
	} else {
		e.lruLink[id] = lruLink16Invalid | (head << 16)
		// head.prev = id
		e.lruLink[head] = setLinkPrev(e.lruLink[head], id)
	}
	e.lruHead = id
	e.mruPush(id)
}

// lruMoveFront performs the unlink+insert-at-head update of the LRU
// but skips the rank walk. Use when the caller does not need the
// rank (e.g. raw state-ref where MTF cannot win). Also records the
// emit in the MRU ring so the rank side-cache mirrors the chain
// head update.
//
//go:nosplit
func (e *encState) lruMoveFront(id uint32) {
	if e.lruHead == id {
		e.mruPush(id)
		return
	}
	link := e.lruLink[id]
	p := linkPrev(link)
	n := linkNext(link)
	// p is always valid here (id was not head). Patch p.next = n.
	e.lruLink[p] = setLinkNext(e.lruLink[p], n)
	if n != lruLink16Invalid {
		// Patch n.prev = p.
		e.lruLink[n] = setLinkPrev(e.lruLink[n], p)
	}
	// Insert id at head: id.prev=invalid, id.next=head.
	head := e.lruHead
	e.lruLink[id] = lruLink16Invalid | (head << 16)
	// head.prev = id (head is always valid here — id was in chain).
	e.lruLink[head] = setLinkPrev(e.lruLink[head], id)
	e.lruHead = id
	e.mruPush(id)
}

// lookupOrAssign returns (id, hit). On a miss a fresh entry is
// installed and (id, false) is returned; the caller is expected to
// emit an intern record. The key bytes are copied into the encState
// arena so the table is independent of the caller's buffer.
//
// Uses the flat open-addressed hash table (internTable) — a single
// memhash + a couple of cache-line loads instead of Go's
// mapaccess2_faststr (hash + bucket walk + tophash + memequal).
// The stored slot.key aliases the arena copy via unsafestr.String
// so the encoder owns the bytes and the caller's buffer can be
// reused immediately after the call.
//
// For payloads longer than the arena's per-string limit
// (internarena.MaxStringLen, 65 535 bytes), fall back to
// strings.Clone. Such oversized intern attempts are not expected
// on real workloads; the path exists so a hostile input cannot
// crash the encoder.
//
//go:nosplit
func (e *encState) lookupOrAssign(key string) (uint32, bool) {
	// Hot-path fast lookup: hash + one slot probe. Inlinable (no
	// loop, no allocs); the slow tail (collision probing, miss-
	// install, grow) is split out so the linear-probing loop does
	// not pollute the inline budget. Hit rate at slot 0 is high
	// because the table is kept under 0.5 load.
	h := maphash.String(internHashSeed, key)
	if h == 0 {
		h = 1 // reserve 0 as the empty-slot sentinel
	}
	i := h & uint64(len(e.internTable)-1)
	slot := &e.internTable[i]
	if slot.hash == h && slot.key == key {
		return slot.id, true
	}
	if slot.hash == 0 {
		// Empty first slot: direct install.
		return e.installInternSlot(slot, h, key), false
	}
	// Collision at first slot — fall to the probing loop.
	return e.lookupOrAssignSlow(h, key, i)
}

// lookupOrAssignSlow handles the collision case: probe past startIdx
// looking for either an empty slot (install) or a matching entry
// (hit). Separated from lookupOrAssign so the inliner keeps the
// fast path tight.
func (e *encState) lookupOrAssignSlow(h uint64, key string, startIdx uint64) (uint32, bool) {
	mask := uint64(len(e.internTable) - 1)
	for i := (startIdx + 1) & mask; ; i = (i + 1) & mask {
		slot := &e.internTable[i]
		if slot.hash == 0 {
			return e.installInternSlot(slot, h, key), false
		}
		if slot.hash == h && slot.key == key {
			return slot.id, true
		}
	}
}

// installInternSlot writes a fresh entry into slot, copies the key
// into the encoder arena (so it survives the caller's buffer
// lifetime), bumps the LRU + intern counters, and grows the table
// when the load crosses 3/4. The slot pointer can be invalidated
// by the grow; callers must not touch it after this returns.
func (e *encState) installInternSlot(slot *internSlot, h uint64, key string) uint32 {
	id := e.internLoad
	var stored string
	if len(key) <= internarena.MaxStringLen {
		arenaID := e.arena.Put(key)
		stored = unsafestr.String(e.arena.Get(arenaID))
	} else {
		stored = strings.Clone(key)
	}
	slot.hash = h
	slot.key = stored
	slot.id = id
	e.internLoad++
	e.lruAddFresh(id)
	// Grow at 3/4 load, not 1/2. A denser table is smaller (better cache)
	// and rehashes less often; with the well-distributed maphash the longer
	// linear-probe chains cost less than the cache + rehash savings.
	// Measured -12.6% encode on the large-payload Archive profile (thousands
	// of interned strings), neutral on small/medium payloads, wire unchanged.
	if e.internLoad*4 >= uint32(len(e.internTable))*3 {
		e.internTableGrow()
	}
	return id
}

// internTableGrow doubles the flat hash table and rehashes every
// occupied slot. Called from installInternSlot when the load factor
// reaches 3/4. Amortised insert stays O(1); the denser table trades
// slightly longer probe chains for fewer rehashes and a smaller cache
// footprint (a net encode win on large, intern-heavy payloads).
func (e *encState) internTableGrow() {
	old := e.internTable
	newSize := len(old) * 2
	if newSize == 0 {
		newSize = internTableInitSize
	}
	e.internTable = make([]internSlot, newSize)
	mask := uint64(newSize - 1)
	for i := range old {
		if old[i].hash == 0 {
			continue
		}
		for j := old[i].hash & mask; ; j = (j + 1) & mask {
			if e.internTable[j].hash == 0 {
				e.internTable[j] = old[i]
				break
			}
		}
	}
}

// decShape is the decoder-side mirror of an encShape. keyIDs holds
// the intern IDs of the field names (informational; same ordering as
// names). names is the resolved Go-string for each field in
// declaration order — used to dispatch values to struct fields when
// the shape is re-used.
type decShape struct {
	keyIDs []uint32
	names  []string
}

// decColShape is the decoder-side descriptor for a columnar struct shape
// (tagColStruct). Parallel to encState's colShapeNames/colShapeKinds entries.
type decColShape struct {
	names []string
	kinds []colKind
}

type decState struct {
	// Hot scalars first — touched on every tagState* read. Packing
	// them with the mruRing/head update keeps the per-emit footprint
	// in the first cache line.
	lastID  uint32
	lruHead uint32
	mruHead uint32
	_       uint32 // align mruRing on 8-byte boundary

	// mruRing mirrors the encoder's side-cache: the last mruRingSize
	// state-ref ids in emission order, stored as uint16 (the id
	// space is < 2^14 by encoder cap). For tagStateMTF the wire
	// carries rank — direct index into the ring resolves the id in
	// O(1) (mruRing[(mruHead-1-rank)&mask]) instead of walking the
	// LRU chain. Pure decoder-side optimization; the wire format is
	// unchanged.
	mruRing [mruRingSize]uint16

	// values holds the decoded byte slices indexed by intern id —
	// each entry aliases the wire buffer (zero-copy).
	values [][]byte

	// stringValues caches a heap-allocated `string` copy per intern
	// record. Populated once at append time (one `string(b)` alloc
	// per first occurrence); subsequent state-ref / MTF / pair /
	// repeat reads return the cached string from Decoder.ReadString
	// without paying another `string(b)` copy each time. On dense
	// payloads with repeated values (telemetry, archive,
	// LargePayload) this collapses N-1 of every N reads to zero
	// alloc.
	stringValues []string

	// LRU mirror of encState's. Decoder maintains the same MTF chain
	// the encoder did so tagStateMTF + rank resolves to the same ID.
	// See encState.lruLink for the packing layout.
	lruLink []uint32

	// Pair predictor mirror. See encState.pairPred for the storage
	// layout (succ+1 packed into a uint32, 0 = empty). The decoder
	// only ever reads rank 0; any rank ≥ pairPredK on the wire is
	// rejected upstream as malformed.
	pairPred []uint32

	// Shape table mirror. shapes[i] is the shape with wire-ID i+1.
	shapes []decShape

	// Columnar shape table (tagColStruct). colShapes[i] is the columnar
	// shape with wire-ID i+1. Parallel to encState's colShapeNames/colShapeKinds.
	colShapes      []decColShape
	colScratchI64  []int64
	colScratchU64  []uint64
	colScratchF64  []float64
	colScratchBool []bool

	// colLenScratch is reused storage for the column-length index parsed from
	// a FlagColIndex columnar payload (one uint32 byte-length per column).
	colLenScratch []uint32
}

func newDecState() *decState {
	d := &decState{
		values:       make([][]byte, 0, 64),
		stringValues: make([]string, 0, 64),
		lruHead:      lruInvalidID,
		lastID:       lruInvalidID,
	}
	for i := range d.mruRing {
		d.mruRing[i] = mruEmpty
	}
	return d
}

func (d *decState) reset() {
	// Symmetric to encState.reset's water-mark policy: shrink
	// excess capacity back when a prior cycle grew past the soft
	// cap, otherwise reuse the existing slices in place.
	if cap(d.values) > maxRetainedIDs {
		d.values = nil
		d.stringValues = nil
	} else {
		d.values = d.values[:0]
		d.stringValues = d.stringValues[:0]
	}
	d.lastID = lruInvalidID
	d.lruHead = lruInvalidID
	if cap(d.lruLink) > maxRetainedLRUCap {
		d.lruLink = nil
	} else {
		d.lruLink = d.lruLink[:0]
	}
	if cap(d.pairPred) > maxRetainedPairCap {
		d.pairPred = nil
	} else {
		clear(d.pairPred)
	}
	if cap(d.shapes) > maxRetainedShapeCap {
		d.shapes = nil
	} else {
		d.shapes = d.shapes[:0]
	}
	if cap(d.colShapes) > maxRetainedShapeCap {
		d.colShapes = nil
	} else {
		d.colShapes = d.colShapes[:0]
	}
	if cap(d.colScratchI64) > maxRetainedIDs {
		d.colScratchI64, d.colScratchU64, d.colScratchF64, d.colScratchBool = nil, nil, nil, nil
	}
	if cap(d.colLenScratch) > maxRetainedIDs {
		d.colLenScratch = nil
	}
	for i := range d.mruRing {
		d.mruRing[i] = mruEmpty
	}
	d.mruHead = 0
}

// pairAtRank returns the predicted successor of prev. With top-1
// storage the only valid wire rank is 0; any rank ≥ pairPredK was
// already rejected upstream. ok=false marks an empty slot.
//
//go:nosplit
func (d *decState) pairAtRank(prev uint32, rank uint8) (uint32, bool) {
	if int(prev) >= len(d.pairPred) || rank != 0 {
		return 0, false
	}
	v := d.pairPred[prev]
	if v == pairPredEmpty {
		return 0, false
	}
	return v - 1, true
}

//go:nosplit
func (d *decState) pairEnsure(prev uint32) {
	for uint32(len(d.pairPred)) <= prev {
		d.pairPred = append(d.pairPred, pairPredEmpty)
	}
}

// pairRecord mirrors encState.pairRecord exactly. Overwrites the
// stored successor — top-1 keeps only the most recent transition.
//
//go:nosplit
func (d *decState) pairRecord(prev, curr uint32) {
	d.pairEnsure(prev)
	d.pairPred[prev] = curr + 1
}

// shapeDeclare appends a new shape with the next sequential wire ID
// and returns a pointer to its slot. The wire ID equals
// len(d.shapes) after the append; callers do not need it returned
// because the encoder hands shape IDs out in the same order.
func (d *decState) shapeDeclare() *decShape {
	d.shapes = append(d.shapes, decShape{})
	return &d.shapes[len(d.shapes)-1]
}

// shapeLookup returns the shape with the given wire ID (≥ 1). nil
// means an unknown ID — the stream is malformed.
func (d *decState) shapeLookup(id uint32) *decShape {
	if id == 0 || id > uint32(len(d.shapes)) {
		return nil
	}
	return &d.shapes[id-1]
}

// mruPush records id as the newest entry in the decoder side-cache.
// Mirrors encState.mruPush so the ring sees every emit the encoder
// recorded. id must fit in uint16 (id space is < 2^14 by encoder
// cap).
//
//go:nosplit
func (d *decState) mruPush(id uint32) {
	d.mruRing[d.mruHead&mruRingMask] = uint16(id)
	d.mruHead++
}

// mruIDAtRank returns the id stored rank positions back from the head
// of the ring. Provided the encoder emitted tagStateMTF only for
// ranks ≤ mruRingSize-1 (which is the only range where MTF beats raw
// for the common 2-byte id wire form), this resolves the id in O(1)
// without walking the LRU chain.
//
//go:nosplit
func (d *decState) mruIDAtRank(rank uint32) (uint32, bool) {
	if rank >= mruRingSize {
		return 0, false
	}
	id := d.mruRing[(d.mruHead-1-rank)&mruRingMask]
	if id == mruEmpty {
		return 0, false
	}
	return uint32(id), true
}

func (d *decState) lruAddFresh(id uint32) {
	for uint32(len(d.lruLink)) <= id {
		d.lruLink = append(d.lruLink, lruLinkInvalid)
	}
	head := d.lruHead
	if head == lruInvalidID {
		d.lruLink[id] = lruLinkInvalid
	} else {
		d.lruLink[id] = lruLink16Invalid | (head << 16)
		d.lruLink[head] = setLinkPrev(d.lruLink[head], id)
	}
	d.lruHead = id
	d.mruPush(id)
}

// lruIDAtRank returns the ID currently at the given MTF rank (head
// = 0) by walking the linked-list chain. Used as a fallback when the
// MRU ring side-cache misses (rank ≥ mruRingSize, which the encoder
// never emits today but the decoder must still handle for forward
// compatibility with larger ring sizes on the encoder side).
func (d *decState) lruIDAtRank(rank uint32) (uint32, bool) {
	cur := d.lruHead
	for range rank {
		if cur == lruInvalidID {
			return 0, false
		}
		cur = linkNext(d.lruLink[cur])
		if cur == lruLink16Invalid {
			cur = lruInvalidID
		}
	}
	if cur == lruInvalidID {
		return 0, false
	}
	return cur, true
}

func (d *decState) lruMoveToFront(id uint32) {
	if d.lruHead == id {
		d.mruPush(id)
		return
	}
	link := d.lruLink[id]
	p := linkPrev(link)
	n := linkNext(link)
	d.lruLink[p] = setLinkNext(d.lruLink[p], n)
	if n != lruLink16Invalid {
		d.lruLink[n] = setLinkPrev(d.lruLink[n], p)
	}
	head := d.lruHead
	d.lruLink[id] = lruLink16Invalid | (head << 16)
	d.lruLink[head] = setLinkPrev(d.lruLink[head], id)
	d.lruHead = id
	d.mruPush(id)
}

// append registers a fresh intern record with the decoder. b
// aliases the wire buffer; the cached string slot is left empty so
// the first ReadString of this record pays the string(b) copy
// exactly once, and every later state-ref / MTF / pair / repeat
// read returns the cached value without alloc.
//
// Eager materialisation would punish single-shot decodes (Config-
// shaped workloads, ~10 distinct interns, each read once) where
// the cache slot never gets re-read; lazy population matches what
// the old direct-string(b) path did on first sight and adds zero
// alloc on subsequent reads.
func (d *decState) append(b []byte) uint32 {
	id := uint32(len(d.values))
	d.values = append(d.values, b)
	d.stringValues = append(d.stringValues, "")
	d.lruAddFresh(id)
	return id
}

func (d *decState) get(id uint32) ([]byte, bool) {
	if id >= uint32(len(d.values)) {
		return nil, false
	}
	return d.values[id], true
}

// getString returns the cached string copy of the intern record at
// id, populating the slot on first call. Empty interned bytes
// resolve to "" without an extra alloc — the `len(b) == 0` branch
// short-circuits before the materialisation. Used on the state-ref
// / MTF / pair / repeat decode paths so ReadString skips the
// string(b) heap copy after the first sight.
//
//go:nosplit
func (d *decState) getString(id uint32) (string, bool) {
	if id >= uint32(len(d.stringValues)) {
		return "", false
	}
	s := d.stringValues[id]
	if s != "" {
		return s, true
	}
	b := d.values[id]
	if len(b) == 0 {
		return "", true
	}
	s = string(b)
	d.stringValues[id] = s
	return s, true
}
