package qdf

import (
	"reflect"
	"unsafe"
)

// keyToken returns a string usable as a Go map key that uniquely identifies the
// element's key field value WITHOUT allocating. elemP points at an element of a
// keyed struct type td; td.keyOff/td.keyDesc locate the key field. Delegates to
// keyTokenAt over the key field pointer.
func keyToken(td *typeDesc, elemP unsafe.Pointer) string {
	return keyTokenAt(td.keyDesc, unsafe.Add(elemP, td.keyOff))
}

// keyTokenAt returns the token for a key value of descriptor kd located at kp.
//   - string key: the key string itself (its content is the identity).
//   - scalar / [N]byte key: unsafe.String over the key's raw bytes. Valid only
//     while the backing value is alive — true for the duration of a single
//     Diff/Apply call. Safe because these kinds are gap-free (no padding within
//     the key value), so the bytes ARE the whole value.
//   - exotic comparable key (a comparable struct): the allocating reflect
//     fallback (rare; the keyed path is gated to the kinds above elsewhere, so
//     this branch is defensive).
func keyTokenAt(kd *typeDesc, kp unsafe.Pointer) string {
	switch kd.kind {
	case reflect.String:
		return *(*string)(kp)
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return unsafe.String((*byte)(kp), int(kd.rType.Size()))
	case reflect.Array:
		if kd.rType.Elem().Kind() == reflect.Uint8 {
			return unsafe.String((*byte)(kp), kd.rType.Len())
		}
		return keyTokenReflect(kd, kp)
	default:
		return keyTokenReflect(kd, kp)
	}
}

// keyedLinearMax is the element count below which keyed matching uses a linear
// scan (no map allocation). O(n^2) but n small; cheaper than a map's hashing
// plus warm-up allocation, and fully alloc-free.
const keyedLinearMax = 32

// keyTokenable reports whether the key kind is one keyTokenAt handles without
// the allocating reflect fallback (so the keyed path is worth taking).
func keyTokenable(kd *typeDesc) bool {
	switch kd.kind {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return true
	case reflect.Array:
		return kd.rType.Elem().Kind() == reflect.Uint8
	default:
		return false
	}
}

