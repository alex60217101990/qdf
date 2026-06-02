// Package intern provides a small fixed-size string cache used by the
// decoder to deduplicate repeated map keys.
//
// The cache is a direct-mapped table indexed by a hash of the input
// bytes. A hit returns the previously-allocated string with no
// allocation. A miss copies the input once and stores it; collisions
// overwrite the slot rather than chain.
//
// The table is a value type embedded in the Decoder, so it survives
// across Unmarshal calls when the decoder is reused via sync.Pool.
package intern

import (
	"hash/maphash"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Cache is a 256-slot direct-mapped string interner. The zero value is
// ready to use. Lookups are a hash + one read + one comparison. The 256-slot
// table (~4 KiB on 64-bit) lives behind a pointer and is allocated lazily on
// the first Make, so an unused Cache costs ~24 bytes inside its owner — this
// matters for decoders that are constructed per nested value (codegen) and
// never touch a map key. Once allocated it stays in L1 across reuse.
type Cache struct {
	slots *[256]string
	seed  maphash.Seed
	init  bool
}

func (c *Cache) ensure() {
	if !c.init {
		c.seed = maphash.MakeSeed()
		c.slots = new([256]string)
		c.init = true
	}
}

// Make returns a string equal to b. On a cache hit the existing string
// is returned with no allocation. On a miss b is copied into a fresh
// string and stored. The returned string is safe to retain — it does
// not alias b.
//
//go:nosplit
func (c *Cache) Make(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	c.ensure()
	h := maphash.Bytes(c.seed, b)
	slot := h & (uint64(len(c.slots)) - 1)
	if existing := c.slots[slot]; existing != "" && existing == unsafestr.String(b) {
		return existing
	}
	s := string(b)
	c.slots[slot] = s
	return s
}
