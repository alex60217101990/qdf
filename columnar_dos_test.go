package qdf

import (
	"errors"
	"testing"
)

// TestColumnarDecodeMemoryAmplification is the audit-6 RED repro for the
// columnar memory-amplification DoS: a tagColStruct header claims a struct
// count up to maxColumnarElems (1<<24), and decodeColumnar allocates the output
// slice as count*elemSize BEFORE reading the (constant/RLE-compressible) column
// bodies. A tiny hostile input then forces a multi-hundred-MB / multi-GB
// allocation. The count ceiling bounds elements, never bytes.
func TestColumnarDecodeMemoryAmplification(t *testing.T) {
	type dosRow struct {
		A int64 `qdf:"a"`
		B int64 `qdf:"b"`
		C int64 `qdf:"c"`
		D int64 `qdf:"d"`
		E int64 `qdf:"e"`
	}
	// 64 all-scalar rows → the columnar probe fires (tagColStruct, 0xEF) under
	// OptBalanced. Keep the values small so 0xEF only appears as the frame tag.
	rows := make([]dosRow, 64)
	for i := range rows {
		rows[i] = dosRow{A: int64(i)}
	}
	buf, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if buf[5] != tagColStruct {
		t.Fatalf("expected tagColStruct (0x%02x) at buf[5], got 0x%02x — payload not columnar", tagColStruct, buf[5])
	}

	// Splice the 1-byte struct count (64 = 0x40) with a count whose
	// count*stride (stride 40 B) exceeds a sane byte ceiling: 8<<20 * 40 ≈ 335 MB.
	const evilN = 8 << 20
	evil := append([]byte{}, buf[:6]...) // header(5) + tagColStruct(1)
	evil = appendUvarint(evil, uint64(evilN))
	evil = append(evil, buf[7:]...) // rest after the original 1-byte count

	var out []dosRow
	derr := Unmarshal(evil, &out)
	// A tiny hostile input must be rejected, not amplified into a giant slice.
	if !errors.Is(derr, ErrInvalidLength) {
		t.Fatalf("hostile columnar count not rejected: err=%v, len(out)=%d (memory-amplification DoS)", derr, len(out))
	}
}