func diffKeyedSlice(enc *Encoder, td, elem *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	oh := (*sliceHeader)(oldP)
	nh := (*sliceHeader)(newP)
	stride := td.rType.Elem().Size()
	oldKeyAt := func(i int) string { return keyToken(elem, unsafe.Add(oh.Data, uintptr(i)*stride)) }
	newKeyAt := func(i int) string { return keyToken(elem, unsafe.Add(nh.Data, uintptr(i)*stride)) }

	lookup, dup, release := buildKeyLookup(&enc.keyIdx, &enc.keyIdxBusy, oldKeyAt, oh.Len)
	defer release()
	if dup || hasDupNewKeys(newKeyAt, nh.Len) {
		// Ambiguous identity → positional fallback. diffValue already wrote opMerge;
		// diffSlice writes its own tagSlicePatch body, so apply dispatches correctly.
		return diffSlice(enc, td, oldP, newP, depth)
	}

	orderChanged := oh.Len != nh.Len
	if !orderChanged {
		for i := range nh.Len {
			if oldKeyAt(i) != newKeyAt(i) {
				orderChanged = true
				break
			}
		}
	}

	enc.buf = append(enc.buf, tagKeyedSlicePatch)
	flags := byte(0)
	if orderChanged {
		flags |= flagKeyedOrderChanged
	}
	enc.buf = append(enc.buf, flags)

	if orderChanged {
		enc.buf = appendUvarint(enc.buf, uint64(nh.Len))
		for i := range nh.Len {
			nP := unsafe.Add(nh.Data, uintptr(i)*stride)
			if err := elem.keyDesc.encode(enc, unsafe.Add(nP, elem.keyOff)); err != nil {
				return err
			}
		}
	}

	// Count ops, then emit (so nOps is known before the entries).
	nOps := 0
	for i := range nh.Len {
		nP := unsafe.Add(nh.Data, uintptr(i)*stride)
		oi, ok := lookupGet(lookup, oldKeyAt, oh.Len, newKeyAt(i))
		if ok && equalValue(elem, unsafe.Add(oh.Data, uintptr(oi)*stride), nP, depth) {
			continue
		}
		nOps++
	}
	enc.buf = appendUvarint(enc.buf, uint64(nOps))
	for i := range nh.Len {
		nP := unsafe.Add(nh.Data, uintptr(i)*stride)
		k := newKeyAt(i)
		oi, ok := lookupGet(lookup, oldKeyAt, oh.Len, k)
		if ok {
			oP := unsafe.Add(oh.Data, uintptr(oi)*stride)
			if equalValue(elem, oP, nP, depth) {
				continue
			}
			if err := elem.keyDesc.encode(enc, unsafe.Add(nP, elem.keyOff)); err != nil {
				return err
			}
			if err := diffValue(enc, elem, oP, nP, depth+1); err != nil {
				return err
			}
		} else {
			if err := elem.keyDesc.encode(enc, unsafe.Add(nP, elem.keyOff)); err != nil {
				return err
			}
			if err := writeReplace(enc, elem, nP); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyKeyedSlice reads a tagKeyedSlicePatch body and reconciles the base slice
// by element identity (the key field), mirroring diffKeyedSlice on the encode
// side. Element copies use typed reflect Set so the compiler emits GC write
// barriers (raw memmove would not); the new slice is built with reflect.MakeSlice.
func applyKeyedSlice(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if depth > maxDeltaDepth {
		return ErrInvalidPatch
	}
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagKeyedSlicePatch {
		return ErrInvalidPatch
	}
	dec.i++
	if dec.i >= len(dec.buf) {
		return ErrInvalidPatch
	}
	flags := dec.buf[dec.i]
	dec.i++

	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	if elem == nil || !elem.keyed || elem.keyDesc == nil {
		return ErrInvalidPatch
	}
	elemType := td.rType.Elem()
	stride := elemType.Size()
	bh := (*sliceHeader)(baseP)
	baseKeyAt := func(i int) string { return keyToken(elem, unsafe.Add(bh.Data, uintptr(i)*stride)) }

	if flags&flagKeyedOrderChanged == 0 {
		// Value-only updates on the same key sequence; apply each op in place.
		lookup, _, release := buildKeyLookup(&dec.keyIdx, &dec.keyIdxBusy, baseKeyAt, bh.Len)
		defer release()
		nOps, k := readUvarint(dec.buf[dec.i:])
		if k <= 0 || nOps > uint64(len(dec.buf)-dec.i) {
			return ErrInvalidPatch
		}
		dec.i += k
		keyHold := reflect.New(elem.keyDesc.rType).Elem()
		for range nOps {
			if err := elem.keyDesc.decode(dec, keyHold.Addr().UnsafePointer()); err != nil {
				return err
			}
			tok := keyTokenAt(elem.keyDesc, keyHold.Addr().UnsafePointer())
			oi, ok := lookupGet(lookup, baseKeyAt, bh.Len, tok)
			if !ok {
				return ErrInvalidPatch // op for a key not in base (divergent base)
			}
			if err := applyValue(dec, elem, unsafe.Add(bh.Data, uintptr(oi)*stride), depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	// orderChanged: read new key order, build base lookup, construct the new slice.
	newLen64, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k
	newLen := int(newLen64)
	if newLen < 0 || uint64(newLen) > uint64(len(dec.buf)-dec.i) {
		return ErrInvalidPatch
	}
	// Decode all order keys into ONE contiguous typed buffer so each token can
	// alias its stable element (no per-key string copy). keysV stays live for the
	// function, keeping the key bytes (and any string-key heap content) reachable.
	order := make([]string, newLen)
	keysV := reflect.MakeSlice(reflect.SliceOf(elem.keyDesc.rType), newLen, newLen)
	keysBase := keysV.UnsafePointer()
	ksize := elem.keyDesc.rType.Size()
	for i := range newLen {
		kp := unsafe.Add(keysBase, uintptr(i)*ksize)
		if err := elem.keyDesc.decode(dec, kp); err != nil {
			return err
		}
		order[i] = keyTokenAt(elem.keyDesc, kp) // aliases keysV (stable) — no copy
	}

	lookup, _, release := buildKeyLookup(&dec.keyIdx, &dec.keyIdxBusy, baseKeyAt, bh.Len)
	defer release()

	nv := reflect.MakeSlice(td.rType, newLen, newLen)
	nb := nv.UnsafePointer()
	orderIdx := make(map[string]int, newLen)
	for i, tok := range order {
		if _, dup := orderIdx[tok]; dup {
			// Duplicate identity in the decoded order is ambiguous; the diff side
			// never emits one (hasDupNewKeys routes to positional), so reject a
			// hostile/divergent patch rather than silently mis-assigning slots.
			return ErrInvalidPatch
		}
		orderIdx[tok] = i
	}
	filled := make([]bool, newLen)
	// Copy unchanged base elements into their new slots (GC-safe typed Set).
	for i := range newLen {
		oi, ok := lookupGet(lookup, baseKeyAt, bh.Len, order[i])
		if ok {
			dst := reflect.NewAt(elemType, unsafe.Add(nb, uintptr(i)*stride)).Elem()
			src := reflect.NewAt(elemType, unsafe.Add(bh.Data, uintptr(oi)*stride)).Elem()
			dst.Set(src)
			filled[i] = true
		}
	}
	// Apply ops into their slots by key.
	nOps, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nOps > uint64(len(dec.buf)-dec.i) {
		return ErrInvalidPatch
	}
	dec.i += k2
	keyHold2 := reflect.New(elem.keyDesc.rType).Elem()
	for range nOps {
		if err := elem.keyDesc.decode(dec, keyHold2.Addr().UnsafePointer()); err != nil {
			return err
		}
		tok := keyTokenAt(elem.keyDesc, keyHold2.Addr().UnsafePointer())
		ni, ok := orderIdx[tok]
		if !ok {
			return ErrInvalidPatch
		}
		slot := unsafe.Add(nb, uintptr(ni)*stride)
		if err := applyValue(dec, elem, slot, depth+1); err != nil {
			return err
		}
		filled[ni] = true
	}
	for i := range newLen {
		if !filled[i] {
			return ErrInvalidPatch // a key with neither a base match nor an op
		}
	}
	reflect.NewAt(td.rType, baseP).Elem().Set(nv)
	return nil
}

// keyLookup chooses linear (small n, no map) vs a built key→index map. When a
// map is used it is carried in m so lookupGet reads the exact map this build
// produced, immune to a nested keyed-slice frame reusing the shared scratch.
type keyLookup struct {
	useMap bool
	m      map[string]int
}

// buildKeyLookup builds an old/base key→index lookup. For n > keyedLinearMax it
// borrows the caller's reusable scratch map (enc.keyIdx / dec.keyIdx) when it is
// free; if a parent keyed-slice frame already holds it (re-entrancy via a nested
// keyed slice), it allocates a fresh local map so the parent's lookup is never
// clobbered. release() returns the borrow (no-op for the linear / nested cases).
func buildKeyLookup(m *map[string]int, busy *bool, keyAt func(int) string, n int) (lk keyLookup, dup bool, release func()) {
	noop := func() {}
	if n <= keyedLinearMax {
		for i := range n {
			ki := keyAt(i)
			for j := range i {
				if keyAt(j) == ki {
					return keyLookup{}, true, noop
				}
			}
		}
		return keyLookup{useMap: false}, false, noop
	}
	var lm map[string]int
	release = noop
	if !*busy {
		*busy = true
		if *m == nil {
			*m = make(map[string]int, n)
		} else {
			clear(*m)
		}
		lm = *m
		release = func() { *busy = false }
	} else {
		lm = make(map[string]int, n)
	}
	for i := range n {
		ki := keyAt(i)
		if _, exists := lm[ki]; exists {
			return keyLookup{useMap: true, m: lm}, true, release
		}
		lm[ki] = i
	}
	return keyLookup{useMap: true, m: lm}, false, release
}

func lookupGet(l keyLookup, keyAt func(int) string, n int, key string) (int, bool) {
	if l.useMap {
		i, ok := l.m[key]
		return i, ok
	}
	for i := range n {
		if keyAt(i) == key {
			return i, true
		}
	}
	return 0, false
}

func hasDupNewKeys(keyAt func(int) string, n int) bool {
	if n <= keyedLinearMax {
		for i := range n {
			ki := keyAt(i)
			for j := range i {
				if keyAt(j) == ki {
					return true
				}
			}
		}
		return false
	}
	seen := make(map[string]struct{}, n)
	for i := range n {
		k := keyAt(i)
		if _, ok := seen[k]; ok {
			return true
		}
		seen[k] = struct{}{}
	}
	return false
}

// keyTokenReflect is the allocating fallback for exotic comparable key types
// (a comparable struct). Rare and off the hot path; the keyed diff/apply path is
// gated (keyTokenable, a later task) to the non-fallback kinds, so a struct key
// falls back to positional diff rather than relying on this token.
func keyTokenReflect(kd *typeDesc, kp unsafe.Pointer) string {
	return reflect.NewAt(kd.rType, kp).Elem().String()
}
