//go:build !qdf_reflect2

package qdf

import (
	"reflect"
	"unsafe"
)

// Stdlib reflect backend for slice / map allocation. The qdf_reflect2
// build tag substitutes a modern-go/reflect2 implementation.

func makeSliceUnsafe(t reflect.Type, n int, p unsafe.Pointer) {
	v := reflect.MakeSlice(t, n, n)
	reflect.NewAt(t, p).Elem().Set(v)
}

func sliceDataUnsafe(t reflect.Type, p unsafe.Pointer) unsafe.Pointer {
	v := reflect.NewAt(t, p).Elem()
	return unsafe.Pointer(v.UnsafePointer())
}

func makeMapUnsafe(t reflect.Type, n int, p unsafe.Pointer) {
	v := reflect.MakeMapWithSize(t, n)
	reflect.NewAt(t, p).Elem().Set(v)
}
