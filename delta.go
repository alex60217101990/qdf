package qdf

import "errors"

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
