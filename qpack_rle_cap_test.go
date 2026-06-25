package qdf

import (
	"encoding/binary"
	"testing"
)

// TestRLEStandaloneCapNotBypassedByLargeBuffer is a regression for the OOM where
// the standalone element-count cap was nested inside `n64 > remaining`: a large
// input buffer made that guard vacuous, letting a tiny RLE body claim up to
// `remaining` elements and allocate ~8x. The cap must fire unconditionally.
func TestRLEStandaloneCapNotBypassedByLargeBuffer(t *testing.T) {
	n := uint64(qpackMaxStandaloneCount) + 1 // just over the cap

	var hdr []byte
	hdr = append(hdr, tagPackRLE, qpackKindUint64)
	hdr = binary.AppendUvarint(hdr, n)

	// Pad so that remaining-after-header > n (defeats the old `n64 > remaining`
	// guard). Filler bytes stand in for run data; decode must reject at the
	// header before reading them — and before allocating make([]uint64, n).
	buf := make([]byte, len(hdr)+int(n)+16)
	copy(buf, hdr)

	d := &Decoder{buf: buf}
	d.i++ // consume the peeked tag (tagPackRLE)
	if _, err := d.readPackedRLEUint64Slice(); err != ErrInvalidLength {
		t.Fatalf("want ErrInvalidLength for over-cap RLE in a large buffer, got %v", err)
	}
}
