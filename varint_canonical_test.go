package qdf

import "testing"

// TestReadUvarintCanonical pins that readUvarint rejects a non-canonical
// 10-byte varint (one whose 10th byte sets bits above 63) instead of silently
// truncating, matching encoding/binary.Uvarint. Hardening against hostile wire.
func TestReadUvarintCanonical(t *testing.T) {
	// 9 continuation bytes + a 10th byte of 0x02 would shift bit 1 to position
	// 64 — out of range. Must be rejected (n == -1), not truncated to 0.
	bad := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x02}
	if v, n := readUvarint(bad); n != -1 {
		t.Fatalf("non-canonical 10-byte varint accepted: v=%d n=%d (want n=-1)", v, n)
	}
	// Canonical MaxUint64 (9×0xFF + 0x01) must still decode exactly.
	maxv := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}
	if v, n := readUvarint(maxv); n != 10 || v != ^uint64(0) {
		t.Fatalf("canonical MaxUint64 mis-decoded: v=%#x n=%d", v, n)
	}
	// A bare overlong continuation with no terminator is still incomplete.
	if _, n := readUvarint([]byte{0x80}); n != 0 {
		t.Fatalf("truncated varint: want n=0, got %d", n)
	}
	// Small canonical values unaffected.
	for _, tc := range []struct {
		b []byte
		v uint64
	}{{[]byte{0x00}, 0}, {[]byte{0x7f}, 127}, {[]byte{0x80, 0x01}, 128}} {
		if v, n := readUvarint(tc.b); n != len(tc.b) || v != tc.v {
			t.Fatalf("readUvarint(%x): v=%d n=%d want v=%d n=%d", tc.b, v, n, tc.v, len(tc.b))
		}
	}
}
