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

	lookup, dup := buildKeyLookup(&enc.keyIdx, oldKeyAt, oh.Len)
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
		oi, ok := lookupGet(lookup, &enc.keyIdx, oldKeyAt, oh.Len, newKeyAt(i))
		if ok && equalValue(elem, unsafe.Add(oh.Data, uintptr(oi)*stride), nP, depth) {
			continue
		}
		nOps++
	}
	enc.buf = appendUvarint(enc.buf, uint64(nOps))
	for i := range nh.Len {
		nP := unsafe.Add(nh.Data, uintptr(i)*stride)
		k := newKeyAt(i)
		oi, ok := lookupGet(lookup, &enc.keyIdx, oldKeyAt, oh.Len, k)
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

// keyLookup chooses linear (small n, no map) vs a reused cleared map.
type keyLookup struct{ useMap bool }

func buildKeyLookup(m *map[string]int, keyAt func(int) string, n int) (keyLookup, bool) {
	if n <= keyedLinearMax {
		for i := range n {
			ki := keyAt(i)
			for j := range i {
				if keyAt(j) == ki {
					return keyLookup{}, true
				}
			}
		}
		return keyLookup{useMap: false}, false
	}
	if *m == nil {
		*m = make(map[string]int, n)
	} else {
		clear(*m)
	}
	for i := range n {
		ki := keyAt(i)
		if _, exists := (*m)[ki]; exists {
			return keyLookup{useMap: true}, true
		}
		(*m)[ki] = i
	}
	return keyLookup{useMap: true}, false
}

func lookupGet(l keyLookup, m *map[string]int, keyAt func(int) string, n int, key string) (int, bool) {
	if l.useMap {
		i, ok := (*m)[key]
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
