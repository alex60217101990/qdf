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
//
// The delta diff hot path (the per-element caller that once motivated a cache)
// now reads the precomputed typeDesc.pod field instead, so the remaining callers
// (columnar.go, reflect_encode.go) are per-slice/per-type — not hot enough to
// warrant a sync.Map. This is the uncached structural walk.
func noPointers(t reflect.Type) bool {
	return noPointersWalk(t)
}

// noPointersWalk is the uncached structural worker behind noPointers. It uses
// NumField/Field instead of the Fields() iterator so a cache miss does not
// allocate an iterator backing for every nested struct level.
func noPointersWalk(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array:
		return noPointersWalk(t.Elem())
	case reflect.Struct:
		// NumField/Field, NOT the Fields() range-over-func iterator: the iterator
		// allocates a heap closure+state per call, and this walk runs per decode
		// (columnar reuse gate) — a modernize pass reintroduced that alloc here.
		for i := range t.NumField() {
			if !noPointersWalk(t.Field(i).Type) {
				return false
			}
		}
		return true
	default:
		// string, slice, map, ptr, interface, chan, func, unsafe.Pointer.
		return false
	}
}

// tightPODWalk reports whether t is pointer-free AND has no internal or tail
// padding, so its full byte span is content (and may be bulk-hashed in the delta
// value fingerprint). Scalars are tight by definition. An array is tight iff its
// element is tight (the array stride already includes the element's own padding).
// A struct is tight iff it is pointer-free, every field type is tight, and the
// fields are packed with no gaps and no tail pad: field[0] at offset 0, each
// subsequent field immediately after the previous, and the struct size equal to
// the end of the last field. An empty struct is tight (zero-length span). Any
// type with a pointer (string/slice/map/ptr/iface/chan/func) is not tight.
func tightPODWalk(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array:
		return tightPODWalk(t.Elem())
	case reflect.Struct:
		n := t.NumField()
		if n == 0 {
			return true
		}
		var next uintptr
		for i := range n {
			f := t.Field(i)
			if f.Offset != next || !tightPODWalk(f.Type) {
				return false
			}
			next = f.Offset + f.Type.Size()
		}
		return t.Size() == next
	default:
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
	if d.recycledMaps > 0 {
		t := reflect.TypeFor[map[K]V]()
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			d.recycledMaps--
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
	if d.recycledMaps > 0 {
		t := reflect.TypeFor[map[K]V]()
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			d.recycledMaps--
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
// analog of reuseOrMakeMap, used by decodeMap. Zero-cost (just a nil check)
// when there is nothing to reuse.
func reuseOrMakeMapReflect(d *Decoder, t reflect.Type, n int, p unsafe.Pointer) {
	if *(*unsafe.Pointer)(p) != nil { // direct-target reuse: existing map at p
		reflect.NewAt(t, p).Elem().Clear() // clear keeps the map, drops entries
		return
	}
	if d.recycledMaps > 0 {
		if lst := d.mapFreeList[t]; len(lst) > 0 {
			ptr := lst[len(lst)-1]
			d.mapFreeList[t] = lst[:len(lst)-1]
			d.recycledMaps--
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

// maxRetainedRecycledMaps bounds the free-list capacity a pooled decoder keeps
// across resets. Below it the backing array survives, so the next harvest
// appends into it instead of growing a fresh slice; above it the list is
// released, because a one-off wide decode should not leave its peak footprint on
// a decoder that returns to the pool.
//
// 4096 pointers is 32 KB per map type. It sits between the two numbers that
// matter: a 1000-row telemetry batch grows its list to 1023, so ordinary batches
// keep their backing and keep the reuse; maxRecycledMaps lets one list reach
// 16384 (128 KB), and nothing near that should survive a return to the pool.
const maxRetainedRecycledMaps = 1 << 12

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
	for i := range oldLen {
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
				d.recycledMaps++
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

// dropRecycledMaps empties the per-type free lists without giving their backing
// arrays back.
//
// clear() on the outer map deletes the keys, and with them the slices held as
// values — so the next harvest appends to a nil slice and allocates one afresh.
// Measured at eleven allocations per decode on the telemetry fixture, which is
// 43% of everything the reuse path still allocates: the machinery that exists to
// avoid allocation was allocating its own bookkeeping every time.
//
// The elements are nilled before the truncation rather than left past len.
// Keeping stale unsafe.Pointers in the backing array would pin the harvested
// maps for the GC — trading an allocation for a leak, which is a worse deal than
// the one being fixed.
func (d *Decoder) dropRecycledMaps() {
	for t, lst := range d.mapFreeList {
		for i := range lst {
			lst[i] = nil
		}
		if cap(lst) > maxRetainedRecycledMaps {
			// A spike-sized list is released rather than kept: maxRecycledMaps
			// lets one list reach 16384 pointers, and holding that 128 KB on a
			// pooled decoder for the life of the pool is what the cap above
			// exists to prevent. Keeping the capacity is an optimisation for the
			// steady state, not a license to pin the peak.
			delete(d.mapFreeList, t)
			continue
		}
		d.mapFreeList[t] = lst[:0]
	}
	d.recycledMaps = 0
}
