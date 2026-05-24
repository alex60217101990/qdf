package qdf

import (
	"reflect"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Specialised encoders / decoders for the most common Go map shapes,
// installed by fillDesc when the runtime type matches. These bypass the
// per-entry reflect.Value + SetMapIndex pair that dominates the generic
// reflect path on map-heavy payloads.

var (
	mapStringStringType = reflect.TypeFor[map[string]string]()
	mapStringIntType    = reflect.TypeFor[map[string]int]()
	mapStringInt64Type  = reflect.TypeFor[map[string]int64]()
	mapStringAnyType    = reflect.TypeFor[map[string]any]()
)

// installMapFastPath returns (encode, decode, true) if t is one of the
// specialised map shapes; otherwise (_, _, false).
func installMapFastPath(t reflect.Type) (
	enc func(*Encoder, unsafe.Pointer) error,
	dec func(*Decoder, unsafe.Pointer) error,
	ok bool,
) {
	switch t {
	case mapStringStringType:
		return encodeMapStringString, decodeMapStringString, true
	case mapStringIntType:
		return encodeMapStringInt, decodeMapStringInt, true
	case mapStringInt64Type:
		return encodeMapStringInt64, decodeMapStringInt64, true
	case mapStringAnyType:
		return encodeMapStringAny, decodeMapStringAny, true
	}
	return nil, nil, false
}

// ----- map[string]string -----

func encodeMapStringString(e *Encoder, p unsafe.Pointer) error {
	m := *(*map[string]string)(p)
	if m == nil {
		e.WriteNil()
		return nil
	}
	e.WriteMapHeader(len(m))
	for k, v := range m {
		e.WriteString(k)
		e.WriteString(v)
	}
	return nil
}

func decodeMapStringString(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagNil {
		d.i++
		*(*map[string]string)(p) = nil
		return nil
	}
	n, err := d.ReadMapHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 2); err != nil {
		return err
	}
	m := make(map[string]string, n)
	for range n {
		kb, err := d.readStringBytes()
		if err != nil {
			return err
		}
		k := d.keyCache.Make(kb)
		v, err := d.ReadString()
		if err != nil {
			return err
		}
		m[k] = v
	}
	*(*map[string]string)(p) = m
	return nil
}

// ----- map[string]int -----

func encodeMapStringInt(e *Encoder, p unsafe.Pointer) error {
	m := *(*map[string]int)(p)
	if m == nil {
		e.WriteNil()
		return nil
	}
	e.WriteMapHeader(len(m))
	for k, v := range m {
		e.WriteString(k)
		e.WriteInt(int64(v))
	}
	return nil
}

func decodeMapStringInt(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagNil {
		d.i++
		*(*map[string]int)(p) = nil
		return nil
	}
	n, err := d.ReadMapHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 2); err != nil {
		return err
	}
	m := make(map[string]int, n)
	for range n {
		kb, err := d.readStringBytes()
		if err != nil {
			return err
		}
		k := d.keyCache.Make(kb)
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		m[k] = int(v)
	}
	*(*map[string]int)(p) = m
	return nil
}

// ----- map[string]int64 -----

func encodeMapStringInt64(e *Encoder, p unsafe.Pointer) error {
	m := *(*map[string]int64)(p)
	if m == nil {
		e.WriteNil()
		return nil
	}
	e.WriteMapHeader(len(m))
	for k, v := range m {
		e.WriteString(k)
		e.WriteInt(v)
	}
	return nil
}

func decodeMapStringInt64(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagNil {
		d.i++
		*(*map[string]int64)(p) = nil
		return nil
	}
	n, err := d.ReadMapHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 2); err != nil {
		return err
	}
	m := make(map[string]int64, n)
	for range n {
		kb, err := d.readStringBytes()
		if err != nil {
			return err
		}
		k := d.keyCache.Make(kb)
		v, err := d.ReadInt()
		if err != nil {
			return err
		}
		m[k] = v
	}
	*(*map[string]int64)(p) = m
	return nil
}

// ----- map[string]any -----

func encodeMapStringAny(e *Encoder, p unsafe.Pointer) error {
	m := *(*map[string]any)(p)
	if m == nil {
		e.WriteNil()
		return nil
	}
	e.WriteMapHeader(len(m))
	for k, v := range m {
		e.WriteString(k)
		if err := encodeReflect(e, v); err != nil {
			return err
		}
	}
	return nil
}

func decodeMapStringAny(d *Decoder, p unsafe.Pointer) error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	if t == tagNil {
		d.i++
		*(*map[string]any)(p) = nil
		return nil
	}
	n, err := d.ReadMapHeader()
	if err != nil {
		return err
	}
	if err := d.CheckLength(n, 2); err != nil {
		return err
	}
	m := make(map[string]any, n)
	for range n {
		kb, err := d.readStringBytes()
		if err != nil {
			return err
		}
		k := d.keyCache.Make(kb)
		v, err := decodeAny(d)
		if err != nil {
			return err
		}
		m[k] = v
	}
	*(*map[string]any)(p) = m
	return nil
}

var _ = unsafestr.String
