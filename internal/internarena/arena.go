// Package internarena is a bytes-only bump-pointer allocator for the
// Dense-mode intern table. The encoder used to call strings.Clone on
// every first-occurrence intern, which produces one heap object per
// distinct string and adds GC pressure on streams with thousands of
// unique keys. This arena collapses every interned payload into a
// chain of large slabs and hands out compact integer ids that resolve
// back to byte slices that alias the active slab.
//
// Why hand-roll instead of taking github.com/VoolFI71/go-arena: that
// package targets generic typed allocation (New[T], MakeSlice[T]),
// none of which we use. We need exactly one operation — copy a byte
// payload, return an id — and we want to inline the cursor into our
// encoder state so a Put / Get pair stays a few amortised
// instructions. ~80 LOC beats a v0.0.0-pseudo-version dependency on
// every axis but "did someone else write it for me".
//
// Tricks captured from mcyoung's "Cheating the Reaper in Go"
// (https://mcyoung.xyz/2025/04/21/go-arenas/):
//
//   - cursor as uintptr, not unsafe.Pointer. The store-of-pointer
//     write barrier fires on every Alloc otherwise.
//   - one fixed alignment (8 bytes is overkill for bytes, we drop
//     it — bytes do not need to be aligned). Branchless Put hot path.
//   - chunks []byte slice keeps prior chunks alive across GC even
//     while the cursor (uintptr) cannot itself act as a GC root.
//   - Reset() does not free chunks; the next Put reuses chunks[0]
//     from offset 0. The "6× speedup" pattern from the article's
//     final section. Safe for the encoder pool's reuse cycle — every
//     Reset means the encoder's intern table is logically empty, so
//     handing out the same memory to a new id is correct.
//
// What we do NOT need (and skip):
//
//   - the back-pointer + reflect.StructOf dance. Only required when
//     the arena holds pointer-typed payloads so a pointer into a
//     chunk can keep the whole graph alive. Bytes have no GC scan.
//   - per-size-class sync.Pool, runtime.SetFinalizer. One arena per
//     encState; encoder pool already manages the lifecycle.
//   - one-past-the-end pointer interlock. uintptr cursor sidesteps
//     the issue entirely.
//
// Concurrency: a single Arena is owned by one Encoder. The Encoder
// pool gates concurrent access. The Arena itself is NOT goroutine-
// safe.
//
// Aliasing contract: byte slices returned by Get() alias the chunk
// the id was stored in. A slice stays valid as long as no Reset()
// runs and the encoder does not Put past a chunk-growth boundary
// that retires the chunk the slice points into. In practice the
// only caller is encState, which holds Get() results inside its
// own map keyed by string. Resets clear the map first, then the
// arena, so an aliased string is never read past its arena's life.
package internarena

import (
	"unsafe"
)

// initialChunkBytes is the first slab's capacity. Doubles on every
// growth, matching the runtime's slice-growth heuristic.
const initialChunkBytes = 4 * 1024

// Arena holds a chain of byte slabs and hands out ids that resolve
// to slices into the active slab.
type Arena struct {
	// next / end are the bump-pointer cursor inside chunks[cur].
	// Both are uintptr so writing them does not trigger the runtime
	// write barrier on every Put.
	next uintptr
	end  uintptr

	// cur is the index of the chunk the cursor lives in. Reset()
	// rolls this back to 0.
	cur int

	// chunks owns every slab the arena ever allocated. Each entry
	// is a []byte whose underlying array stays GC-rooted via the
	// slice header. The slice grows by append; doubling.
	chunks [][]byte

	// locs[id] packs (chunk_idx<<48) | (offset<<16) | length so a
	// single uint64 carries every position a caller needs. 16 bits
	// of chunk_idx handles 65 535 slabs (way past practical limits),
	// 32 bits of offset handles 4 GiB chunks, 16 bits of length
	// covers strings up to 65 535 bytes — anything longer falls back
	// to a per-string allocation in the caller.
	locs []uint64
}

// MaxStringLen is the largest single payload the arena can store.
// Callers must handle longer payloads themselves (e.g. by skipping
// the intern table — see encState.lookupOrAssign).
const MaxStringLen = 1<<16 - 1

// Len reports the number of stored ids.
//
//go:nosplit
func (a *Arena) Len() int { return len(a.locs) }

