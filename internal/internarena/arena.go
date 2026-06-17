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

// maxChunkBytes caps individual slab growth. The loc-pack format gives each
// chunk a 32-bit offset field (see the locs doc on Arena), so a single chunk
// must never exceed what a uint32 offset can address — otherwise Put would
// truncate the offset at line `uint32(a.off)` and Get would resolve the wrong
// byte range (silent corruption). Doubling growth is unbounded, so without
// this cap a long-lived arena that interns multiple GiB of strings would
// eventually allocate a >4 GiB slab and trip the truncation. Capping well
// below 4 GiB keeps every offset in range; with the 16-bit chunk index
// (65 535 slabs) the arena still addresses ~64 TiB total — far past any real
// payload, and normal workloads never fill even one of these slabs.
const maxChunkBytes = 1 << 30 // 1 GiB

// Arena holds a chain of byte slabs and hands out ids that resolve
// to slices into the active slab.
type Arena struct {
	// off / end are the bump-pointer cursor — but expressed as a
	// byte offset into chunks[cur] (not a raw machine address)
	// so we never round-trip through unsafe.Pointer(uintptr). The
	// chunks slice keeps the slab's backing array GC-rooted; we
	// resolve actual addresses lazily via chunks[cur] on each Put.
	// Same write-barrier savings as the article's uintptr-cursor
	// (no pointer-typed store on the hot path), no go vet warning.
	off uintptr
	end uintptr

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
			// Bytes used in the current chunk = off.
			total += int(a.off)
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
	// off==end==0; even an empty Put needs chunks[0] so the
	// slab access below does not panic.
	if len(a.chunks) == 0 || a.off+n > a.end {
		a.grow(n)
	}
	cur := a.chunks[a.cur]
	offset := uint32(a.off)
	// Extend the slice header to the slab's true capacity so we can
	// write into the uninitialised tail bytes. The underlying array
	// is owned by chunks[a.cur]; we never escape this aliased slice.
	if n > 0 {
		full := unsafe.Slice(unsafe.SliceData(cur), cap(cur))
		copy(full[a.off:a.off+n], s)
	}
	a.off += n

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

// DefaultRetainBytes is the soft cap on total chunk capacity the
// arena keeps across Reset() calls. A burst of unusually long
// payloads grows chunks past this; the next Reset drops the extras
// so a long-running encoder pool does not pin spike memory forever.
// Callers may override via ResetWithLimit.
const DefaultRetainBytes = 256 * 1024 // 256 KiB

// Reset is shorthand for ResetWithLimit(DefaultRetainBytes).
func (a *Arena) Reset() { a.ResetWithLimit(DefaultRetainBytes) }

// ResetWithLimit rolls every cursor back to chunks[0] and clears
// the loc table. If the cumulative capacity of all chunks exceeds
// retainBytes the spike chunks (chunks[1..]) are dropped so the
// arena's resident memory stays bounded. chunks[0] — the small
// lazily-allocated initial slab — is always kept warm for the next
// cycle. retainBytes == 0 disables the cap entirely (keep
// everything).
//
// All ids handed out before Reset become invalid; the caller must
// clear its own id-keyed structures before / atomically with the
// call.
func (a *Arena) ResetWithLimit(retainBytes int) {
	a.locs = a.locs[:0]
	if len(a.chunks) == 0 {
		a.off = 0
		a.end = 0
		a.cur = 0
		return
	}
	// Drop excess chunks past chunks[0] when total capacity has
	// outgrown the soft cap. Keep chunks[0] — it is the lazy
	// initial slab whose size matches the steady-state workload
	// before any spike happened.
	if retainBytes > 0 && len(a.chunks) > 1 {
		total := 0
		for _, c := range a.chunks {
			total += cap(c)
		}
		if total > retainBytes {
			// Clear the dropped slots before truncating so the GC
			// can reclaim them; the slice header otherwise keeps
			// their backing arrays alive.
			for i := 1; i < len(a.chunks); i++ {
				a.chunks[i] = nil
			}
			a.chunks = a.chunks[:1]
		}
	}
	a.cur = 0
	a.off = 0
	a.end = uintptr(cap(a.chunks[0]))
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
			a.off = 0
			a.end = uintptr(cap(c))
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
	// Cap slab size so a chunk offset can never exceed the 32-bit field in
	// the loc-pack format. need is always <= MaxStringLen (65 535), far below
	// the cap, so clamping here never starves a request; it only bounds the
	// doubling growth. Growth past this point adds chunks instead of a giant
	// slab (chunkIdx is 16-bit, so ~64 TiB of headroom remains).
	if size > maxChunkBytes {
		size = maxChunkBytes
	}
	c := make([]byte, 0, size)
	a.chunks = append(a.chunks, c)
	a.cur = len(a.chunks) - 1
	a.off = 0
	a.end = size
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
