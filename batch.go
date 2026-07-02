package qdf

// Str is a pointer-free string handle: an (offset, length) pair into the
// owning Batch's slab. 8 bytes, zero pointers — a struct made of Str/Bytes/
// Time and scalars is GC-noscan. Resolve with (*Batch[T]).Str. A handle is
// only meaningful against the Batch that produced it; after Release the
// bytes it points at may be overwritten (debug/race builds panic instead).
type Str struct{ off, len uint32 }

// Bytes is the []byte analogue of Str.
type Bytes struct{ off, len uint32 }

// Time is a pointer-free time value (time.Time carries *Location, which
// would re-introduce GC scanning). Wire-compatible with the timestamp tag.
type Time struct {
	Sec  int64
	Nsec uint32
}
