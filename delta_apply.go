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
	default:
		return ErrInvalidPatch // later tasks add slice/map/ptr merge
	}
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
