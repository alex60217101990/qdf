package qdf

import "testing"

// A tagPackDeltaFor value whose element count makes (n64-1)*bitsPer overflow
// uint64 must be rejected by Skip, not silently mis-advanced. With bitsPer=56
// and n64=329406144173384852, (n64-1)*56 wraps to 40 (bodyBits) -> bodyBytes=5,
// so the pre-fix multiply produced a 5-byte body size that passed the bounds
// check against a 5-byte tail and desynced the skip cursor (returning nil). The
// division-based bound rejects it (ErrShortBuffer) before the multiply.
func TestSkipDeltaForCountOverflow(t *testing.T) {
	var buf []byte
	buf = append(buf, tagPackDeltaFor, qpackKindInt64, 56) // tag, kind, bitsPer=56
	buf = appendUvarint(buf, 0)                            // firstVal
	buf = appendUvarint(buf, 0)                            // minDelta
	buf = appendUvarint(buf, 329406144173384852)           // n64: (n64-1)*56 overflows to 40
	buf = append(buf, 0, 0, 0, 0, 0)                       // 5-byte tail the wrapped bodyBytes would consume

	d := &Decoder{buf: buf, headerRead: true}
	err := d.Skip()
	if err == nil {
		t.Fatal("Skip accepted an overflowing tagPackDeltaFor count (cursor desynced); want a bounds error")
	}
}
