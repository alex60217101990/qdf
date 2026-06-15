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

// keyTokenReflect is the allocating fallback for exotic comparable key types
// (a comparable struct). Rare and off the hot path; the keyed diff/apply path is
// gated (keyTokenable, a later task) to the non-fallback kinds, so a struct key
// falls back to positional diff rather than relying on this token.
func keyTokenReflect(kd *typeDesc, kp unsafe.Pointer) string {
	return reflect.NewAt(kd.rType, kp).Elem().String()
}
