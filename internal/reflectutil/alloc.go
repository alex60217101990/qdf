//go:build !qdf_reflect2

// Package reflectutil holds the reflect-based helpers the qdf
// codec uses to allocate slices and maps without going through
// reflect.Value. The default backend uses the stdlib reflect
// package; the qdf_reflect2 build tag swaps in a
// modern-go/reflect2 implementation that skips the type checks
// on the hot path.
package reflectutil

import (
	"reflect"
	"unsafe"
)

// MakeSlice allocates a slice of element type t.Elem() with length n
// and writes the resulting header into the storage at p.
func MakeSlice(t reflect.Type, n int, p unsafe.Pointer) {
	v := reflect.MakeSlice(t, n, n)
	reflect.NewAt(t, p).Elem().Set(v)
}

// SliceData returns the data pointer of the slice value stored at
// p (whose type is the slice type t).
func SliceData(t reflect.Type, p unsafe.Pointer) unsafe.Pointer {
	v := reflect.NewAt(t, p).Elem()
	return unsafe.Pointer(v.UnsafePointer())
}

// MakeMap allocates a map of type t with the given size hint and
// writes the resulting header into the storage at p.
func MakeMap(t reflect.Type, n int, p unsafe.Pointer) {
	v := reflect.MakeMapWithSize(t, n)
	reflect.NewAt(t, p).Elem().Set(v)
}
