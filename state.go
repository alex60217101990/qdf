package qdf

import "strings"

// Intern table backing Dense mode. The encoder maintains a string→ID
// map; the decoder maintains the matching ID-ordered list of byte
// slices. IDs are assigned in encode order starting at 0. There is no
// eviction — callers bound lifetime by resetting or recycling.

type encState struct {
	ids map[string]uint32

	// lastID and lastValid track the most-recently-emitted state-ref ID
	// for the tagStateRepeat Markov-0 predictor: a state-ref equal to
	// the immediately preceding emission is encoded as a single byte.
	lastID    uint32
	lastValid bool

	// Move-to-front LRU over intern IDs. lruHead is the ID at rank 0
	// (most recently emitted state-ref or freshly interned). prev/next
	// are indexed by ID. Walking head -> next chain gives rank. After
	// each state-ref / MTF / intern emit, the touched ID is unlinked
	// and re-inserted at head.
	lruHead uint32
	lruPrev []uint32
	lruNext []uint32
}

const lruInvalidID = ^uint32(0)

func newEncState() *encState {
	return &encState{
		ids:     make(map[string]uint32, 64),
		lruHead: lruInvalidID,
	}
}

func (e *encState) reset() {
	clear(e.ids)
	e.lastID = 0
	e.lastValid = false
	e.lruHead = lruInvalidID
	e.lruPrev = e.lruPrev[:0]
	e.lruNext = e.lruNext[:0]
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

// lookupOrAssign returns (id, hit). On a miss a fresh entry is installed
// and (id, false) is returned; the caller is expected to emit an intern
// record. The key is copied so the table is independent of the caller's
// buffer.
func (e *encState) lookupOrAssign(key string) (uint32, bool) {
	if id, ok := e.ids[key]; ok {
		return id, true
	}
	id := uint32(len(e.ids))
	// strings.Clone allocates exactly one immutable backing array and
	// returns a string header that aliases it. The previous
	// `string(copyToBytes(key))` did the same work but paid for two
	// allocations: a []byte buffer then a string copy of it.
	e.ids[strings.Clone(key)] = id
	e.lruAddFresh(id)
	return id, false
}

type decState struct {
	values [][]byte

	// Mirror of encState's lastID/lastValid. tagStateRepeat resolves to
	// values[lastID] without consuming a varuint.
	lastID    uint32
	lastValid bool

	// LRU mirror of encState's. Decoder maintains the same MTF chain
	// the encoder did so tagStateMTF + rank resolves to the same ID.
	lruHead uint32
	lruPrev []uint32
	lruNext []uint32
}

func newDecState() *decState {
	return &decState{
		values:  make([][]byte, 0, 64),
		lruHead: lruInvalidID,
	}
}

func (d *decState) reset() {
	d.values = d.values[:0]
	d.lastID = 0
	d.lastValid = false
	d.lruHead = lruInvalidID
	d.lruPrev = d.lruPrev[:0]
	d.lruNext = d.lruNext[:0]
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
	for i := uint32(0); i < rank; i++ {
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
