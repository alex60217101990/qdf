package qdf

import (
	"errors"
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