// BytesUsed is the total byte volume the arena has handed out across
// every live chunk. Useful for capacity tuning and debug logs.
func (a *Arena) BytesUsed() int {
	total := 0
	for i, c := range a.chunks {
		if i < a.cur {
			total += cap(c)
			continue
		}
		if i == a.cur {
			// Bytes used in the current chunk = next - start.
			start := uintptr(unsafe.Pointer(unsafe.SliceData(c)))
			total += int(a.next - start)
		}
	}
	return total
}

// Put copies s into the arena and returns the assigned id. Caller
// guarantees len(s) <= MaxStringLen; longer payloads must take the
// fallback path.
//
//go:nosplit
func (a *Arena) Put(s string) uint32 {
	n := uintptr(len(s))
	// Lazy first-chunk allocation. Cold arena has chunks==nil and
	// next==end==0; even an empty Put needs chunks[0] so the
	// chunkStart lookup below does not panic.
	if len(a.chunks) == 0 || a.next+n > a.end {
		a.grow(n)
	}
	// Copy bytes through the cursor.
	dst := unsafe.Slice((*byte)(unsafe.Pointer(a.next)), n)
	copy(dst, s)
	// Compute offset within the current chunk for the loc record.
	cur := a.chunks[a.cur]
	chunkStart := uintptr(unsafe.Pointer(unsafe.SliceData(cur)))
	offset := uint32(a.next - chunkStart)
	a.next += n

	id := uint32(len(a.locs))
	a.locs = append(a.locs, pack(uint16(a.cur), offset, uint16(n)))
	return id
}

// Get returns the bytes for id. The slice aliases the chunk the
// payload lives in — caller MUST NOT retain it across a Reset() or
// a Put() that triggers chunk growth.
//
//go:nosplit
func (a *Arena) Get(id uint32) []byte {
	chunkIdx, offset, length := unpack(a.locs[id])
	chunk := a.chunks[chunkIdx]
	return chunk[offset : offset+uint32(length)]
}

// Reset rolls every cursor back to the start of chunks[0] without
// freeing any chunk. Subsequent Put calls overwrite the prior
// contents. All ids handed out before Reset become invalid; the
// caller must clear its own id-keyed structures before / atomically
// with the call.
func (a *Arena) Reset() {
	a.locs = a.locs[:0]
	if len(a.chunks) == 0 {
		a.next = 0
		a.end = 0
		a.cur = 0
		return
	}
	a.cur = 0
	first := a.chunks[0]
	a.next = uintptr(unsafe.Pointer(unsafe.SliceData(first)))
	a.end = a.next + uintptr(cap(first))
}

// grow advances the cursor to a chunk that can fit need bytes,
// either by walking to the next existing chunk (post-Reset reuse) or
// by allocating a fresh one (doubling growth).
func (a *Arena) grow(need uintptr) {
	// Try to walk to an existing chunk that can hold the request.
	// After Reset, chunks past cur still exist; reuse them before
	// allocating new memory.
	for a.cur+1 < len(a.chunks) {
		a.cur++
		c := a.chunks[a.cur]
		if uintptr(cap(c)) >= need {
			a.next = uintptr(unsafe.Pointer(unsafe.SliceData(c)))
			a.end = a.next + uintptr(cap(c))
			return
		}
	}
	// Allocate a fresh chunk. Doubling growth, but at least `need`.
	size := uintptr(initialChunkBytes)
	if len(a.chunks) > 0 {
		size = uintptr(cap(a.chunks[len(a.chunks)-1])) * 2
	}
	if size < need {
		size = need
	}
	// Round up to the nearest power of two for slab-friendly sizes.
	size = nextPow2(size)
	c := make([]byte, 0, size)
	a.chunks = append(a.chunks, c)
	a.cur = len(a.chunks) - 1
	a.next = uintptr(unsafe.Pointer(unsafe.SliceData(c)))
	a.end = a.next + size
}

// pack folds (chunk, offset, length) into a single uint64.
//
//go:nosplit
func pack(chunkIdx uint16, offset uint32, length uint16) uint64 {
	return uint64(chunkIdx)<<48 | uint64(offset)<<16 | uint64(length)
}

// unpack reverses pack.
//
//go:nosplit
func unpack(loc uint64) (chunkIdx uint16, offset uint32, length uint16) {
	chunkIdx = uint16(loc >> 48)
	offset = uint32(loc>>16) & 0xFFFFFFFF
	length = uint16(loc)
	return
}

// nextPow2 returns the smallest power of two ≥ n. Used to keep chunk
// caps on slab-friendly sizes.
//
//go:nosplit
func nextPow2(n uintptr) uintptr {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}
