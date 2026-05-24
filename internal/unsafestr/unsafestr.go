// Package unsafestr provides zero-copy conversions between string and
// []byte. The returned values share storage with the input and MUST be
// treated as read-only — mutating a []byte returned by Bytes is
// undefined behaviour.
package unsafestr

import "unsafe"

// Bytes returns a []byte aliasing the input string. Read-only.
//
//go:nosplit
func Bytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// String returns a string aliasing the input []byte. Read-only.
//
//go:nosplit
func String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
