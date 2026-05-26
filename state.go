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

// pairPredK is the ring size of the per-prev-ID successor predictor
// backing tagStatePair. K=4 keeps the linear scan cheap and the rank
// always fits in a single varuint byte (ranks 0..3).
const pairPredK = 4

// pairRing remembers up to pairPredK most-recent successor IDs that
// followed a particular prev ID. items[0] is most-recent; entries past
// n are unused. Updates are O(K) (small linear shift); lookup is O(K).
// The ring is stored contiguously in a slice indexed by prev ID so
// the predictor adds zero per-call heap allocations once the encoder
// has warmed up (Reset only zeros the `n` field, retaining capacity).
type pairRing struct {
	items [pairPredK]uint32
	n     uint8
}

// encShape stores a struct/map shape known by both encoder and decoder.
// keys are the wire-bytes of the field names (alias into encoder buffer
// is fine for the encoder side; decoder copies into shapeKeyBuf).
type encShape struct {
	// keys is the ordered list of intern-IDs of the field names. We
	// compare by the concatenated intern-IDs so two structs with the
	// SAME ordered key set share a shape regardless of identity.
	keys []uint32
}

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

	// Pair predictor: for each "prev" state-ref ID, remember the last
	// pairPredK successors. A hit emits tagStatePair + varuint(rank).
	// Slice indexed by prev ID. A ring with n==0 is uninitialised;
	// once we add the first successor n becomes ≥1 and never returns
	// to 0 until Reset, so n is a sufficient "has-entry" predicate
	// without a parallel bool slice.
	pairPred []pairRing

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
	shapes        []encShape
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

