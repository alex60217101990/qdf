package qdf

import (
	"reflect"
	"unsafe"
)

// applyValue reads one op (op byte + payload) and applies it to the value at baseP.
func applyValue(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if depth > maxDeltaDepth {
		return ErrInvalidPatch
	}
	if dec.i >= len(dec.buf) {
		return ErrInvalidPatch
	}
	op := dec.buf[dec.i]
	dec.i++
	switch op {
	case opReplace:
		return td.decode(dec, baseP)
	case opMerge:
		return applyMerge(dec, td, baseP, depth)
	default:
		return ErrInvalidPatch
	}
}

// applyMerge dispatches a recursive sub-patch by container kind.
func applyMerge(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	switch td.kind {
	case reflect.Struct:
		return applyStruct(dec, td, baseP, depth)
	case reflect.Pointer:
		// merge into the pointed-at struct. base pointer must be non-nil: the diff
		// side only emits opMerge for a pointer when both old/new were non-nil, and
		// baseFP guarantees base matches old.
		ptr := *(*unsafe.Pointer)(baseP)
		if ptr == nil {
			return ErrInvalidPatch
		}
		return applyStruct(dec, td.elem, ptr, depth+1)
	case reflect.Slice:
		return applySlice(dec, td, baseP, depth)
	case reflect.Array:
		return applyArray(dec, td, baseP, depth)
	case reflect.Map:
		return applyMap(dec, td, baseP, depth)
	default:
		return ErrInvalidPatch // unknown/unmergeable kind
	}
}

// applySlice reads a tagSlicePatch and reconciles the base slice in place:
// resize to newLen (preserving overlap), then apply each entry.
func applySlice(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagSlicePatch {
		return ErrInvalidPatch
	}
	dec.i++
	newLen64, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k

	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	stride := td.rType.Elem().Size()
	bv := reflect.NewAt(td.rType, baseP).Elem()

	newLen := int(newLen64)
	if newLen < 0 {
		return ErrInvalidPatch
	}
	// Bound hostile growth: every element beyond the current base length must be
	// described by the patch (>=1 byte each), so the number of grown elements
	// cannot exceed the remaining input. This rejects an allocation-amplification
	// patch (huge newLen, zero entries) while still allowing the legitimate delta
	// case of a large base with a tiny no-grow patch (newLen <= base length).
	if newLen > bv.Len() && uint64(newLen-bv.Len()) > uint64(len(dec.buf)-dec.i) {
		return ErrInvalidPatch
	}

	if newLen != bv.Len() {
		nv := reflect.MakeSlice(td.rType, newLen, newLen)
		cp := min(bv.Len(), newLen)
		reflect.Copy(nv, bv.Slice(0, cp))
		bv.Set(nv)
	}
	sh := (*sliceHeader)(baseP) // re-read after potential Set (Data pointer moved)

	nEntries, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nEntries > uint64(newLen) {
		return ErrInvalidPatch
	}
	dec.i += k2
	for range nEntries {
		idx, k3 := readUvarint(dec.buf[dec.i:])
		if k3 <= 0 || idx >= uint64(newLen) {
			return ErrInvalidPatch
		}
		dec.i += k3
		if err := applyValue(dec, elem, unsafe.Add(sh.Data, uintptr(idx)*stride), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func applyArray(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagSlicePatch {
		return ErrInvalidPatch
	}
	dec.i++
	newLen, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 || newLen != uint64(td.rType.Len()) {
		return ErrInvalidPatch
	}
	dec.i += k
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	stride := td.rType.Elem().Size()
	nEntries, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nEntries > newLen {
		return ErrInvalidPatch
	}
	dec.i += k2
	for range nEntries {
		idx, k3 := readUvarint(dec.buf[dec.i:])
		if k3 <= 0 || idx >= newLen {
			return ErrInvalidPatch
		}
		dec.i += k3
		if err := applyValue(dec, elem, unsafe.Add(baseP, uintptr(idx)*stride), depth+1); err != nil {
			return err
		}
	}
	return nil
}

// applyMap reads a tagMapPatch and mutates the base map in place: set/merge the
// updated keys, then delete the tombstoned keys.
func applyMap(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagMapPatch {
		return ErrInvalidPatch
	}
	dec.i++
	mv := reflect.NewAt(td.rType, baseP).Elem()
	if mv.IsNil() {
		mv.Set(reflect.MakeMap(td.rType))
	}
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

	nUpd, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 || nUpd > uint64(len(dec.buf)) {
		return ErrInvalidPatch
	}
	dec.i += k
	for range nUpd {
		keyBuf := reflect.New(keyType).Elem()
		if err := keyDesc.decode(dec, keyBuf.Addr().UnsafePointer()); err != nil {
			return err
		}
		valBuf := reflect.New(valDesc.rType).Elem()
		if existing := mv.MapIndex(keyBuf); existing.IsValid() {
			valBuf.Set(existing) // merge target starts from the current value
		}
		if err := applyValue(dec, valDesc, valBuf.Addr().UnsafePointer(), depth+1); err != nil {
			return err
		}
		mv.SetMapIndex(keyBuf, valBuf)
	}

	nDel, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nDel > uint64(len(dec.buf)) {
		return ErrInvalidPatch
	}
	dec.i += k2
	for range nDel {
		keyBuf := reflect.New(keyType).Elem()
		if err := keyDesc.decode(dec, keyBuf.Addr().UnsafePointer()); err != nil {
			return err
		}
		mv.SetMapIndex(keyBuf, reflect.Value{}) // delete
	}
	return nil
}

// applyStruct reads a tagStructPatch body and overwrites only the listed fields.
func applyStruct(dec *Decoder, td *typeDesc, baseP unsafe.Pointer, depth int) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagStructPatch {
		return ErrInvalidPatch
	}
	dec.i++
	nChanged, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k
	for range nChanged {
		idx, k2 := readUvarint(dec.buf[dec.i:])
		if k2 <= 0 || idx >= uint64(len(td.fields)) {
			return ErrInvalidPatch
		}
		dec.i += k2
		f := &td.fields[idx]
		if err := applyValue(dec, f.desc, unsafe.Add(baseP, f.offset), depth+1); err != nil {
			return err
		}
	}
	return nil
}
