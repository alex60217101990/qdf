//go:build qdfdebug || race

package qdf

import "unsafe"

// Debug/race resolve: bounds-check the handle against the live slab so a
// stale handle (kept across Release) or a corrupted one panics loudly
// instead of returning silent garbage.
func (s *batchSlab) str(h Str) string {
	if h.len == 0 {
		return ""
	}
	if int(h.off)+int(h.len) > len(s.buf) {
		panic("qdf: stale or out-of-range Str handle (was Release called?)")
	}
	return unsafe.String((*byte)(unsafe.Add(s.base(), h.off)), int(h.len))
}

func (s *batchSlab) bytes(h Bytes) []byte {
	if h.len == 0 {
		return nil
	}
	if int(h.off)+int(h.len) > len(s.buf) {
		panic("qdf: stale or out-of-range Bytes handle (was Release called?)")
	}
	return unsafe.Slice((*byte)(unsafe.Add(s.base(), h.off)), int(h.len))
}
