package qdf

import (
	"errors"
	"hash/maphash"
	"reflect"
)

// schemaSeed is a process-stable seed. A fingerprint only needs to be stable
// within a single producer→consumer exchange that shares this binary; cross-
// build stability is not required for Phase 1 (both ends import the same type).
var schemaSeed = maphash.MakeSeed()

// schemaFingerprint hashes a type descriptor's shape (kind + field names +
// recursive field/element kinds) so Apply can reject a patch built for a
// different type. Cycles are broken by a visited set.
func schemaFingerprint(td *typeDesc) uint64 {
	var h maphash.Hash
	h.SetSeed(schemaSeed)
	visited := map[*typeDesc]bool{}
	hashDesc(&h, td, visited)
	return h.Sum64()
}

func hashDesc(h *maphash.Hash, td *typeDesc, visited map[*typeDesc]bool) {
	if td == nil {
		h.WriteByte(0xFF)
		return
	}
	if visited[td] {
		h.WriteByte(0xFE) // cycle marker
		return
	}
	visited[td] = true
	h.WriteByte(byte(td.kind))
	h.WriteByte(td.marshalerKind)
	for i := range td.fields {
		f := &td.fields[i]
		h.WriteString(f.name)
		hashDesc(h, f.desc, visited)
	}
	if td.elem != nil {
		hashDesc(h, td.elem, visited)
	}
}

// _ keeps the reflect import used (reflect.Kind is used via td.kind which is
// reflect.Kind; the import is needed for the type to resolve in this file).
var _ reflect.Kind

var (
	// ErrInvalidPatch is returned when a patch blob is truncated, has a bad
	// magic/version, or is otherwise malformed.
	ErrInvalidPatch = errors.New("qdf: invalid or truncated patch")
	// ErrPatchSchemaMismatch is returned by Apply when the patch was built for
	// a different type than the supplied base.
	ErrPatchSchemaMismatch = errors.New("qdf: patch schema fingerprint mismatch")
	// ErrPatchBaseMismatch is returned by Apply when the patch carries a base
	// fingerprint that does not match the supplied base value.
	ErrPatchBaseMismatch = errors.New("qdf: patch base fingerprint mismatch")
)