func (e *encState) reset() {
	// Order matters: clear the map before resetting the arena. Map
	// keys alias arena bytes; the arena Reset rolls cursors back
	// and the next Put overwrites the prior payload area, so any
	// surviving aliased key would read garbage.
	clear(e.ids)
	e.arena.Reset()
	e.lastID = lruInvalidID
	e.lruHead = lruInvalidID
	e.lruPrev = e.lruPrev[:0]
	e.lruNext = e.lruNext[:0]
	// Zero only the per-prev "n" field so the ring slots and the
	// underlying slice capacity stay reusable across Reset. The hot
	// path adds zero heap allocations once the encoder has warmed.
	for i := range e.pairPred {
		e.pairPred[i].n = 0
	}
	e.shapes = e.shapes[:0]
	e.shapeBindings = e.shapeBindings[:0]
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

// shapeDeclare reserves the next sequential wire ID and returns it.
// Caller emits the keys on the wire; this side only tracks the count
// to keep IDs aligned with the decoder.
func (e *encState) shapeDeclareEnc() uint32 {
	e.shapes = append(e.shapes, encShape{})
	return uint32(len(e.shapes))
}

// pairLookup returns the rank (0..pairPredK-1) of curr in the ring of
// successors for prev, or (0, false) if not present. The ring is NOT
// updated by this call; the caller must invoke pairRecord after the
// emission decision is final.
//
//go:nosplit
func (e *encState) pairLookup(prev, curr uint32) (uint8, bool) {
	if int(prev) >= len(e.pairPred) {
		return 0, false
	}
	r := &e.pairPred[prev]
	if r.n == 0 {
		return 0, false
	}
	if r.items[0] == curr {
		return 0, true
	}
	for i := uint8(1); i < r.n; i++ {
		if r.items[i] == curr {
			return i, true
		}
	}
	return 0, false
}

// pairEnsure grows the predictor slice so prev is a valid index.
//
//go:nosplit
func (e *encState) pairEnsure(prev uint32) {
	for uint32(len(e.pairPred)) <= prev {
		e.pairPred = append(e.pairPred, pairRing{})
	}
}

// pairRecord installs curr at the head of prev's successor ring.
// Existing curr (if any) is moved up to head; new entries push older
// ones down and the oldest is evicted past pairPredK.
func (e *encState) pairRecord(prev, curr uint32) {
	e.pairEnsure(prev)
	r := &e.pairPred[prev]
	if r.n == 0 {
		r.items[0] = curr
		r.n = 1
		return
	}
	idx := uint8(255)
	for i := uint8(0); i < r.n; i++ {
		if r.items[i] == curr {
			idx = i
			break
		}
	}
	if idx == 0 {
		return
	}
	if idx == 255 {
		if r.n < pairPredK {
			r.n++
		}
		for i := r.n - 1; i > 0; i-- {
			r.items[i] = r.items[i-1]
		}
		r.items[0] = curr
		return
	}
	for i := idx; i > 0; i-- {
		r.items[i] = r.items[i-1]
	}
	r.items[0] = curr
}

// shapeAssign returns (id, hit). On a miss the encState installs a new
// shape with the next sequential ID (≥ 1) and returns (id, false). The
// keys slice is copied to detach it from the caller's storage.
func (e *encState) shapeAssign(keys []uint32) (uint32, bool) {
	// Linear scan keyed on length-then-equality. shape tables stay
	// small in practice (one entry per distinct struct type emitted),
	// so the O(N*K) cost is dwarfed by the encode budget. A hash map
	// is overkill until benches say otherwise.
	for i := range e.shapes {
		if len(e.shapes[i].keys) != len(keys) {
			continue
		}
		match := true
		for j := range keys {
			if e.shapes[i].keys[j] != keys[j] {
				match = false
				break
			}
		}
		if match {
			return uint32(i + 1), true
		}
	}
	cp := make([]uint32, len(keys))
	copy(cp, keys)
	e.shapes = append(e.shapes, encShape{keys: cp})
	return uint32(len(e.shapes)), false
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

	// Pair predictor mirror. Same shape and update rules as encState's
	// version so a tagStatePair + rank resolves to the same ID. Slice
	// storage matches the encoder side; n==0 marks an unused slot.
	pairPred []pairRing

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
	d.values = d.values[:0]
	d.lastID = lruInvalidID
	d.lruHead = lruInvalidID
	d.lruPrev = d.lruPrev[:0]
	d.lruNext = d.lruNext[:0]
	for i := range d.pairPred {
		d.pairPred[i].n = 0
	}
	d.shapes = d.shapes[:0]
}

// pairAtRank returns the ID at the given rank in prev's successor
// ring. ok=false signals a malformed stream.
//
//go:nosplit
func (d *decState) pairAtRank(prev uint32, rank uint8) (uint32, bool) {
	if int(prev) >= len(d.pairPred) {
		return 0, false
	}
	r := &d.pairPred[prev]
	if r.n == 0 || rank >= r.n {
		return 0, false
	}
	return r.items[rank], true
}

//go:nosplit
func (d *decState) pairEnsure(prev uint32) {
	for uint32(len(d.pairPred)) <= prev {
		d.pairPred = append(d.pairPred, pairRing{})
	}
}

// pairRecord mirrors encState.pairRecord exactly.
func (d *decState) pairRecord(prev, curr uint32) {
	d.pairEnsure(prev)
	r := &d.pairPred[prev]
	if r.n == 0 {
		r.items[0] = curr
		r.n = 1
		return
	}
	idx := uint8(255)
	for i := uint8(0); i < r.n; i++ {
		if r.items[i] == curr {
			idx = i
			break
		}
	}
	if idx == 0 {
		return
	}
	if idx == 255 {
		if r.n < pairPredK {
			r.n++
		}
		for i := r.n - 1; i > 0; i-- {
			r.items[i] = r.items[i-1]
		}
		r.items[0] = curr
		return
	}
	for i := idx; i > 0; i-- {
		r.items[i] = r.items[i-1]
	}
	r.items[0] = curr
}

// shapeDeclare appends a new shape with the next sequential wire ID
// and returns that ID (≥ 1). Caller must populate KeyIDs in the
// returned slot.
func (d *decState) shapeDeclare() (*decShape, uint32) {
	d.shapes = append(d.shapes, decShape{})
	return &d.shapes[len(d.shapes)-1], uint32(len(d.shapes))
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
