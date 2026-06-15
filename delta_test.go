package qdf

import (
	"reflect"
	"testing"
)

func TestPatchHeaderRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   byte
		schema  uint64
		base    uint64
		hasBase bool
	}{
		{"no-base", 0, 0x1122334455667788, 0, false},
		{"with-base", flagPatchBaseFP, 0xDEADBEEFCAFEBABE, 0x0102030405060708, true},
		{"rans+base", flagPatchBaseFP | flagPatchRANS, 1, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := writePatchHeader(nil, tc.flags, tc.schema, tc.base)
			h, n, err := readPatchHeader(buf)
			if err != nil {
				t.Fatalf("readPatchHeader: %v", err)
			}
			if h.flags != tc.flags || h.schemaFP != tc.schema {
				t.Fatalf("flags/schema mismatch: got %#x/%#x", h.flags, h.schemaFP)
			}
			if tc.hasBase && h.baseFP != tc.base {
				t.Fatalf("baseFP: got %#x want %#x", h.baseFP, tc.base)
			}
			if n != len(buf) {
				t.Fatalf("consumed %d want %d", n, len(buf))
			}
		})
	}
}

func TestSchemaFingerprintStableAndDistinct(t *testing.T) {
	type A struct {
		X int
		Y string
	}
	type B struct {
		X int
		Z string // different field name
	}
	tdA1, _ := descOf(reflect.TypeFor[A]())
	tdA2, _ := descOf(reflect.TypeFor[A]())
	tdB, _ := descOf(reflect.TypeFor[B]())
	if schemaFingerprint(tdA1) != schemaFingerprint(tdA2) {
		t.Fatal("fingerprint not stable across calls")
	}
	if schemaFingerprint(tdA1) == schemaFingerprint(tdB) {
		t.Fatal("distinct shapes collided")
	}
}

func TestReadPatchHeaderRejectsBadMagic(t *testing.T) {
	bad := []byte{'X', 'D', 'P', patchVersion1, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if _, _, err := readPatchHeader(bad); err != ErrInvalidPatch {
		t.Fatalf("got %v want ErrInvalidPatch", err)
	}
	if _, _, err := readPatchHeader([]byte{'Q'}); err != ErrInvalidPatch {
		t.Fatalf("short: got %v want ErrInvalidPatch", err)
	}
	truncated := writePatchHeader(nil, flagPatchBaseFP, 1, 2)[:15] // cut mid-baseFP
	if _, _, err := readPatchHeader(truncated); err != ErrInvalidPatch {
		t.Fatalf("truncated baseFP: got %v want ErrInvalidPatch", err)
	}
}
