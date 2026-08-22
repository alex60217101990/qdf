package unsafestr

import "unsafe" // also required by the go:linkname directive below

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ unsafe.Pointer, needzero bool) unsafe.Pointer

// DirtBytes allocates a byte slice of length n WITHOUT zeroing the memory
// (runtime.mallocgc with needzero=false). The caller MUST write every byte
// before it is read. Used for bump arenas immediately overwritten by a copy,
// where the zero-fill is pure waste. Mirrors bytedance/gopkg/lang/dirtmake;
// the Go runtime's own string(b) conversion (rawstring) allocates the same way.
func DirtBytes(n int) []byte {
	if n == 0 {
		return nil
	}
	p := mallocgc(uintptr(n), nil, false)
	return unsafe.Slice((*byte)(p), n)
}
