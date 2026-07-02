//go:build !qdfdebug && !race

package qdf

import "unsafe"

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
