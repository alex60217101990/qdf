package vecquant

import (
	"encoding/binary"
	"testing"
)

func TestCoordStreamRoundTrip(t *testing.T) {
	cases := [][]int32{
		{},
		{0},
		{0, 1, -1, 2, -2, 127, -128, 1000, -1000, 1 << 20, -(1 << 20)},
	}
	for _, q := range cases {
		enc := encodeCoords(q)
		got, err := decodeCoords(enc, len(q))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got) != len(q) {
			t.Fatalf("len %d != %d", len(got), len(q))
		}
		for i := range q {
			if got[i] != q[i] {
				t.Fatalf("i=%d got %d want %d", i, got[i], q[i])
			}
		}
	}
}

func TestCoordStreamNeverLarger(t *testing.T) {
	// Highly compressible (all zero) must use the rANS branch and be small.
	q := make([]int32, 4096)
	enc := encodeCoords(q)
	if len(enc) >= 4096 {
		t.Fatalf("zero stream not compressed: %d bytes", len(enc))
	}
}

func TestDecodeCoordsRejectsOversizedRawLen(t *testing.T) {
	// Craft a hostile stream with an oversized rawLen that would trigger OOM.
	// Mode 1 (rANS) + huge rawLen + minimal body should be rejected.
	count := 4

	// Construct a buffer: mode=1, varint(1<<40), empty body.
	var buf []byte
	buf = append(buf, 1) // mode=1 (rANS)

	// Encode a large rawLen as uvarint (e.g. 1<<40).
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], 1<<40)
	buf = append(buf, tmp[:n]...)

	// Add a tiny/empty body.
	buf = append(buf, []byte{}...)

	// decodeCoords should reject this without panicking or OOMing.
	got, err := decodeCoords(buf, count)
	if err == nil {
		t.Fatal("expected error for oversized rawLen, got nil")
	}
	if got != nil {
		t.Fatal("expected nil result on error")
	}
	if got == nil && err != nil {
		// Success: error was returned as expected.
		t.Logf("correctly rejected oversized rawLen: %v", err)
	}
}

func TestDecodeCoordsRejectsOversizedRawLenMode0(t *testing.T) {
	// Same test but for mode 0 (raw varint path).
	count := 4

	var buf []byte
	buf = append(buf, 0) // mode=0 (raw)

	// Encode a large rawLen as uvarint.
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], 1<<40)
	buf = append(buf, tmp[:n]...)

	// Add body that claims to be 1<<40 bytes but is actually small.
	buf = append(buf, []byte{0xAA, 0xBB}...)

	// decodeCoords should reject this.
	got, err := decodeCoords(buf, count)
	if err == nil {
		t.Fatal("expected error for oversized rawLen (mode 0), got nil")
	}
	if got != nil {
		t.Fatal("expected nil result on error")
	}
	if got == nil && err != nil {
		t.Logf("correctly rejected oversized rawLen (mode 0): %v", err)
	}
}
