// Package unsafestr provides a zero-copy conversion from []byte to
// string. The returned value shares storage with the input and MUST be
// treated as read-only — the input []byte must not be mutated while the
// returned string is in use.
package unsafestr

import "unsafe"

// String returns a string aliasing the input []byte. Read-only.
//
//go:nosplit
func String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
