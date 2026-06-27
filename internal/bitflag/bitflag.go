// Package bitflag provides tiny generic helpers for bit-flag enums over any
// unsigned integer type. It centralizes the set/test idioms so flag enums (e.g.
// a comparison operator built from less/equal/greater bits) read by intent
// rather than raw `&` / `|` expressions and stay consistent across the codebase.
package bitflag

// Flag is any unsigned integer usable as a bit set.
type Flag interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

// Has reports whether v has ANY of the bits in f set (v & f != 0). For a
// single-bit f this is the usual membership test.
func Has[F Flag](v, f F) bool { return v&f != 0 }

// All reports whether v has ALL of the bits in f set (v & f == f).
func All[F Flag](v, f F) bool { return v&f == f }

// Set returns v with the bits in f set.
func Set[F Flag](v, f F) F { return v | f }

// Clear returns v with the bits in f cleared.
func Clear[F Flag](v, f F) F { return v &^ f }

// Toggle returns v with the bits in f flipped.
func Toggle[F Flag](v, f F) F { return v ^ f }
