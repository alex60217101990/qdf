package qdf

import "github.com/alex60217101990/qdf/internal/bumparena"

// Arena is a bump allocator for decoded string (and []byte-as-string) bodies.
// Pass it to a decode via WithArena (or Decoder.SetArena) and every copied
// string is packed into the arena's dense, contiguous blocks instead of one
// heap allocation per string. Across an epoch of many decodes this amortizes
// the block allocation to near zero and keeps the strings cache-local; the GC
// scans each block as a single object instead of one object per string.
//
// Lifetime / safety: strings returned by an arena decode ALIAS the arena's
// memory. They stay valid as long as the Arena (or any string from it) is
// reachable — the GC keeps a block alive from an interior pointer, so you need
// not track the buffer manually. The SAFE pattern is one Arena per epoch
// (request / batch / stream window), then drop it:
//
//	a := qdf.NewArena()
//	for _, msg := range batch {
//	    var v Event
//	    _ = qdf.Unmarshal(msg, &v, qdf.WithArena(a))
//	    use(v)          // v's strings live in a
//	}
//	// drop a; the GC frees every block once the decoded values die.
//
// Reset reuses the arena across epochs for zero allocation, but is UNSAFE while
// any string from the previous epoch is still live (see Reset).
//
// An Arena is NOT safe for concurrent use; give each goroutine its own.
type Arena struct {
	b bumparena.Bump
}

// NewArena returns an empty arena. The first decode allocates the first block.
func NewArena() *Arena { return &Arena{b: bumparena.New()} }

// appendStr copies b (non-empty) into the arena and returns an aliasing string.
// Trivial forwarder so the decoder's hot path resolves directly to the bump
// core (this wrapper inlines).
func (a *Arena) appendStr(b []byte) string { return a.b.AppendStr(b) }

// Reset rewinds the arena to reuse its current block, so the next epoch of
// decodes allocates nothing.
//
// UNSAFE: Reset invalidates every string previously returned from this arena —
// the next decode overwrites the same memory. Call it ONLY once every value
// decoded into the arena since the last Reset (or since NewArena) is dead. This
// is a manual use-after-free contract the race detector cannot catch; if unsure,
// drop the Arena and make a new one instead (the GC then frees it safely).
func (a *Arena) Reset() { a.b.Reset() }
