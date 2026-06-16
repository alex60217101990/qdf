package qdf

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"
	"weak"

	"github.com/alex60217101990/qdf/internal/reflectutil"
)

var (
	// ErrBaselineRequired is returned by (*BaselineRegistry).Apply when the
	// patch carries no base fingerprint (the producer used
	// OptDeltaNoBaseFingerprint), so its baseline cannot be content-addressed.
	ErrBaselineRequired = errors.New("qdf: patch has no base fingerprint; baseline registry requires one")
	// ErrBaselineEvicted is returned by (*BaselineRegistry).Apply when the
	// patch's baseline id is not resolvable — it was never registered, or the
	// caller dropped its strong reference and the GC reclaimed it.
	ErrBaselineEvicted = errors.New("qdf: patch baseline not found in registry (never registered or GC-reclaimed)")
)

// BaselineRegistry is a consumer-side, content-addressed cache of recent
// baselines of type T, used to apply a chain of patches in a state-sync stream
// without threading the previous value by hand. It maps each baseline's content
// fingerprint (the same value Diff embeds as baseFP) to a weak.Pointer, so the
// GC reclaims any baseline the caller no longer references. The registry never
// pins a baseline alive.
//
// A BaselineRegistry is safe for concurrent use.
type BaselineRegistry[T any] struct {
	mu sync.Mutex
	m  map[uint64]weak.Pointer[T]
	td *typeDesc
}

// NewBaselineRegistry returns an empty registry for baselines of type T.
func NewBaselineRegistry[T any]() *BaselineRegistry[T] {
	td, _ := descOf(reflect.TypeFor[T]())
	return &BaselineRegistry[T]{m: make(map[uint64]weak.Pointer[T]), td: td}
}

// Len reports the number of live (non-evicted) baselines, pruning dead weak
// entries as a side effect.
func (r *BaselineRegistry[T]) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, wp := range r.m {
		if wp.Value() == nil {
			delete(r.m, id)
		}
	}
	return len(r.m)
}

// deepClone returns a structural deep copy of *v. It walks the value by its
// *typeDesc over unsafe.Pointer — the same idiom as fpHash / applyValue /
// equalValue — rather than reflect.Value, so it shares the package's fast paths
// and honors the qdf_reflect2 build tag (slice/map backing is allocated through
// reflectutil, exactly as the decode path does).
//
// The clone is byte-for-byte equivalent in everything the fingerprint reads, so
// it always fingerprints identically to its source — the invariant Apply's
// baseFP check relies on. It preserves nil-vs-empty (a nil slice/map stays nil,
// an empty non-nil one stays empty non-nil). Containers (slice/map/pointer) are
// freshly allocated so Apply's in-place mutation of the clone never touches the
// registered baseline.
func deepClone[T any](v *T) *T {
	out := new(T)
	td, err := descOf(reflect.TypeFor[T]())
	if err != nil || td == nil {
		// Unsupported type: descOf failing means no patch can target it, so the
		// clone is unreachable through Apply in practice. A plain value copy is the
		// safe best effort.
		*out = *v
		return out
	}
	cloneValue(td, unsafe.Pointer(out), unsafe.Pointer(v), 0)
	return out
}

