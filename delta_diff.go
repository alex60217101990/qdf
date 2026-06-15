package qdf

import (
	"bytes"
	"reflect"
	"unsafe"
)

// diffValue compares the value at oldP/newP and, if changed, writes one op
// (op byte + payload). The caller has already written any preceding selector
// (field index / slice index / map key).
func diffValue(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	if depth > maxDeltaDepth {
		return ErrCycleDetected
	}
	if equalValue(td, oldP, newP, depth) {
		return nil
	}
	switch td.kind {
	case reflect.Struct:
		if len(td.fields) == 0 { // time.Time / marshaler struct: whole replace
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return diffStruct(enc, td, oldP, newP, depth)
	case reflect.Pointer:
		op, np := *(*unsafe.Pointer)(oldP), *(*unsafe.Pointer)(newP)
		if op == nil || np == nil {
			// presence change (both-nil was handled by equalValue earlier) → replace
			return writeReplace(enc, td, newP)
		}
		if td.elem != nil && td.elem.kind == reflect.Struct && len(td.elem.fields) > 0 {
			enc.buf = append(enc.buf, opMerge)
			return diffStruct(enc, td.elem, op, np, depth+1)
		}
		return writeReplace(enc, td, newP)
	case reflect.Slice:
		if td.rType.Elem().Kind() == reflect.Uint8 {
			return writeReplace(enc, td, newP) // []byte: whole replace
		}
		// A nil↔non-nil transition is not expressible by a positional slice patch
		// (apply rebuilds via MakeSlice, which is always non-nil). Fall back to a
		// whole-value replace so the field codec preserves nil-vs-empty.
		if ((*sliceHeader)(oldP).Data == nil) != ((*sliceHeader)(newP).Data == nil) {
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return diffSlice(enc, td, oldP, newP, depth)
	case reflect.Array:
		if td.rType.Elem().Kind() == reflect.Uint8 {
			return writeReplace(enc, td, newP) // [N]byte: whole replace
		}
		enc.buf = append(enc.buf, opMerge)
		return diffArray(enc, td, oldP, newP, depth)
	case reflect.Map:
		// Same nil↔non-nil concern as slices: applyMap never reconstructs a nil
		// map, so a nilness transition must be a whole-value replace.
		if (*(*unsafe.Pointer)(oldP) == nil) != (*(*unsafe.Pointer)(newP) == nil) {
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return diffMap(enc, td, oldP, newP, depth)
	default:
		// Scalars/string/[]byte and presence/nilness changes are a whole-value
		// replace; structs/slices/arrays/maps/pointers merge in their own cases above.
		return writeReplace(enc, td, newP)
	}
}

// diffSlice writes a tagSlicePatch body for a non-[]byte slice. The whole-slice
// equality short-circuit already happened in diffValue→equalValue (one SIMD
// bytes.Equal over POD backing arrays), so we only get here on a real difference.
func diffSlice(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	oh := (*sliceHeader)(oldP)
	nh := (*sliceHeader)(newP)
	stride := td.rType.Elem().Size()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	return diffElems(enc, elem, stride, oh.Data, oh.Len, nh.Data, nh.Len, depth)
}

func diffArray(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	n := td.rType.Len()
	stride := td.rType.Elem().Size()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	return diffElems(enc, elem, stride, oldP, n, newP, n, depth)
}

// diffMap writes a tagMapPatch: updated/added keys (each with a recursive op),
// then a tombstone list of deleted keys. Keys are the identity, so there is no
// positional ambiguity. Two passes (count, then emit) avoid buffer surgery.
func diffMap(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	ov := reflect.NewAt(td.rType, oldP).Elem()
	nv := reflect.NewAt(td.rType, newP).Elem()
	keyType := td.rType.Key()
	keyDesc, err := descOf(keyType)
	if err != nil {
		return err
	}
	valDesc := td.elem
	if valDesc == nil {
		valDesc, err = descOf(td.rType.Elem())
		if err != nil {
			return err
		}
	}

	// Compare map values with the SAME comparator diffValue uses (equalValue),
	// not reflect.Value.Equal: the latter compares pointer identity for *T values
	// (inconsistent with diffValue's deref) and panics on non-comparable values
	// (map/slice). Reuse two addressable buffers to avoid per-key allocation.
	oCmp := reflect.New(valDesc.rType).Elem()
	nCmp := reflect.New(valDesc.rType).Elem()
	valEqual := func(oVal, nVal reflect.Value) bool {
		oCmp.Set(oVal)
		nCmp.Set(nVal)
		return equalValue(valDesc, oCmp.Addr().UnsafePointer(), nCmp.Addr().UnsafePointer(), depth)
	}

	enc.buf = append(enc.buf, tagMapPatch)

	// Pass 1: count updates/additions.
	nUpd := 0
	for it := nv.MapRange(); it.Next(); {
		oVal := ov.MapIndex(it.Key())
		if oVal.IsValid() && valEqual(oVal, it.Value()) {
			continue
		}
		nUpd++
	}
	enc.buf = appendUvarint(enc.buf, uint64(nUpd))
	// Pass 2: emit updates/additions.
	keyBuf := reflect.New(keyType).Elem()
	for it := nv.MapRange(); it.Next(); {
		k := it.Key()
		nVal := it.Value()
		oVal := ov.MapIndex(k)
		if oVal.IsValid() && valEqual(oVal, nVal) {
			continue
		}
		keyBuf.Set(k)
		if err := keyDesc.encode(enc, keyBuf.Addr().UnsafePointer()); err != nil {
			return err
		}
		if oVal.IsValid() {
			// The skip check above ran valEqual (oVal is valid), which already set
			// oCmp/nCmp to (oVal, nVal); reuse those addressable buffers directly.
			if err := diffValue(enc, valDesc,
				oCmp.Addr().UnsafePointer(), nCmp.Addr().UnsafePointer(), depth+1); err != nil {
				return err
			}
		} else {
			// Addition: oVal is invalid, so valEqual short-circuited and nCmp is
			// stale; set it from nVal and reuse the buffer for the replace.
			nCmp.Set(nVal)
			if err := writeReplace(enc, valDesc, nCmp.Addr().UnsafePointer()); err != nil {
				return err
			}
		}
	}

	// Deletions: keys in old, absent in new.
	nDel := 0
	for it := ov.MapRange(); it.Next(); {
		if !nv.MapIndex(it.Key()).IsValid() {
			nDel++
		}
	}
	enc.buf = appendUvarint(enc.buf, uint64(nDel))
	for it := ov.MapRange(); it.Next(); {
		k := it.Key()
		if nv.MapIndex(k).IsValid() {
			continue
		}
		keyBuf.Set(k)
		if err := keyDesc.encode(enc, keyBuf.Addr().UnsafePointer()); err != nil {
			return err
		}
	}
	return nil
}

// diffElems is the shared positional element differ for slices and arrays.
func diffElems(enc *Encoder, elem *typeDesc, stride uintptr,
	oldData unsafe.Pointer, oldLen int, newData unsafe.Pointer, newLen int, depth int) error {
	minLen := min(newLen, oldLen)
	// elem.pod was precomputed at build (noPointersWalk of the element type) —
	// read the field instead of walking per call. elem is always non-nil here
	// (resolved by diffSlice/diffArray); guard defensively against an unresolved
	// descriptor.
	pod := elem != nil && elem.pod
	var entries []int
	for i := range minLen {
		oP := unsafe.Add(oldData, uintptr(i)*stride)
		nP := unsafe.Add(newData, uintptr(i)*stride)
		var same bool
		if pod {
			same = bytes.Equal(unsafe.Slice((*byte)(oP), stride), unsafe.Slice((*byte)(nP), stride))
		} else {
			same = equalValue(elem, oP, nP, depth)
		}
		if !same {
			entries = append(entries, i)
		}
	}
	for i := minLen; i < newLen; i++ {
		entries = append(entries, i)
	}

	enc.buf = append(enc.buf, tagSlicePatch)
	enc.buf = appendUvarint(enc.buf, uint64(newLen))
	enc.buf = appendUvarint(enc.buf, uint64(len(entries)))
	for _, i := range entries {
		enc.buf = appendUvarint(enc.buf, uint64(i))
		nP := unsafe.Add(newData, uintptr(i)*stride)
		if i < oldLen {
			oP := unsafe.Add(oldData, uintptr(i)*stride)
			if err := diffValue(enc, elem, oP, nP, depth+1); err != nil {
				return err
			}
		} else {
			if err := writeReplace(enc, elem, nP); err != nil { // appended element
				return err
			}
		}
	}
	return nil
}

// writeReplace writes opReplace + the whole new value via the normal codec.
func writeReplace(enc *Encoder, td *typeDesc, newP unsafe.Pointer) error {
	enc.buf = append(enc.buf, opReplace)
	return td.encode(enc, newP)
}

// diffStruct writes a tagStructPatch body with only the changed fields.
func diffStruct(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer, depth int) error {
	var changed []int
	for i := range td.fields {
		f := &td.fields[i]
		if !equalValue(f.desc, unsafe.Add(oldP, f.offset), unsafe.Add(newP, f.offset), depth) {
			changed = append(changed, i)
		}
	}
	enc.buf = append(enc.buf, tagStructPatch)
	enc.buf = appendUvarint(enc.buf, uint64(len(changed)))
	for _, i := range changed {
		f := &td.fields[i]
		enc.buf = appendUvarint(enc.buf, uint64(i))
		if err := diffValue(enc, f.desc,
			unsafe.Add(oldP, f.offset), unsafe.Add(newP, f.offset), depth+1); err != nil {
			return err
		}
	}
	return nil
}

// equalValue reports whether the value of type described by td at aP equals the
// one at bP. POD scalars reduce to a width compare; strings/[]byte to a SIMD
// memcmp (bytes.Equal); containers recurse. This is the diff walk's unchanged
// fast path — for an unchanged subtree it is the ONLY work done.
func equalValue(td *typeDesc, aP, bP unsafe.Pointer, depth int) bool {
	if depth > maxDeltaDepth {
		// Force the diff path, which hits its own cap and errors out.
		return false
	}
	switch td.kind {
	case reflect.Bool:
		return *(*bool)(aP) == *(*bool)(bP)
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		// Platform-width: 8 bytes on 64-bit, 4 on 32-bit. Reading *(*uint64)
		// here would be an OOB read on 32-bit (see slices_fast.go width gating).
		return *(*uint)(aP) == *(*uint)(bP)
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return *(*uint64)(aP) == *(*uint64)(bP)
	case reflect.Int8, reflect.Uint8:
		return *(*uint8)(aP) == *(*uint8)(bP)
	case reflect.Int16, reflect.Uint16:
		return *(*uint16)(aP) == *(*uint16)(bP)
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return *(*uint32)(aP) == *(*uint32)(bP)
	case reflect.String:
		return *(*string)(aP) == *(*string)(bP)
	case reflect.Slice:
		// A nil slice and an empty-non-nil slice both have len 0 but the field
		// codec preserves the distinction; treat them as unequal so diffValue
		// emits an op (a whole-value opReplace via its nilness check).
		if ((*sliceHeader)(aP).Data == nil) != ((*sliceHeader)(bP).Data == nil) {
			return false
		}
		if td.rType.Elem().Kind() == reflect.Uint8 {
			return bytes.Equal(*(*[]byte)(aP), *(*[]byte)(bP))
		}
		return equalSliceEV(td, aP, bP, depth)
	case reflect.Array:
		return equalArrayEV(td, aP, bP, depth)
	case reflect.Struct:
		// time.Time and custom-marshaler structs have no td.fields. Use
		// reflect.DeepEqual, NOT reflect.Value.Equal, which panics when the struct
		// holds a non-comparable field (slice/map/func) — a real crash for a
		// Marshaler type with e.g. a []int field. Rare path; correctness over speed.
		if len(td.fields) == 0 {
			av := reflect.NewAt(td.rType, aP).Elem()
			bv := reflect.NewAt(td.rType, bP).Elem()
			// Fast path: a comparable fields-less struct (time.Time — common, on the
			// hot path for any timestamped value — and comparable Marshaler structs)
			// compares with reflect.Value.Equal: no .Interface() boxing, no DeepEqual
			// reflection. Only a NON-comparable Marshaler field (slice/map/func), where
			// Value.Equal would panic, falls back to DeepEqual.
			if td.rType.Comparable() {
				return av.Equal(bv)
			}
			return reflect.DeepEqual(av.Interface(), bv.Interface())
		}
		for i := range td.fields {
			f := &td.fields[i]
			if !equalValue(f.desc, unsafe.Add(aP, f.offset), unsafe.Add(bP, f.offset), depth+1) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		ap, bp := *(*unsafe.Pointer)(aP), *(*unsafe.Pointer)(bP)
		if ap == nil || bp == nil {
			return ap == bp
		}
		return equalValue(td.elem, ap, bp, depth+1)
	case reflect.Map:
		// nil map vs empty-non-nil map: same nil-vs-empty distinction as slices.
		if (*(*unsafe.Pointer)(aP) == nil) != (*(*unsafe.Pointer)(bP) == nil) {
			return false
		}
		return equalMapEV(td, aP, bP, depth)
	default:
		// Interface and exotic kinds: reflect.Value.Equal panics on a
		// non-comparable dynamic value (a []int / map / func inside an any), so
		// compare structurally with DeepEqual, which handles them. This path is
		// rare and off the hot path.
		av := reflect.NewAt(td.rType, aP).Elem()
		bv := reflect.NewAt(td.rType, bP).Elem()
		return reflect.DeepEqual(av.Interface(), bv.Interface())
	}
}

// equalSliceEV compares two non-[]byte slices element-by-element. For fixed-size
// pointer-free element types it does one bytes.Equal over the backing arrays
// (the block-memcmp fast path).
//
// Note: uses the existing sliceHeader type from reuse.go (fields Data/Len/Cap).
func equalSliceEV(td *typeDesc, aP, bP unsafe.Pointer, depth int) bool {
	ah := (*sliceHeader)(aP)
	bh := (*sliceHeader)(bP)
	if ah.Len != bh.Len {
		return false
	}
	n := ah.Len
	if n == 0 {
		return true
	}
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	stride := td.rType.Elem().Size()
	if elem != nil && elem.pod {
		// POD memcmp compares padding too: a padding-only diff yields a spurious opReplace, never wrong data — acceptable for Phase 1.
		ab := unsafe.Slice((*byte)(ah.Data), uintptr(n)*stride)
		bb := unsafe.Slice((*byte)(bh.Data), uintptr(n)*stride)
		return bytes.Equal(ab, bb)
	}
	for i := range n {
		if !equalValue(elem, unsafe.Add(ah.Data, uintptr(i)*stride),
			unsafe.Add(bh.Data, uintptr(i)*stride), depth+1) {
			return false
		}
	}
	return true
}

func equalArrayEV(td *typeDesc, aP, bP unsafe.Pointer, depth int) bool {
	if td.rType.Elem().Kind() == reflect.Uint8 {
		n := uintptr(td.rType.Len())
		return bytes.Equal(unsafe.Slice((*byte)(aP), n), unsafe.Slice((*byte)(bP), n))
	}
	n := td.rType.Len()
	stride := td.rType.Elem().Size()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	if elem != nil && elem.pod {
		// POD memcmp compares padding too: a padding-only diff yields a spurious opReplace, never wrong data — acceptable for Phase 1.
		total := uintptr(n) * stride
		return bytes.Equal(unsafe.Slice((*byte)(aP), total), unsafe.Slice((*byte)(bP), total))
	}
	for i := range n {
		if !equalValue(elem, unsafe.Add(aP, uintptr(i)*stride),
			unsafe.Add(bP, uintptr(i)*stride), depth+1) {
			return false
		}
	}
	return true
}

func equalMapEV(td *typeDesc, aP, bP unsafe.Pointer, depth int) bool {
	av := reflect.NewAt(td.rType, aP).Elem()
	bv := reflect.NewAt(td.rType, bP).Elem()
	if av.Len() != bv.Len() {
		return false
	}
	if av.Len() == 0 {
		// Both maps empty (lengths matched) and equalValue's Map case already
		// pre-checked nil-vs-non-nil before calling here, so matching nilness +
		// zero entries ⇒ equal contents. Skip the two reflect.New comparison
		// buffers — the common (nil/empty map) case allocated them for nothing.
		return true
	}
	valDesc := td.elem
	if valDesc == nil {
		var err error
		valDesc, err = descOf(td.rType.Elem())
		if err != nil {
			return false
		}
	}
	aCmp := reflect.New(valDesc.rType).Elem()
	bCmp := reflect.New(valDesc.rType).Elem()
	iter := av.MapRange()
	for iter.Next() {
		bVal := bv.MapIndex(iter.Key())
		if !bVal.IsValid() {
			return false
		}
		aCmp.Set(iter.Value())
		bCmp.Set(bVal)
		if !equalValue(valDesc, aCmp.Addr().UnsafePointer(), bCmp.Addr().UnsafePointer(), depth+1) {
			return false
		}
	}
	return true
}
