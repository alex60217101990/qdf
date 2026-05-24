//go:build qdf_reflect2

package qdf

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/modern-go/reflect2"
)

// reflect2 backend. Skips the reflect.MakeSlice / MakeMapWithSize type
// checks on the hot path; mostly visible on map-heavy decodes.
//
//	go build -tags qdf_reflect2 ./...

var reflect2Cache sync.Map // map[reflect.Type]reflect2.Type

func reflect2TypeFor(t reflect.Type) reflect2.Type {
	if v, ok := reflect2Cache.Load(t); ok {
		return v.(reflect2.Type)
	}
	rt := reflect2.Type2(t)
	actual, _ := reflect2Cache.LoadOrStore(t, rt)
	return actual.(reflect2.Type)
}

func makeSliceUnsafe(t reflect.Type, n int, p unsafe.Pointer) {
	st := reflect2TypeFor(t).(reflect2.SliceType)
	hdr := st.UnsafeMakeSlice(n, n)
	// UnsafeMakeSlice returns a heap-allocated slice header; copy the
	// three words into the caller's storage.
	*(*[3]uintptr)(p) = *(*[3]uintptr)(hdr)
}

func sliceDataUnsafe(t reflect.Type, p unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(p)
}

func makeMapUnsafe(t reflect.Type, n int, p unsafe.Pointer) {
	mt := reflect2TypeFor(t).(reflect2.MapType)
	m := mt.UnsafeMakeMap(n)
	*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(m)
}