// cloneValue writes a deep copy of the value of type td at src into dst,
// mirroring fpHash's structural walk so a clone always fingerprints identically
// to its source. A pointer-free (pod) span is bulk-copied in one byte copy — no
// GC write barrier needed, since it holds no pointers. Every pointer-bearing
// write goes through a typed store (string, unsafe.Pointer) or reflect.Set, so
// the GC write barrier fires; a raw byte copy is never used over a span that
// contains pointers.
func cloneValue(td *typeDesc, dst, src unsafe.Pointer, depth int) {
	if depth > maxDeltaDepth {
		// Mirror the diff/apply/fingerprint walks: stop instead of overflowing the
		// (uncatchable) stack on a cyclic or pathologically deep structure. A
		// truncated clone fingerprints differently, so Apply's baseFP check then
		// rejects it with ErrPatchBaseMismatch — a clean error, never a crash.
		return
	}
	if td.pod {
		// Pointer-free: one byte copy reproduces every byte (scalars, tight
		// arrays/structs, padded pointer-free structs) and is GC-safe.
		sz := td.rType.Size()
		copy(unsafe.Slice((*byte)(dst), sz), unsafe.Slice((*byte)(src), sz))
		return
	}
	switch td.kind {
	case reflect.String:
		*(*string)(dst) = *(*string)(src) // typed string store: GC-barriered
	case reflect.Slice:
		cloneSlice(td, dst, src, depth)
	case reflect.Array:
		cloneArray(td, dst, src, depth)
	case reflect.Struct:
		// Mirror fpHash's struct dispatch. A struct the delta layer treats as
		// fields-less (len(td.fields)==0: time.Time, a custom Marshaler, or a
		// degenerate struct with no qdf-visible fields) is hashed as a whole — by
		// its instant, its marshaled form, or via fpHashReflect over its real
		// fields. Cloning such a struct field-by-field would skip its unexported
		// fields (which is ALL of time.Time's wall/ext/loc), zeroing it and
		// producing a clone whose fingerprint differs → ErrPatchBaseMismatch on a
		// legitimate baseline. A typed value copy reproduces every field, is
		// GC-safe, and shares no mutable structure Apply would touch in place
		// (Apply replaces such a value wholesale via the codec).
		if len(td.fields) == 0 {
			reflect.NewAt(td.rType, dst).Elem().Set(reflect.NewAt(td.rType, src).Elem())
			return
		}
		for i := range td.fields {
			f := &td.fields[i]
			cloneValue(f.desc, unsafe.Add(dst, f.offset), unsafe.Add(src, f.offset), depth+1)
		}
	case reflect.Pointer:
		sp := *(*unsafe.Pointer)(src)
		if sp == nil {
			return // nil pointer: dst already nil
		}
		elem := td.elem
		if elem == nil {
			elem, _ = descOf(td.rType.Elem())
			if elem == nil {
				return
			}
		}
		np := reflect.New(elem.rType) // GC-safe allocation of the pointee
		cloneValue(elem, np.UnsafePointer(), sp, depth+1)
		reflect.NewAt(td.rType, dst).Elem().Set(np) // typed pointer store: barriered
	case reflect.Map:
		cloneMap(td, dst, src, depth)
	case reflect.Interface:
		iv := reflect.NewAt(td.rType, src).Elem()
		if iv.IsNil() {
			return
		}
		ev := iv.Elem() // dynamic value
		edesc, derr := descOf(ev.Type())
		if derr != nil || edesc == nil {
			reflect.NewAt(td.rType, dst).Elem().Set(iv) // fallback: shallow box copy
			return
		}
		srcBuf := reflect.New(ev.Type())
		srcBuf.Elem().Set(ev) // addressable copy of the boxed value
		dstBuf := reflect.New(ev.Type())
		cloneValue(edesc, dstBuf.UnsafePointer(), srcBuf.UnsafePointer(), depth+1)
		reflect.NewAt(td.rType, dst).Elem().Set(dstBuf.Elem()) // box the clone
	default:
		// Exotic kinds (chan/func): a typed value copy (barriered, shallow).
		reflect.NewAt(td.rType, dst).Elem().Set(reflect.NewAt(td.rType, src).Elem())
	}
}

// cloneSlice deep-copies the slice at src into a freshly allocated backing at
// dst, preserving nil-vs-empty.
func cloneSlice(td *typeDesc, dst, src unsafe.Pointer, depth int) {
	sh := (*sliceHeader)(src)
	if sh.Data == nil {
		return // nil slice stays nil (dst already zeroed)
	}
	n := sh.Len
	reflectutil.MakeSlice(td.rType, n, dst) // reflect2-aware; non-nil even for n==0
	if n == 0 {
		return // empty-non-nil preserved
	}
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
		if elem == nil {
			return
		}
	}
	dsh := (*sliceHeader)(dst)
	stride := td.rType.Elem().Size()
	if elem.pod {
		// Pointer-free elements: one bulk byte copy (GC-safe, holds no pointers).
		copy(unsafe.Slice((*byte)(dsh.Data), uintptr(n)*stride),
			unsafe.Slice((*byte)(sh.Data), uintptr(n)*stride))
		return
	}
	for i := range n {
		cloneValue(elem, unsafe.Add(dsh.Data, uintptr(i)*stride),
			unsafe.Add(sh.Data, uintptr(i)*stride), depth+1)
	}
}

