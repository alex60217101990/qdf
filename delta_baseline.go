package qdf

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"
	"weak"
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

// deepClone returns a structural deep copy of *v: scalars/strings/[N]byte
// copied by value; slices/maps/pointers/interfaces recursively allocated and
// copied. It preserves nil-vs-empty (a nil slice/map stays nil, an empty
// non-nil one stays empty non-nil) so the clone's fingerprint equals the
// original's — which is correctness-critical, because Apply's baseFP check
// would otherwise reject the cloned baseline.
//
// Struct cloning mirrors how fpHash hashes the struct, so the two never
// disagree: a struct with qdf-visible fields is cloned field-by-field (its
// unexported fields are left zero, which is safe because the fingerprint, like
// the wire, ignores them); a fields-less struct — time.Time, a custom
// Marshaler, or a struct with no exported fields — is copied whole by value,
// because the fingerprint reads its real contents (the instant, the marshaled
// form, or all fields) and a field-by-field clone would zero the unexported
// internals (time.Time is entirely unexported) and fingerprint differently.
func deepClone[T any](v *T) *T {
	out := new(T)
	cloneValue(reflect.ValueOf(out).Elem(), reflect.ValueOf(v).Elem(), 0)
	return out
}

func cloneValue(dst, src reflect.Value, depth int) {
	if depth > maxDeltaDepth {
		// Mirror the diff/apply/fingerprint walks: stop instead of overflowing the
		// (uncatchable) stack on a cyclic or pathologically deep structure. A
		// truncated clone fingerprints differently, so Apply's baseFP check then
		// rejects it with ErrPatchBaseMismatch — a clean error, never a crash.
		return
	}
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return
		}
		np := reflect.New(src.Type().Elem())
		cloneValue(np.Elem(), src.Elem(), depth+1)
		dst.Set(np)
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		n := src.Len()
		ns := reflect.MakeSlice(src.Type(), n, n)
		for i := range n {
			cloneValue(ns.Index(i), src.Index(i), depth+1)
		}
		dst.Set(ns)
	case reflect.Map:
		if src.IsNil() {
			return
		}
		nm := reflect.MakeMapWithSize(src.Type(), src.Len())
		it := src.MapRange()
		for it.Next() {
			k := reflect.New(src.Type().Key()).Elem()
			cloneValue(k, it.Key(), depth+1)
			val := reflect.New(src.Type().Elem()).Elem()
			cloneValue(val, it.Value(), depth+1)
			nm.SetMapIndex(k, val)
		}
		dst.Set(nm)
	case reflect.Array:
		for i := range src.Len() {
			cloneValue(dst.Index(i), src.Index(i), depth+1)
		}
	case reflect.Struct:
		// Mirror fpHash's struct dispatch. A struct the delta layer treats as
		// fields-less (len(td.fields)==0: time.Time, a custom Marshaler, or a
		// degenerate struct with no qdf-visible fields) is hashed as a whole — by
		// its instant, its marshaled form, or via fpHashReflect over its real
		// fields. Cloning such a struct field-by-field would skip its unexported
		// fields (which is ALL of time.Time's wall/ext/loc), zeroing it and
		// producing a clone whose fingerprint differs → ErrPatchBaseMismatch on a
		// legitimate baseline. Copy it by value instead, which reproduces every
		// field. (A shallow value copy is safe: Apply replaces such a value
		// wholesale via the codec, never mutates its internals in place.)
		if td, err := descOf(src.Type()); err == nil && len(td.fields) == 0 {
			dst.Set(src)
			return
		}
		for i := range src.NumField() {
			df := dst.Field(i)
			if !df.CanSet() {
				continue
			}
			cloneValue(df, src.Field(i), depth+1)
		}
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		ev := src.Elem()
		nv := reflect.New(ev.Type()).Elem()
		cloneValue(nv, ev, depth+1)
		dst.Set(nv)
	default:
		dst.Set(src)
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
