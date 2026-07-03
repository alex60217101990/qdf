//go:build qdfdebug || race

package qdf

import "unsafe"

// staleHandleMsg is shared by every debug-build panic that fires on a handle
// resolved after its Batch's slab was recycled (Release nils the Batch's
// slab pointer and bumps batchSlab.epoch — either observation means the
// handle is stale).
const staleHandleMsg = "qdf: stale batch handle (was Release called?)"

// checkEpoch panics if the slab backing a handle is gone (Release nils the
// Batch's slab field, so s is nil here — Go permits calling a pointer-
// receiver method on a nil receiver as long as the body doesn't dereference
// it before the check) or has moved to a later generation (the slab was
// recycled via sync.Pool and reused by an unrelated decode before this
// resolve ran). Called by every Batch resolve method before touching s.buf.
func (s *batchSlab) checkEpoch(want uint32) {
	if s == nil || s.epoch != want {
		panic(staleHandleMsg)
	}
}

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
