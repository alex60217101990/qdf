package qdf

import (
	"bytes"
	"reflect"
	"unsafe"
)

// diffValue compares the value at oldP/newP and, if changed, writes one op
// (op byte + payload). Returns whether anything was written. The caller has
// already written any preceding selector (field index / slice index / map key).
func diffValue(enc *Encoder, td *typeDesc, oldP, newP unsafe.Pointer) (bool, error) {
	if equalValue(td, oldP, newP) {
		return false, nil
	}
	switch td.kind {
	case reflect.Struct:
		if len(td.fields) == 0 { // time.Time / marshaler struct: whole replace
			return writeReplace(enc, td, newP)
		}
		enc.buf = append(enc.buf, opMerge)
		return true, diffStruct(enc, td, oldP, newP)
	default:
		// Phase 1: non-struct change is a whole-value replace. Pointer/slice/map
		// merge ops are added in later tasks (they extend this switch).
		return writeReplace(enc, td, newP)
	}
}

// writeReplace writes opReplace + the whole new value via the normal codec.
func writeReplace(enc *Encoder, td *typeDesc, newP unsafe.Pointer) (bool, error) {
	enc.buf = append(enc.buf, opReplace)
	if err := td.encode(enc, newP); err != nil {
		return false, err
	}
	return true, nil
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
		if _, err := diffValue(enc, f.desc,
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
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uintptr,
		reflect.Float64:
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
	for i := 0; i < n; i++ {
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
	for i := 0; i < n; i++ {
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
	iter := av.MapRange()
	for iter.Next() {
		bvVal := bv.MapIndex(iter.Key())
		if !bvVal.IsValid() {
			return false
		}
		if !iter.Value().Equal(bvVal) {
			return false
		}
	}
	return true
}