// cloneArray deep-copies a non-pod array (a pod array took the byte-copy fast
// path) element by element.
func cloneArray(td *typeDesc, dst, src unsafe.Pointer, depth int) {
	n := td.rType.Len()
	elem := td.elem
	if elem == nil {
		elem, _ = descOf(td.rType.Elem())
		if elem == nil {
			return
		}
	}
	stride := td.rType.Elem().Size()
	for i := range n {
		cloneValue(elem, unsafe.Add(dst, uintptr(i)*stride),
			unsafe.Add(src, uintptr(i)*stride), depth+1)
	}
}

// cloneMap deep-copies the map at src into a freshly allocated map at dst,
// preserving nil-vs-empty. Keys are shared (a map key is immutable; Apply never
// mutates one); values are deep-cloned, because Apply can mutate a value's inner
// containers in place and must not reach through to the registered baseline.
func cloneMap(td *typeDesc, dst, src unsafe.Pointer, depth int) {
	mv := reflect.NewAt(td.rType, src).Elem()
	if mv.IsNil() {
		return // nil map stays nil
	}
	// Allocate through reflectutil so qdf_reflect2 swaps in the faster allocator,
	// matching applyMap / the decode path.
	reflectutil.MakeMap(td.rType, mv.Len(), dst)
	dmv := reflect.NewAt(td.rType, dst).Elem()
	valDesc := td.elem
	if valDesc == nil {
		valDesc, _ = descOf(td.rType.Elem())
	}
	if valDesc == nil {
		dmv.Set(mv) // unsupported value type: shallow copy (still a fresh map)
		return
	}
	valType := td.rType.Elem()
	for it := mv.MapRange(); it.Next(); {
		srcBuf := reflect.New(valType)
		srcBuf.Elem().Set(it.Value()) // addressable copy of the value
		dstBuf := reflect.New(valType)
		cloneValue(valDesc, dstBuf.UnsafePointer(), srcBuf.UnsafePointer(), depth+1)
		dmv.SetMapIndex(it.Key(), dstBuf.Elem())
	}
}

// Register stores v as a resolvable baseline and returns its content id — the
// same fingerprint Diff embeds as baseFP. The registry holds v through a
// weak.Pointer; the CALLER must keep v reachable for it to remain resolvable.
func (r *BaselineRegistry[T]) Register(v *T) uint64 {
	id := r.fingerprint(v)
	r.mu.Lock()
	r.m[id] = weak.Make(v)
	r.mu.Unlock()
	return id
}

// fingerprint computes the content id of *v (identical to Diff's baseFP).
func (r *BaselineRegistry[T]) fingerprint(v *T) uint64 {
	if r.td == nil {
		return 0 // unsupported type; such a T can never produce a patch either
	}
	return valueFingerprint(r.td, unsafe.Pointer(v))
}

// Apply resolves the patch's baseline (by its embedded baseFP) to a *T, applies
// the patch onto a fresh deep copy of it, registers the result as a new
// baseline (so it can base the next patch in the stream), and returns the new
// *T. The caller should keep the returned pointer to chain further patches off
// it; dropping it lets the GC reclaim it.
//
// Errors: ErrBaselineRequired (the patch carries no baseFP — produced with
// OptDeltaNoBaseFingerprint), ErrBaselineEvicted (the baseline id is not
// resolvable), plus any error from the underlying Apply.
//
// Because baselines are content-addressed by a 64-bit fingerprint, two distinct
// values that happen to share a fingerprint (probability ~N²/2⁶⁴, negligible
// for thousands of live baselines) collide on the same map key, and the later
// registration overwrites the earlier one; the earlier baseline then resolves
// as ErrBaselineEvicted even while the caller still holds it.
func (r *BaselineRegistry[T]) Apply(patch []byte) (*T, error) {
	h, _, err := readPatchHeader(patch)
	if err != nil {
		return nil, err
	}
	if h.flags&flagPatchBaseFP == 0 {
		return nil, ErrBaselineRequired
	}
	r.mu.Lock()
	wp, ok := r.m[h.baseFP]
	r.mu.Unlock()
	var base *T
	if ok {
		base = wp.Value()
	}
	if base == nil {
		if ok {
			r.mu.Lock()
			if cur, still := r.m[h.baseFP]; still && cur.Value() == nil {
				delete(r.m, h.baseFP)
			}
			r.mu.Unlock()
		}
		return nil, ErrBaselineEvicted
	}
	clone := deepClone(base)
	if err := Apply(clone, patch); err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.m[r.fingerprint(clone)] = weak.Make(clone)
	r.mu.Unlock()
	return clone, nil
}
