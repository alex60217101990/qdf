package qdf

import (
	"reflect"
	"time"
	"unsafe"
)

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

// Batch is a pointer-free decode result: Rows carries handle/scalar structs
// (GC never scans them), the slab owns every byte the handles reference.
type Batch[T any] struct {
	Rows []T
	slab *batchSlab
}

// UnmarshalBatch decodes data into a pointer-free Batch. T must contain only
// scalars, qdf.Str, qdf.Bytes and qdf.Time fields (see batchPlanOf); string
// bodies land in one pooled slab. Call Release when done to recycle the slab
// and the Rows backing — afterwards Rows and every handle are invalid.
//
// Batch is returned BY VALUE (4 words: a slice header + a slab pointer): the
// generic Batch[T] header itself cannot be pooled (its type is instantiated
// per T), so the only way to avoid an allocation for it on every decode is to
// never heap-allocate it in the first place. Returned by value it is built on
// the caller's stack (or inlined into their struct); (*Batch[T]).Release
// stays a pointer receiver so it can still nil out the fields of an
// addressable local (`b, _ := UnmarshalBatch[T](...); defer b.Release()`).
func UnmarshalBatch[T any](data []byte, opts ...QueryOption) (Batch[T], error) {
	plan, err := batchPlanOf(reflect.TypeFor[T]())
	if err != nil {
		return Batch[T]{}, err
	}
	slab := newBatchSlab()
	var rowsPtr unsafe.Pointer
	takeRows := func(n int) unsafe.Pointer {
		rowsPtr = slab.takeRows(n * int(plan.stride))
		return rowsPtr
	}
	n, err := unmarshalBatchCore(data, plan, slab, takeRows, opts...)
	if err != nil {
		slab.release()
		return Batch[T]{}, err
	}
	var rows []T
	if n > 0 {
		rows = unsafe.Slice((*T)(rowsPtr), n)
	}
	return Batch[T]{Rows: rows, slab: slab}, nil
}

// Str resolves a string handle produced by this batch's decode.
func (b *Batch[T]) Str(h Str) string { return b.slab.str(h) }

// BytesOf resolves a bytes handle. The view aliases the slab: valid until Release.
func (b *Batch[T]) BytesOf(h Bytes) []byte { return b.slab.bytes(h) }

// TimeOf converts a pointer-free Time to time.Time (UTC).
func (b *Batch[T]) TimeOf(h Time) time.Time {
	return time.Unix(h.Sec, int64(h.Nsec)).UTC()
}

// Release recycles the slab. Rows and all handles become invalid.
func (b *Batch[T]) Release() {
	if b.slab != nil {
		b.slab.release()
		b.slab = nil
	}
	b.Rows = nil
}
