package qdf

import "testing"

// TestOOM_ConstantCountCapTightened pins that the standalone constant-codec
// element-count ceiling is tight enough to prevent a multi-GB allocation from a
// ~14-byte header. A constant codec (bitsPer == 0) carries an EMPTY body, so the
// per-element byte bound cannot apply and only the absolute ceiling defends the
// make(). The ceiling must be on the order of maxColumnarElems (1<<24, 128 MiB
// for int64), NOT 1<<30 (8 GiB). `mid` sits above the cap but far below the old
// 8 GiB ceiling, so it asserts the bound was actually tightened. Reader-level
// assertions keep the test safe: a correct reader rejects BEFORE the make.
func TestOOM_ConstantCountCapTightened(t *testing.T) {
	const mid = uint64(1) << 25 // 33.5M elements ⇒ 268 MiB int64 make if accepted

	if qpackMaxStandaloneCount > maxColumnarElems {
		t.Fatalf("qpackMaxStandaloneCount=%d must be <= maxColumnarElems=%d to cap the constant-codec make",
			qpackMaxStandaloneCount, maxColumnarElems)
	}

	t.Run("for", func(t *testing.T) {
		buf := []byte{qpackKindUint64, 0x00, 0x00} // kind, bits=0, min=0
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, _, _, _, _, err := d.readPackedForHeader(qpackKindUint64); err == nil {
			t.Fatal("FOR bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("delta", func(t *testing.T) {
		buf := []byte{qpackKindUint64, 0x00, 0x00, 0x00} // kind, bits=0, first=0, minDelta=0
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, _, _, _, _, _, err := d.readPackedDeltaForHeader(qpackKindUint64); err == nil {
			t.Fatal("Delta-FOR bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("pfor", func(t *testing.T) {
		buf := []byte{qpackKindInt64}
		buf = appendUvarint(buf, mid) // n
		buf = append(buf, 0x00, 0x00) // b=0, min=0
		d := &Decoder{buf: buf}
		if _, err := d.readPackedPForInt64Slice(); err == nil {
			t.Fatal("PFor b=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("dict", func(t *testing.T) {
		// 1 distinct value ⇒ bitsForDistinct(1)==0 ⇒ empty index body.
		buf := []byte{qpackKindInt64}
		buf = appendUvarint(buf, 1) // 1 distinct value
		buf = appendUvarint(buf, 0) // table[0]
		buf = appendUvarint(buf, mid)
		d := &Decoder{buf: buf}
		if _, err := d.readPackedDictInt64Slice(); err == nil {
			t.Fatal("Dict bits=0 accepted a 268 MiB standalone count")
		}
	})
	t.Run("rle", func(t *testing.T) {
		buf := []byte{qpackKindUint64}
		buf = appendUvarint(buf, mid) // claimed element count, body is a tiny run
		d := &Decoder{buf: buf}
		if _, err := d.readPackedRLEHeader(qpackKindUint64); err == nil {
			t.Fatal("RLE accepted a 268 MiB standalone count")
		}
	})
}
