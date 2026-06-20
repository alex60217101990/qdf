// Package bumparena is a monotonic bump allocator for decoded string bodies.
// It is the reusable core behind the public qdf.Arena: a value is copied into a
// densely-packed block and returned as a string aliasing that copy, so a decode
// with many string fields costs ~one allocation per block instead of one per
// string. Blocks grow geometrically and are allocated without zero-fill.
//
// It does NOT carry the encoder's interning id-table (see internal/internarena);
// decode returns strings directly, never looks values up by id, so the id-table
// would be pure overhead.
package bumparena

import (
	"unsafe"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

const (
	// firstBlock is the first block size; subsequent blocks grow geometrically.
	firstBlock = 512
	// blockCap caps geometric growth so one block (and thus the retention of any
	// single returned string) is bounded.
	blockCap = 1 << 16
)

// Bump is a monotonic bump allocator. The zero value is NOT ready; use New.
// Not safe for concurrent use.
type Bump struct {
	buf  []byte // current block; full blocks are abandoned to the GC
	off  int    // bump cursor into buf
	next int    // size of the next fresh block (geometric growth)
}

// New returns an empty bump allocator. The first AppendStr allocates the first
// block.
func New() Bump { return Bump{next: firstBlock} }

// AppendStr copies b (which must be non-empty) into the arena and returns a
// string aliasing the copy. Successive calls pack contiguously. The rare block
// allocation is split into the //go:noinline appendStrGrow so the hot path
// stays small; AppendStr itself does not currently fit the inline budget (the
// call node for the grow branch keeps it over), but the fast path is a bounds
// check plus a copy — cheap relative to the per-decode allocation it avoids.
func (a *Bump) AppendStr(b []byte) string {
	n := len(b)
	off := a.off
	if off+n > len(a.buf) {
		return a.appendStrGrow(b)
	}
	dst := a.buf[off : off+n]
	a.off = off + n
	copy(dst, b)
	return unsafe.String(unsafe.SliceData(dst), n)
}

// appendStrGrow is the cold path: the value does not fit the current block, so
// allocate a fresh one. Out of line so AppendStr stays small.
//
//go:noinline
func (a *Bump) appendStrGrow(b []byte) string {
	n := len(b)
	// dirtmake: no zero-fill — copy overwrites exactly what we expose. Geometric
	// block growth (like a Vec / std::vector): few allocations on a large epoch,
	// little waste on a small one. The old block is abandoned to the GC, still
	// kept alive by any strings that alias it.
	a.buf = unsafestr.DirtBytes(max(a.next, n))
	if a.next < blockCap {
		a.next *= 2
	}
	dst := a.buf[:n]
	a.off = n
	copy(dst, b)
	return unsafe.String(unsafe.SliceData(dst), n)
}

// Reset rewinds the cursor to reuse the current block. It does NOT clear or
// zero memory; the next AppendStr overwrites it. Callers must ensure every
// string previously returned is dead before calling Reset.
func (a *Bump) Reset() { a.off = 0 }
