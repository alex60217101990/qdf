//go:build !qdfdebug && !race

package qdf

import "unsafe"

// checkEpoch is a no-op in production builds: resolving a handle after
// Release is documented UB (the slab is pooled and may already be reused by
// an unrelated decode), not a checked error. The empty body on a
// pointer-receiver method with an unused parameter compiles to nothing and
// is inlined away — no branch, no load, zero cost. See batch_check_debug.go
// for the panicking counterpart.
func (s *batchSlab) checkEpoch(_ uint32) {}

// str resolves a handle with no checks: 2 loads and an unsafe.String.
func (s *batchSlab) str(h Str) string {
	if h.len == 0 {
		return ""
	}
	return unsafe.String((*byte)(unsafe.Add(s.base(), h.off)), int(h.len))
}

func (s *batchSlab) bytes(h Bytes) []byte {
	if h.len == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Add(s.base(), h.off)), int(h.len))
}
