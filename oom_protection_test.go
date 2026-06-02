package qdf

import (
	"encoding/binary"
	"testing"
)

// OOM-protection tests: a hostile payload that claims a multi-billion-
// element array, map, or string must return an error before the
// decoder reaches a `make` call. Decoder.CheckLength is the gate;
// these tests pin the contract.

func mkHeader() []byte { return []byte{'Q', 'D', 'F', 0x01, 0x00} }

func TestOOM_HugeArr32Header(t *testing.T) {
	// tagArr32 + 4-byte LE length 0xFFFFFFFF (~4 G elements). No body.
	buf := append(mkHeader(), tagArr32)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF)
	var out []int
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge arr32 length")
	}
	if out != nil && len(out) > 1<<28 {
		t.Fatalf("decoder allocated huge slice: %d", len(out))
	}
}

func TestOOM_HugeMap32Header(t *testing.T) {
	buf := append(mkHeader(), tagMap32)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF)
	var out map[string]int
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge map32 length")
	}
}

func TestOOM_HugeStr32Header(t *testing.T) {
	buf := append(mkHeader(), tagStr32)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF)
	var out string
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge str32 length")
	}
	if len(out) > 1<<20 {
		t.Fatalf("decoder produced huge string: %d", len(out))
	}
}

func TestOOM_HugeBin32Header(t *testing.T) {
	buf := append(mkHeader(), tagBin32)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF)
	var out []byte
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge bin32 length")
	}
}

func TestOOM_HugeInternStr(t *testing.T) {
	// tagInternStr + 10-byte varuint encoding ~MaxUint64 length, no body.
	buf := append(mkHeader(), tagInternStr)
	for range 9 {
		buf = append(buf, 0xFF)
	}
	buf = append(buf, 0x7F) // terminator
	var out string
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge intern_str length")
	}
}

func TestOOM_HugePackBool(t *testing.T) {
	// tagPackBool + 10-byte varuint of MaxUint64 - 1, no body.
	buf := append(mkHeader(), tagPackBool)
	for range 9 {
		buf = append(buf, 0xFF)
	}
	buf = append(buf, 0x7F)
	var out []bool
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge pack_bool length")
	}
}

func TestOOM_HugePackRaw(t *testing.T) {
	// tagPackRaw + kind=qpackKindUint64 + huge varuint count, no body.
	buf := append(mkHeader(), tagPackRaw, qpackKindUint64)
	for range 9 {
		buf = append(buf, 0xFF)
	}
	buf = append(buf, 0x7F)
	var out []uint64
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge pack_raw count")
	}
}

func TestOOM_HugePackFor(t *testing.T) {
	// tagPackFor + kind + bits=8 + min varuint (1 byte) + huge n varuint.
	buf := append(mkHeader(), tagPackFor, qpackKindUint64, 0x08, 0x00)
	for range 9 {
		buf = append(buf, 0xFF)
	}
	buf = append(buf, 0x7F)
	var out []uint64
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge pack_for count")
	}
}

func TestOOM_HugePackDeltaFor(t *testing.T) {
	buf := append(mkHeader(), tagPackDeltaFor, qpackKindUint64, 0x08, 0x00, 0x00)
	for range 9 {
		buf = append(buf, 0xFF)
	}
	buf = append(buf, 0x7F)
	var out []uint64
	if err := Unmarshal(buf, &out); err == nil {
		t.Fatalf("expected error on huge delta_for count")
	}
}

func TestOOM_NegativeArrayHeaderRejected(t *testing.T) {
	// Encoder cannot produce one, but ReadArrayHeader sees uint32 cast
	// of a "negative" value (top bit set) — must reject before alloc.
	buf := append(mkHeader(), tagArr32)
	buf = binary.LittleEndian.AppendUint32(buf, 0x80000001)
	d := NewDecoderOnBuf(buf)
	n, err := d.ReadArrayHeader()
	if err == nil && n < 0 {
		t.Fatalf("negative array len leaked: %d", n)
	}
	// either an error OR a sane non-negative result we cannot consume
	// is acceptable; the failure mode is "panic" or "OOM-attempting make".
}

// TestOOM_RLEColumnarBound pins the fix for the one integer codec that could
// bypass the columnar length bound. RLE can legitimately claim far more
// elements than remaining bytes (a long run is 2 bytes), so its header read
// must gate the count through colLenOK; inside a columnar column (colMaxLen
// set) a tiny body must not be able to claim a multi-GB element count.
func TestOOM_RLEColumnarBound(t *testing.T) {
	// kind byte + varuint n = 1<<20 (well under the 1<<30 standalone ceiling,
	// so only colLenOK can reject it).
	buf := []byte{qpackKindInt64}
	buf = appendUvarint(buf, 1<<20)
	d := &Decoder{buf: buf, colMaxLen: 8} // columnar column of 8 rows
	if _, err := d.readPackedRLEHeader(qpackKindInt64); err == nil {
		t.Fatal("RLE header claiming 1<<20 elems accepted under colMaxLen=8")
	}
	// Standalone (colMaxLen == 0) still accepts a plausible count.
	d2 := &Decoder{buf: buf}
	if _, err := d2.readPackedRLEHeader(qpackKindInt64); err != nil {
		t.Fatalf("standalone RLE header wrongly rejected: %v", err)
	}
}

// CheckLength contract: callers must invoke it before allocating. Verify
// it rejects clearly-impossible claims.
func TestOOM_CheckLengthRejectsImpossible(t *testing.T) {
	d := NewDecoderOnBuf(make([]byte, 100))
	if err := d.CheckLength(1<<30, 8); err == nil {
		t.Fatal("CheckLength accepted 8 GiB ask on 100-byte buffer")
	}
	if err := d.CheckLength(-1, 1); err == nil {
		t.Fatal("CheckLength accepted negative")
	}
	if err := d.CheckLength(10, 1); err != nil {
		t.Fatalf("CheckLength rejected 10 bytes from 100: %v", err)
	}
}
