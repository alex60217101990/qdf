package qdf

import (
	"math"
	"slices"
)

// Shared helpers for OptMapShape — map key-set interning. The reflect map
// encoder (encodeStringMapShaped in reflect_encode.go) and the generated
// concrete fast paths (maps_fast_generated.go) both reuse the encState /
// decState shape machinery through these. See
// docs/superpowers/specs/2026-06-04-map-shape-interning-design.md.

// mapStringShapeOrder emits the tagMapShape declare-or-reuse header for a
// concrete string-keyed map and returns the canonical (sorted) key order the
// caller must emit values in. Generic over the value type so the generated fast
// paths stay reflection-free.
//
// Hot reuse path: one range for the order-independent set-hash, a membership
// check per bound key (guards a set-hash collision), then the bound order — no
// key slice allocated. Declare path (first sight of a key-set): gather + sort +
// intern the keys; rare. Collision/mismatch degrades to a fresh declaration,
// never to wrong data (cf. hash-key-sampling-collision).
//
// Callers must gate on len(m) > 0, e.state != nil, OptMapShape and OptDense
// before calling.
func mapStringShapeOrder[V any](e *Encoder, m map[string]V) []string {
	// Ensure the stream header precedes the first tag. For a top-level map the
	// plain path emits it via WriteMapHeader, which this shape path bypasses.
	// Idempotent (guarded by e.headerOut).
	e.writeHeader()
	n := len(m)
	st := e.state

	if e.stateSuspended {
		// Inside a never-larger trial (diffKeyedSlice / diffColumnar): emit a
		// self-contained id-0 declaration every time and never bind/reuse a
		// map-shape id — a bound id whose declaration is thrown away with the
		// losing candidate would dangle (ErrUnknownStateID), the same leak the
		// trial suspends interning to avoid. Mirrors Encoder.StructShape under
		// suspension: advance shapeCount (the decoder registers a shape per
		// declaration it reads; the trial re-bases the counter for the discarded
		// candidate) but touch no mapShapes / lastMapShapeID binding.
		keys := make([]string, 0, n)
		for k := range m {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		st.shapeDeclareEnc()
		e.buf = append(e.buf, tagMapShape)
		e.buf = appendUvarint(e.buf, 0)
		e.buf = appendUvarint(e.buf, uint64(n))
		for _, k := range keys {
			e.WriteString(k)
		}
		return keys
	}

	// Fast path: a run of homogeneous rows reuses the previous shape. Verify
	// membership directly (len + every bound key present) — no set-hash, no
	// registry scan. Exact, so collision-proof.
	if st.lastMapShapeID != 0 && len(st.lastMapShapeKeys) == n && mapHasAll(m, st.lastMapShapeKeys) {
		e.buf = append(e.buf, tagMapShape)
		e.buf = appendUvarint(e.buf, uint64(st.lastMapShapeID))
		return st.lastMapShapeKeys
	}

	// Key-set changed: hash to find an earlier shape, else declare.
	var setHash uint64
	for k := range m {
		setHash += internKeyHash(k) // commutative: order-independent
	}
	if id, order, ok := st.mapShapeFindKeys(setHash, n); ok {
		if len(order) == n && mapHasAll(m, order) {
			st.lastMapShapeID, st.lastMapShapeKeys = id, order
			e.buf = append(e.buf, tagMapShape)
			e.buf = appendUvarint(e.buf, uint64(id))
			return order
		}
		// set-hash collision or different key-set → declare a fresh shape.
	}
	keys := make([]string, 0, n)
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	id := st.shapeDeclareEnc()
	st.mapShapeRegister(setHash, n, keys, id)
	st.lastMapShapeID, st.lastMapShapeKeys = id, keys
	e.buf = append(e.buf, tagMapShape)
	e.buf = appendUvarint(e.buf, 0)
	e.buf = appendUvarint(e.buf, uint64(n))
	for _, k := range keys {
		e.WriteString(k)
	}
	return keys
}

// mapHasAll reports whether m contains every key in keys.
func mapHasAll[V any](m map[string]V, keys []string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// decodeMapStringShapeHeader consumes a tagMapShape header (the caller has
// peeked tag == tagMapShape but not advanced) and returns the ordered key
// names for the shape. On a declaration it reads and registers the keys; on a
// reuse it looks the shape up. The caller then reads len(names) values in order.
// Shared by the reflect decodeMap branch and the generated fast paths.
func decodeMapStringShapeHeader(d *Decoder) ([]string, error) {
	d.i++ // consume tagMapShape
	shapeID, sz := readUvarint(d.buf[d.i:])
	if sz <= 0 {
		return nil, ErrInvalidLength
	}
	// Reject a shape ID that would truncate on the uint32 narrowing below: a
	// crafted id >= 2^32 must not alias a real (id mod 2^32) shape.
	if shapeID > uint64(math.MaxUint32) {
		return nil, ErrUnknownStateID
	}
	d.i += sz
	if d.state == nil {
		d.state = newDecState()
	}
	if shapeID == 0 {
		cnt64, sz := readUvarint(d.buf[d.i:])
		if sz <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += sz
		cnt := int(cnt64)
		if err := d.CheckLength(cnt, 1); err != nil {
			return nil, err
		}
		sh := d.state.shapeDeclare()
		keys := make([]string, 0, cnt)
		for range cnt {
			kb, err := d.readStringBytes()
			if err != nil {
				return nil, err
			}
			keys = append(keys, d.keyCache.Make(kb))
		}
		sh.names = keys
		return keys, nil
	}
	sh := d.state.shapeLookup(uint32(shapeID))
	if sh == nil {
		return nil, ErrUnknownStateID
	}
	return sh.names, nil
}
