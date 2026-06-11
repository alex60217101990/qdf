package qdf

import (
	"reflect"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/reflectutil"
)

// This file groups the decode-time "reuse the caller's container" helpers:
// reuseOrMakeSlice / reuseOrMakeMap let a decode into a pre-sized (pooled)
// target reuse the existing slice backing / map instead of allocating a fresh
// one — the dominant decode allocation on a server hot path that recycles its
// decode target. sliceHeader and noPointers support the slice variant.

// sliceHeader mirrors reflect.SliceHeader using unsafe.Pointer instead of
// uintptr so the GC can see the data pointer. Required for taking pointers
// out of a slice without losing it to the GC.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// noPointers reports whether t contains no pointers (so a byte-clear of its
// memory is GC-safe — no write barriers needed). Used to gate decode slice
// backing reuse: a pointer-free element can be zeroed with clear() over a raw
// []byte view before the values are decoded in place.
func noPointers(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array:
		return noPointers(t.Elem())
	case reflect.Struct:
		for field := range t.Fields() {
			if !noPointers(field.Type) {
				return false
			}
		}
		return true
	default:
		// string, slice, map, ptr, interface, chan, func, unsafe.Pointer.
		return false
	}
}

// reuseOrMakeSlice sets the slice at p to length n and returns its data base.
// When the caller-provided slice already has cap >= n it reuses that backing
// instead of allocating a fresh one — eliminating the result backing
// allocation on a decode into a pre-sized (pooled) slice, the dominant decode
// allocation. The reused elements are zeroed first so a wire shape that omits
// fields (schema evolution) cannot leak stale data: pointer-free elements
// (elemPF) take a barrier-free byte clear; pointer-containing elements take
// reflect.Value.Clear (a single barrier-correct typedmemclr). With no usable
// backing it allocates fresh via MakeSlice. elemPF must be noPointers(t.Elem()).
func reuseOrMakeSlice(t reflect.Type, n int, p unsafe.Pointer, stride uintptr, elemPF bool) unsafe.Pointer {
	if hdr := (*sliceHeader)(p); hdr.Cap >= n && hdr.Data != nil {
		hdr.Len = n
		if elemPF {
			// Pointer-free: a raw byte clear is GC-safe and zeroes any struct
			// fields the wire shape does not set (schema evolution).
			clear(unsafe.Slice((*byte)(hdr.Data), n*int(stride)))
		} else {
			// Pointer-containing: a byte clear would skip write barriers and
			// corrupt the GC. reflect.Value.Clear bulk-zeroes the slice
			// elements with a single barrier-correct typedmemclr.
			reflect.NewAt(t, p).Elem().Clear()
		}
		return hdr.Data
	}
	reflectutil.MakeSlice(t, n, p)
	return reflectutil.SliceData(t, p)
}

// reuseOrMakeMap returns a map[K]V to decode n entries into, reusing the
// caller's existing non-nil map at p (cleared so no stale keys survive a
// schema-evolving re-decode) instead of allocating a fresh one. clear() keeps
// the bucket capacity, so a server re-decoding into the same target skips the
// makemap + bucket-growth that dominate map-heavy decode allocation. A nil
// destination (fresh target) allocates as before. Mirrors reuseOrMakeSlice.
func reuseOrMakeMap[K comparable, V any](p unsafe.Pointer, n int) map[K]V {
	if m := *(*map[K]V)(p); m != nil {
		clear(m)
		return m
	}
	return make(map[K]V, n)
}
