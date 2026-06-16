package qdf

import (
	"errors"
	"reflect"
	"sync"
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
}

// NewBaselineRegistry returns an empty registry for baselines of type T.
func NewBaselineRegistry[T any]() *BaselineRegistry[T] {
	return &BaselineRegistry[T]{m: make(map[uint64]weak.Pointer[T])}
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
// would otherwise reject the cloned baseline. Unexported struct fields are not
// supported (consistent with qdf encoding) and are left zero.
func deepClone[T any](v *T) *T {
	out := new(T)
	cloneValue(reflect.ValueOf(out).Elem(), reflect.ValueOf(v).Elem())
	return out
}

func cloneValue(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return
		}
		np := reflect.New(src.Type().Elem())
		cloneValue(np.Elem(), src.Elem())
		dst.Set(np)
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		n := src.Len()
		ns := reflect.MakeSlice(src.Type(), n, n)
		for i := range n {
			cloneValue(ns.Index(i), src.Index(i))
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
			cloneValue(k, it.Key())
			val := reflect.New(src.Type().Elem()).Elem()
			cloneValue(val, it.Value())
			nm.SetMapIndex(k, val)
		}
		dst.Set(nm)
	case reflect.Array:
		for i := 0; i < src.Len(); i++ {
			cloneValue(dst.Index(i), src.Index(i))
		}
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			df := dst.Field(i)
			if !df.CanSet() {
				continue
			}
			cloneValue(df, src.Field(i))
		}
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		ev := src.Elem()
		nv := reflect.New(ev.Type()).Elem()
		cloneValue(nv, ev)
		dst.Set(nv)
	default:
		dst.Set(src)
	}
}
