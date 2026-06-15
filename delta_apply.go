package qdf

import (
	"reflect"
	"unsafe"
)

// applyValue reads one op (op byte + payload) and applies it to the value at baseP.
func applyValue(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
	if dec.i >= len(dec.buf) {
		return ErrInvalidPatch
	}
	op := dec.buf[dec.i]
	dec.i++
	switch op {
	case opReplace:
		return td.decode(dec, baseP)
	case opMerge:
		return applyMerge(dec, td, baseP)
	default:
		return ErrInvalidPatch
	}
}

// applyMerge dispatches a recursive sub-patch by container kind.
func applyMerge(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
	switch td.kind {
	case reflect.Struct:
		return applyStruct(dec, td, baseP)
	case reflect.Pointer:
		// merge into the pointed-at struct. base pointer must be non-nil: the diff
		// side only emits opMerge for a pointer when both old/new were non-nil, and
		// baseFP guarantees base matches old.
		ptr := *(*unsafe.Pointer)(baseP)
		if ptr == nil {
			return ErrInvalidPatch
		}
		return applyStruct(dec, td.elem, ptr)
	case reflect.Slice:
		return applySlice(dec, td, baseP)
	case reflect.Array:
		return applyArray(dec, td, baseP)
	default:
		return ErrInvalidPatch // later tasks add map/ptr merge
	}
}

// applySlice reads a tagSlicePatch and reconciles the base slice in place:
// resize to newLen (preserving overlap), then apply each entry.
func applySlice(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagSlicePatch {
		return ErrInvalidPatch
	}
	dec.i++
	newLen64, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k
	newLen := int(newLen64)
	// Bound newLen by remaining input so a hostile patch cannot pre-allocate a
	// huge slice (each entry costs >=1 byte; this is a loose but safe ceiling).
	if newLen < 0 || uint64(newLen) > uint64(len(dec.buf)) {
		return ErrInvalidPatch
	}

	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
	}
	stride := td.rType.Elem().Size()
	bv := reflect.NewAt(td.rType, baseP).Elem()
	if newLen != bv.Len() {
		nv := reflect.MakeSlice(td.rType, newLen, newLen)
		cp := bv.Len()
		if newLen < cp {
			cp = newLen
		}
		reflect.Copy(nv, bv.Slice(0, cp))
		bv.Set(nv)
	}
	sh := (*sliceHeader)(baseP) // re-read after potential Set (Data pointer moved)

	nEntries, k2 := readUvarint(dec.buf[dec.i:])
	if k2 <= 0 || nEntries > uint64(newLen) {
		return ErrInvalidPatch
	}
	dec.i += k2
	for e := uint64(0); e < nEntries; e++ {
		idx, k3 := readUvarint(dec.buf[dec.i:])
		if k3 <= 0 || idx >= uint64(newLen) {
			return ErrInvalidPatch
		}
		dec.i += k3
		if err := applyValue(dec, elem, unsafe.Add(sh.Data, uintptr(idx)*stride)); err != nil {
			return err
		}
	}
	return nil
}

func applyArray(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
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
	for e := uint64(0); e < nEntries; e++ {
		idx, k3 := readUvarint(dec.buf[dec.i:])
		if k3 <= 0 || idx >= newLen {
			return ErrInvalidPatch
		}
		dec.i += k3
		if err := applyValue(dec, elem, unsafe.Add(baseP, uintptr(idx)*stride)); err != nil {
			return err
		}
	}
	return nil
}

// applyStruct reads a tagStructPatch body and overwrites only the listed fields.
func applyStruct(dec *Decoder, td *typeDesc, baseP unsafe.Pointer) error {
	if dec.i >= len(dec.buf) || dec.buf[dec.i] != tagStructPatch {
		return ErrInvalidPatch
	}
	dec.i++
	nChanged, k := readUvarint(dec.buf[dec.i:])
	if k <= 0 {
		return ErrInvalidPatch
	}
	dec.i += k
	for c := uint64(0); c < nChanged; c++ {
		idx, k2 := readUvarint(dec.buf[dec.i:])
		if k2 <= 0 || idx >= uint64(len(td.fields)) {
			return ErrInvalidPatch
		}
		dec.i += k2
		f := &td.fields[idx]
		if err := applyValue(dec, f.desc, unsafe.Add(baseP, f.offset)); err != nil {
			return err
		}
	}
	return nil
}
