//go:build qdf_reflect2

package reflectutil

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

// MakeSlice — see alloc.go for the contract.
func MakeSlice(t reflect.Type, n int, p unsafe.Pointer) {
	st := reflect2TypeFor(t).(reflect2.SliceType)
	hdr := st.UnsafeMakeSlice(n, n)
	*(*[3]uintptr)(p) = *(*[3]uintptr)(hdr)
}

// SliceData — see alloc.go for the contract.
func SliceData(_ reflect.Type, p unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(p)
}

// MakeMap — see alloc.go for the contract.
func MakeMap(t reflect.Type, n int, p unsafe.Pointer) {
	mt := reflect2TypeFor(t).(reflect2.MapType)
	m := mt.UnsafeMakeMap(n)
	*(*unsafe.Pointer)(p) = *(*unsafe.Pointer)(m)
}
