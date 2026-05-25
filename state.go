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
}

func newEncState() *encState {
	return &encState{ids: make(map[string]uint32, 64)}
}

func (e *encState) reset() {
	clear(e.ids)
	e.lastID = 0
	e.lastValid = false
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
	return id, false
}

type decState struct {
	values [][]byte

	// Mirror of encState's lastID/lastValid. tagStateRepeat resolves to
	// values[lastID] without consuming a varuint.
	lastID    uint32
	lastValid bool
}

func newDecState() *decState { return &decState{values: make([][]byte, 0, 64)} }

func (d *decState) reset() {
	d.values = d.values[:0]
	d.lastID = 0
	d.lastValid = false
}

func (d *decState) append(b []byte) uint32 {
	id := uint32(len(d.values))
	d.values = append(d.values, b)
	return id
}

func (d *decState) get(id uint32) ([]byte, bool) {
	if id >= uint32(len(d.values)) {
		return nil, false
	}
	return d.values[id], true
}

