package qdf

import (
	"bytes"
	"reflect"
	"unsafe"
)

// diffValue compares the value at oldP/newP and, if changed, writes one op
// (op byte + payload). The caller has already written any preceding selector
// (field index / slice index / map key).
func diffValue(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) error {
	if equalValue(td, oldP, newP) {
		return nil
	}
	switch td.kind {
	case reflect.Struct:
		if len(td.fields) == 0 { // time.Time / marshaler struct: whole replace
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return diffStruct(enc, td, oldP, newP)
	case reflect.Pointer:
		op, np := *(*unsafe.Pointer)(oldP), *(*unsafe.Pointer)(newP)
		if op == nil || np == nil {
			// presence change (both-nil was handled by equalValue earlier) → replace
			return writeReplace(enc, td, newP)
		}
		if td.elem != nil && td.elem.kind == reflect.Struct && len(td.elem.fields) > 0 {
			enc.buf = append(enc.buf, opMerge)
			return diffStruct(enc, td.elem, op, np)
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
		return diffSlice(enc, td, oldP, newP)
	case reflect.Array:
		if td.rType.Elem().Kind() == reflect.Uint8 {
			return writeReplace(enc, td, newP) // [N]byte: whole replace
		}
		enc.buf = append(enc.buf, opMerge)
		return diffArray(enc, td, oldP, newP)
	case reflect.Map:
		// Same nil↔non-nil concern as slices: applyMap never reconstructs a nil
		// map, so a nilness transition must be a whole-value replace.
		if (*(*unsafe.Pointer)(oldP) == nil) != (*(*unsafe.Pointer)(newP) == nil) {
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return diffMap(enc, td, oldP, newP)
	default:
		// Phase 1: non-struct change is a whole-value replace. Pointer/slice/map
		// merge ops are added in later tasks (they extend this switch).
		return writeReplace(enc, td, newP)
	}
}

// diffSlice writes a tagSlicePatch body for a non-[]byte slice. The whole-slice
// equality short-circuit already happened in diffValue→equalValue (one SIMD
// bytes.Equal over POD backing arrays), so we only get here on a real difference.
func diffSlice(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) error {
	oh := (*sliceHeader)(oldP)
	nh := (*sliceHeader)(newP)
	stride := td.rType.Elem().Size()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	return diffElems(enc, elem, td.rType.Elem(), stride, oh.Data, oh.Len, nh.Data, nh.Len)
}

func diffArray(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) error {
	n := td.rType.Len()
	stride := td.rType.Elem().Size()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	return diffElems(enc, elem, td.rType.Elem(), stride, oldP, n, newP, n)
}

// diffMap writes a tagMapPatch: updated/added keys (each with a recursive op),
// then a tombstone list of deleted keys. Keys are the identity, so there is no
// positional ambiguity. Two passes (count, then emit) avoid buffer surgery.
func diffMap(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) error {
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
		return equalValue(valDesc, oCmp.Addr().UnsafePointer(), nCmp.Addr().UnsafePointer())
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
			oBuf := reflect.New(valDesc.rType).Elem()
			oBuf.Set(oVal)
			nBuf := reflect.New(valDesc.rType).Elem()
			nBuf.Set(nVal)
			if err := diffValue(enc, valDesc,
				oBuf.Addr().UnsafePointer(), nBuf.Addr().UnsafePointer()); err != nil {
				return err
			}
		} else {
			nBuf := reflect.New(valDesc.rType).Elem()
			nBuf.Set(nVal)
			if err := writeReplace(enc, valDesc, nBuf.Addr().UnsafePointer()); err != nil {
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
func diffElems(enc *Encoder, elem *typeDesc, elemType reflect.Type, stride uintptr,
	oldData unsafe.Pointer, oldLen int, newData unsafe.Pointer, newLen int) error {
	minLen := min(newLen, oldLen)
	pod := noPointers(elemType)
	var entries []int
	for i := range minLen {
		oP := unsafe.Add(oldData, uintptr(i)*stride)
		nP := unsafe.Add(newData, uintptr(i)*stride)
		var same bool
		if pod {
			same = bytes.Equal(unsafe.Slice((*byte)(oP), stride), unsafe.Slice((*byte)(nP), stride))
		} else {
			same = equalValue(elem, oP, nP)
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
			if err := diffValue(enc, elem, oP, nP); err != nil {
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
func diffStruct(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) error {
	var changed []int
	for i := range td.fields {
		f := &td.fields[i]
		if !equalValue(f.desc, unsafe.Add(oldP, f.offset), unsafe.Add(newP, f.offset)) {
			changed = append(changed, i)
		}
	}
	enc.buf = append(enc.buf, tagStructPatch)
	enc.buf = appendUvarint(enc.buf, uint64(len(changed)))
	for _, i := range changed {
		f := &td.fields[i]
		enc.buf = appendUvarint(enc.buf, uint64(i))
		if err := diffValue(enc, f.desc,
			unsafe.Add(oldP, f.offset), unsafe.Add(newP, f.offset)); err != nil {
			return err
		}
	}
	return nil
}

// equalValue reports whether the value of type described by td at aP equals the
// one at bP. POD scalars reduce to a width compare; strings/[]byte to a SIMD
// memcmp (bytes.Equal); containers recurse. This is the diff walk's unchanged
// fast path — for an unchanged subtree it is the ONLY work done.
func equalValue(td *typeDesc, aP, bP unsafe.Pointer) bool {
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
		return equalSliceEV(td, aP, bP)
	case reflect.Array:
		return equalArrayEV(td, aP, bP)
	case reflect.Struct:
		// time.Time and custom-marshaler structs have no td.fields; fall back to
		// reflect.DeepEqual on the reflect value (rare; correctness over speed).
		if len(td.fields) == 0 {
			return reflect.NewAt(td.rType, aP).Elem().
				Equal(reflect.NewAt(td.rType, bP).Elem())
		}
		for i := range td.fields {
			f := &td.fields[i]
			if !equalValue(f.desc, unsafe.Add(aP, f.offset), unsafe.Add(bP, f.offset)) {
				return false
			}
		}
		return true
	case reflect.Pointer:
		ap, bp := *(*unsafe.Pointer)(aP), *(*unsafe.Pointer)(bP)
		if ap == nil || bp == nil {
			return ap == bp
		}
		return equalValue(td.elem, ap, bp)
	case reflect.Map:
		// nil map vs empty-non-nil map: same nil-vs-empty distinction as slices.
		if (*(*unsafe.Pointer)(aP) == nil) != (*(*unsafe.Pointer)(bP) == nil) {
			return false
		}
		return equalMapEV(td, aP, bP)
	default:
		// Interface and anything exotic: reflect fallback.
		return reflect.NewAt(td.rType, aP).Elem().
			Equal(reflect.NewAt(td.rType, bP).Elem())
	}
}

// equalSliceEV compares two non-[]byte slices element-by-element. For fixed-size
// pointer-free element types it does one bytes.Equal over the backing arrays
// (the block-memcmp fast path).
//
// Note: uses the existing sliceHeader type from reuse.go (fields Data/Len/Cap).
func equalSliceEV(td *typeDesc, aP, bP unsafe.Pointer) bool {
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
	if noPointers(td.rType.Elem()) {
		// POD memcmp compares padding too: a padding-only diff yields a spurious opReplace, never wrong data — acceptable for Phase 1.
		ab := unsafe.Slice((*byte)(ah.Data), uintptr(n)*stride)
		bb := unsafe.Slice((*byte)(bh.Data), uintptr(n)*stride)
		return bytes.Equal(ab, bb)
	}
	for i := range n {
		if !equalValue(elem, unsafe.Add(ah.Data, uintptr(i)*stride),
			unsafe.Add(bh.Data, uintptr(i)*stride)) {
			return false
		}
	}
	return true
}

func equalArrayEV(td *typeDesc, aP, bP unsafe.Pointer) bool {
	if td.rType.Elem().Kind() == reflect.Uint8 {
		n := uintptr(td.rType.Len())
		return bytes.Equal(unsafe.Slice((*byte)(aP), n), unsafe.Slice((*byte)(bP), n))
	}
	n := td.rType.Len()
	stride := td.rType.Elem().Size()
	if noPointers(td.rType.Elem()) {
		// POD memcmp compares padding too: a padding-only diff yields a spurious opReplace, never wrong data — acceptable for Phase 1.
		total := uintptr(n) * stride
		return bytes.Equal(unsafe.Slice((*byte)(aP), total), unsafe.Slice((*byte)(bP), total))
	}
	for i := range n {
		if !equalValue(td.elem, unsafe.Add(aP, uintptr(i)*stride),
			unsafe.Add(bP, uintptr(i)*stride)) {
			return false
		}
	}
	return true
}

func equalMapEV(td *typeDesc, aP, bP unsafe.Pointer) bool {
	av := reflect.NewAt(td.rType, aP).Elem()
	bv := reflect.NewAt(td.rType, bP).Elem()
	if av.Len() != bv.Len() {
		return false
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
		if !equalValue(valDesc, aCmp.Addr().UnsafePointer(), bCmp.Addr().UnsafePointer()) {
			return false
		}
	}
	return true
}
