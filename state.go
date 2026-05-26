package qdf

import (
	"strings"

	"github.com/alex60217101990/qdf/internal/internarena"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

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

type encState struct {
	ids map[string]uint32

	// arena owns the byte storage that backs every intern key the
	// encoder allocates. Replaces per-key strings.Clone with a
	// bump-pointer slab — see internal/internarena. The map keys
	// stored above are aliases into arena chunks; on Reset we clear
	// the map first, then the arena, so an aliased key is never
	// read past its arena's life.
	//
	// Inlined by value (not *Arena) so encState creation does not
	// pay a separate heap allocation for the Arena struct itself.
	// The slab (chunks[0]) is still lazily allocated on first Put,
	// so an OptSpeed / OptQPack encoder that never enters Dense
	// stays slab-free.
	arena internarena.Arena

	// lastID tracks the most-recently-emitted state-ref ID for the
	// tagStateRepeat Markov-0 predictor. lruInvalidID signals "no
	// previous emission" (e.g. fresh stream or after an inline
	// scalar emission that broke the chain). Valid IDs are bounded
	// by maxStateEntries, well below the sentinel, so an integer
	// compare with lastID is sufficient.
	lastID uint32

	// Move-to-front LRU over intern IDs. lruHead is the ID at rank 0
	// (most recently emitted state-ref or freshly interned). prev/next
	// are indexed by ID. Walking head -> next chain gives rank. After
	// each state-ref / MTF / intern emit, the touched ID is unlinked
	// and re-inserted at head.
	lruHead uint32
	lruPrev []uint32
	lruNext []uint32

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
	// the hot path.
	//
	// lastShapeTd / lastShapeID memoise the most-recent successful
	// shape lookup so a run of identical-type struct emits (the
	// common "array of T" case) is a one-pointer compare per record
	// instead of a slice scan.
	//
	// shapeCount is the running total of declared shape IDs. The
	// encoder hands out IDs in sequence and only needs the count to
	// stay aligned with the decoder; we used to keep a []encShape
	// slice here too, but its only consumer was a now-removed
	// shapeAssign() and the type carried no live data once the
	// typeDesc-keyed shapeBindings replaced ordered-key matching.
	shapeCount    uint32
	shapeBindings []shapeBinding
	lastShapeTd   *typeDesc
	lastShapeID   uint32
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
	return &encState{
		ids:     make(map[string]uint32, 64),
		lruHead: lruInvalidID,
		lastID:  lruInvalidID,
	}
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
	// Map shrink. Go's runtime map cannot reduce its bucket array
	// once it has grown, so the only way to release that memory is
	// to replace the map header with a freshly-allocated one. Do it
	// only when len exceeded the threshold during the prior cycle;
	// the steady-state path stays on clear() which keeps the
	// existing buckets warm.
	//
	// Order: rebuild / clear BEFORE arena.Reset. The map keys alias
	// arena bytes; the arena Reset rolls cursors back and the next
	// Put overwrites the prior payload area, so any surviving
	// aliased key would read garbage.
	if len(e.ids) > maxRetainedIDs {
		e.ids = make(map[string]uint32, 64)
	} else {
		clear(e.ids)
	}
	e.arena.Reset() // Arena has its own watermark, see internarena.

	e.lastID = lruInvalidID
	e.lruHead = lruInvalidID
	if cap(e.lruPrev) > maxRetainedLRUCap {
		e.lruPrev = nil
		e.lruNext = nil
	} else {
		e.lruPrev = e.lruPrev[:0]
		e.lruNext = e.lruNext[:0]
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

// lruAddFresh inserts a brand-new ID (just assigned) at the head of
// the LRU. Caller must have ensured id == len(ids)-1 (i.e. ids assigns
// sequentially starting from 0).
func (e *encState) lruAddFresh(id uint32) {
	for uint32(len(e.lruPrev)) <= id {
		e.lruPrev = append(e.lruPrev, lruInvalidID)
		e.lruNext = append(e.lruNext, lruInvalidID)
	}
	e.lruPrev[id] = lruInvalidID
	e.lruNext[id] = e.lruHead
	if e.lruHead != lruInvalidID {
		e.lruPrev[e.lruHead] = id
	}
	e.lruHead = id
}

// lruMoveToFront unlinks id from its current LRU position and inserts
// it at head. Returns the rank id had BEFORE the move (0 = was at
// head). Caller must ensure id is already in the LRU.
func (e *encState) lruMoveToFront(id uint32) uint32 {
	if e.lruHead == id {
		return 0
	}
	// Walk from head to find rank.
	var rank uint32 = 0
	cur := e.lruHead
	for cur != id {
		rank++
		cur = e.lruNext[cur]
	}
	// Unlink.
	p := e.lruPrev[id]
	n := e.lruNext[id]
	e.lruNext[p] = n
	if n != lruInvalidID {
		e.lruPrev[n] = p
	}
	// Insert at head.
	e.lruPrev[id] = lruInvalidID
	e.lruNext[id] = e.lruHead
	e.lruPrev[e.lruHead] = id
	e.lruHead = id
	return rank
}

// lruMoveFront does the same unlink+insert-at-head as lruMoveToFront
// but skips the rank walk. Use when the caller does not need the
// rank (e.g. raw state-ref where MTF cannot win).
//
//go:nosplit
func (e *encState) lruMoveFront(id uint32) {
	if e.lruHead == id {
		return
	}
	p := e.lruPrev[id]
	n := e.lruNext[id]
	e.lruNext[p] = n
	if n != lruInvalidID {
		e.lruPrev[n] = p
	}
	e.lruPrev[id] = lruInvalidID
	e.lruNext[id] = e.lruHead
	e.lruPrev[e.lruHead] = id
	e.lruHead = id
}

// lookupOrAssign returns (id, hit). On a miss a fresh entry is
// installed and (id, false) is returned; the caller is expected to
// emit an intern record. The key bytes are copied into the encState
// arena so the table is independent of the caller's buffer.
//
// The map key is a string header that aliases the arena copy
// (zero-copy via unsafestr.String). Map lookups still hash the key
// bytes — there is no observable behavioural difference from
// strings.Clone, only 1 fewer allocation per miss.
//
// For payloads longer than the arena's per-string limit
// (internarena.MaxStringLen, 65 535 bytes), fall back to
// strings.Clone. Such oversized intern attempts are not expected on
// real workloads; the path exists so a hostile input cannot crash
// the encoder.
func (e *encState) lookupOrAssign(key string) (uint32, bool) {
	if id, ok := e.ids[key]; ok {
		return id, true
	}
	id := uint32(len(e.ids))
	var stored string
	if len(key) <= internarena.MaxStringLen {
		arenaID := e.arena.Put(key)
		stored = unsafestr.String(e.arena.Get(arenaID))
	} else {
		stored = strings.Clone(key)
	}
	e.ids[stored] = id
	e.lruAddFresh(id)
	return id, false
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

type decState struct {
	values [][]byte

	// Mirror of encState's lastID. lruInvalidID marks "no previous
	// state-ref emitted" (fresh stream or after an inline scalar
	// broke the chain). Valid IDs are bounded by maxStateEntries on
	// the encoder side, so the sentinel cannot collide with a real
	// table position.
	lastID uint32

	// LRU mirror of encState's. Decoder maintains the same MTF chain
	// the encoder did so tagStateMTF + rank resolves to the same ID.
	lruHead uint32
	lruPrev []uint32
	lruNext []uint32

	// Pair predictor mirror. See encState.pairPred for the storage
	// layout (succ+1 packed into a uint32, 0 = empty). The decoder
	// only ever reads rank 0; any rank ≥ pairPredK on the wire is
	// rejected upstream as malformed.
	pairPred []uint32

	// Shape table mirror. shapes[i] is the shape with wire-ID i+1.
	shapes []decShape
}

func newDecState() *decState {
	return &decState{
		values:  make([][]byte, 0, 64),
		lruHead: lruInvalidID,
		lastID:  lruInvalidID,
	}
}

func (d *decState) reset() {
	// Symmetric to encState.reset's water-mark policy: shrink
	// excess capacity back when a prior cycle grew past the soft
	// cap, otherwise reuse the existing slices in place.
	if cap(d.values) > maxRetainedIDs {
		d.values = nil
	} else {
		d.values = d.values[:0]
	}
	d.lastID = lruInvalidID
	d.lruHead = lruInvalidID
	if cap(d.lruPrev) > maxRetainedLRUCap {
		d.lruPrev = nil
		d.lruNext = nil
	} else {
		d.lruPrev = d.lruPrev[:0]
		d.lruNext = d.lruNext[:0]
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

func (d *decState) lruAddFresh(id uint32) {
	for uint32(len(d.lruPrev)) <= id {
		d.lruPrev = append(d.lruPrev, lruInvalidID)
		d.lruNext = append(d.lruNext, lruInvalidID)
	}
	d.lruPrev[id] = lruInvalidID
	d.lruNext[id] = d.lruHead
	if d.lruHead != lruInvalidID {
		d.lruPrev[d.lruHead] = id
	}
	d.lruHead = id
}

// lruIDAtRank returns the ID currently at the given MTF rank (head
// = 0). Caller must guarantee rank < current LRU size.
func (d *decState) lruIDAtRank(rank uint32) (uint32, bool) {
	cur := d.lruHead
	for range rank {
		if cur == lruInvalidID {
			return 0, false
		}
		cur = d.lruNext[cur]
	}
	if cur == lruInvalidID {
		return 0, false
	}
	return cur, true
}

func (d *decState) lruMoveToFront(id uint32) {
	if d.lruHead == id {
		return
	}
	p := d.lruPrev[id]
	n := d.lruNext[id]
	d.lruNext[p] = n
	if n != lruInvalidID {
		d.lruPrev[n] = p
	}
	d.lruPrev[id] = lruInvalidID
	d.lruNext[id] = d.lruHead
	d.lruPrev[d.lruHead] = id
	d.lruHead = id
}

func (d *decState) append(b []byte) uint32 {
	id := uint32(len(d.values))
	d.values = append(d.values, b)
	d.lruAddFresh(id)
	return id
}

func (d *decState) get(id uint32) ([]byte, bool) {
	if id >= uint32(len(d.values)) {
		return nil, false
	}
	return d.values[id], true
}
