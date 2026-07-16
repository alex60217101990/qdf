package qdf

import "testing"

// BenchmarkDecodeLargeInt measures ReadInt throughput for uint-tagged values.
// Large positive int64 values are encoded with tagUint32/tagUint64 by the
// encoder and decoded through ReadInt→decodeInt. Task 20 (DECODEINT-DOUBLE-
// DISPATCH) inlines the tagUintN cases directly in decodeInt, eliminating the
// decodeInt→decodeUint call and the redundant fixint branch check that
// decodeUint performs (already proven false at the call site).
func BenchmarkDecodeLargeInt(b *testing.B) {
	const n = 512

	// Build a raw buffer of n tagUint64-encoded values.
	// All values > 2^32 so the encoder (or our manual build here) must use tagUint64.
	buf := make([]byte, 0, n*9)
	for i := range n {
		v := uint64(5_000_000_000) + uint64(i) // > 2^32, forces tagUint64
		buf = append(buf, tagUint64)
		// LittleEndian — matches the readU64 helper used by the decoder.
		buf = append(buf,
			byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
			byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56),
		)
	}

	// Warm-up decode to verify correctness.
	d := &Decoder{buf: buf, headerRead: true}
	for range n {
		if _, err := d.ReadInt(); err != nil {
			b.Fatalf("warm-up ReadInt: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.i = 0 // reset cursor; buf and headerRead stay set
		for range n {
			if _, err := d.ReadInt(); err != nil {
				b.Fatal(err)
			}
		}
	}
}
