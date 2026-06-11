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
// destination (fresh target) first tries the Decoder's per-type map free-list
// (maps harvested from a reused []struct{map} target whose elements
// decode-slice-reuse is about to zero), then allocates fresh. Mirrors
// reuseOrMakeSlice.
func reuseOrMakeMap[K comparable, V any](d *Decoder, p unsafe.Pointer, n int) map[K]V {
	if m := *(*map[K]V)(p); m != nil { // direct-target reuse (unchanged)
		clear(m)
		return m
	}
	// Only consult the harvest free-list when it actually holds something — a
	// fresh decode (no reused []struct{map} target to harvest from) skips the
	// reflect.TypeFor + map lookup entirely, so this path is zero-cost vs a plain
	// make when no recycling is in play.
	if len(d.mapFreeList) > 0 {
		t := reflect.TypeFor[map[K]V]()
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			// Reconstruct the map header value from its harvested hmap pointer.
			m := *(*map[K]V)(unsafe.Pointer(&ptr))
			clear(m)
			return m
		}
	}
	return make(map[K]V, n)
}

// popOrMakeMap returns a map[K]V to decode n entries into, recycling a harvested
// map of this type from the Decoder free-list (cleared) when one is available,
// else allocating fresh. Unlike reuseOrMakeMap it has no destination pointer
// (the caller — decodeAny — returns the map rather than writing through a *map),
// so it only consults the free-list. Zero-cost when the free-list is empty.
func popOrMakeMap[K comparable, V any](d *Decoder, n int) map[K]V {
	if len(d.mapFreeList) > 0 {
		t := reflect.TypeFor[map[K]V]()
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			m := *(*map[K]V)(unsafe.Pointer(&ptr))
			clear(m)
			return m
		}
	}
	return make(map[K]V, n)
}

// reuseOrMakeMapReflect installs at p a map of type t to decode n entries into,
// reusing the caller's existing non-nil map (cleared) or a harvested free-list
// map of type t, else allocating fresh via reflectutil.MakeMap. The reflect-path
// analogue of reuseOrMakeMap, used by decodeMap. Zero-cost (just a nil check)
// when there is nothing to reuse.
func reuseOrMakeMapReflect(d *Decoder, t reflect.Type, n int, p unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) != nil { // direct-target reuse: existing map at p
		reflect.NewAt(t, p).Elem().Clear() // clear keeps the map, drops entries
		return
	}
	if len(d.mapFreeList) > 0 {
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			*(*unsafe.Pointer)(p) = ptr        // install the harvested map at p
			reflect.NewAt(t, p).Elem().Clear() // empty its entries before refill
			return
		}
	}
	reflectutil.MakeMap(t, n, p)
}

// maxRecycledMaps bounds how many maps of one type the harvest free-list
// retains, so a one-off huge []struct{map} decode can't pin unbounded map
// headers. Past the cap the surplus maps are simply not recycled (allocated
// fresh) — a size bound, not a correctness limit.
const maxRecycledMaps = 1 << 14

// typeDescHasMap reports whether a value of type td holds a map at any
// struct-nesting depth (a map field, or a map reachable through nested struct
// fields). decodeSlice computes this once per slice type so the per-decode
// harvest walk runs only for elements that can actually yield a recyclable map.
// Conservative: maps behind a pointer/interface/slice/array field are not
// counted (their decoders re-allocate them), matching what harvestValue walks.
func typeDescHasMap(td *typeDesc) bool {
	switch td.kind {
	case reflect.Map:
		return true
	case reflect.Struct:
		for j := range td.fields {
			if fd := &td.fields[j]; fd.desc != nil && typeDescHasMap(fd.desc) {
				return true
			}
		}
	}
	return false
}

// harvestMaps pushes the non-nil maps held by the first oldLen elements (each a
// value of the slice's element type elem) onto the Decoder's per-type free-list,
// so reuseOrMakeMap can recycle them after decode-slice-reuse zeroes those
// elements. Recurses through nested struct fields, so a map at any nesting depth
// inside the element is harvested; a map element directly ([]map) is harvested
// too. (Maps behind a pointer/interface/slice/array field, and map-valued maps,
// are not walked — those are re-allocated by their own decoders.)
func harvestMaps(d *Decoder, elem *typeDesc, base unsafe.Pointer, stride uintptr, oldLen int) {
	for i := 0; i < oldLen; i++ {
		d.harvestValue(elem, unsafe.Add(base, uintptr(i)*stride))
	}
}

// harvestValue harvests every map reachable from one value of type td at p,
// recursing through struct fields.
func (d *Decoder) harvestValue(td *typeDesc, p unsafe.Pointer) {
	switch td.kind {
	case reflect.Map:
		if hmap := *(*unsafe.Pointer)(p); hmap != nil {
			if d.mapFreeList == nil {
				d.mapFreeList = make(map[reflect.Type][]unsafe.Pointer)
			}
			if len(d.mapFreeList[td.rType]) < maxRecycledMaps {
				d.mapFreeList[td.rType] = append(d.mapFreeList[td.rType], hmap)
			}
		}
	case reflect.Struct:
		for j := range td.fields {
			fd := &td.fields[j]
			if fd.desc == nil {
				continue
			}
			if k := fd.desc.kind; k == reflect.Map || k == reflect.Struct {
				d.harvestValue(fd.desc, unsafe.Add(p, fd.offset))
			}
		}
	}
}
