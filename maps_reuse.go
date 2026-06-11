package qdf

import "unsafe"

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
