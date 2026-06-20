// Package bufpool implements size-classed, sharded byte-slice pools.
//
// Buffers are bucketed by capacity, so a small Get does not receive a
// previously-released multi-MiB slice. The pool stores *[]byte to avoid
// the interface-conversion allocation that putting a []byte directly
// into sync.Pool incurs.
package bufpool

import (
	"sync"
	"sync/atomic"
)

const (
	classSmall  = 1 << 9  // 512 B
	classMedium = 1 << 13 // 8 KiB
	classLarge  = 1 << 16 // 64 KiB
	classHuge   = 1 << 20 // 1 MiB
)

type shard struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool
	huge   sync.Pool
}

// numShards is a power of two so the shard index masks cleanly.
const numShards = 16

var shards [numShards]shard

func init() {
	for i := range shards {
		s := &shards[i]
		s.small.New = func() any { b := make([]byte, 0, classSmall); return &b }
		s.medium.New = func() any { b := make([]byte, 0, classMedium); return &b }
		s.large.New = func() any { b := make([]byte, 0, classLarge); return &b }
		s.huge.New = func() any { b := make([]byte, 0, classHuge); return &b }
	}
}

var rrCounter atomic.Uint32

func shardIndex() uint32 {
	return rrCounter.Add(1) & (numShards - 1)
}

// Get returns a *[]byte with cap >= hint and length 0. Anything larger
// than classHuge is allocated outside the pool.
func Get(hint int) *[]byte {
	s := &shards[shardIndex()]
	var p any
	switch {
	case hint <= classSmall:
		p = s.small.Get()
	case hint <= classMedium:
		p = s.medium.Get()
	case hint <= classLarge:
		p = s.large.Get()
	case hint <= classHuge:
		p = s.huge.Get()
	default:
		b := make([]byte, 0, hint)
		return &b
	}
	b := p.(*[]byte)
	// Put files by cap, so a class pool may hold a buffer whose cap is below
	// this class's hint ceiling (e.g. a caller returned a sub-512 B slice into
	// the small pool). Honor the documented "cap >= hint" contract: if the
	// pooled buffer is too small, drop it and allocate a right-sized one.
	if cap(*b) < hint {
		nb := make([]byte, 0, hint)
		return &nb
	}
	*b = (*b)[:0]
	return b
}

// Put returns b to the pool. Buffers grown past classHuge are dropped so
// the GC can reclaim them.
func Put(b *[]byte) {
	if b == nil || *b == nil {
		return
	}
	c := cap(*b)
	*b = (*b)[:0]
	s := &shards[shardIndex()]
	switch {
	case c <= classSmall:
		s.small.Put(b)
	case c <= classMedium:
		s.medium.Put(b)
	case c <= classLarge:
		s.large.Put(b)
	case c <= classHuge:
		s.huge.Put(b)
	}
}
