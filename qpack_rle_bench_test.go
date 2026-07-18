package qdf

import "testing"

// BenchmarkRLEDecodeRunHeavyUint64 measures readPackedRLEUint64Slice throughput
// for a run-heavy workload: 4 runs of 256 elements each (1024 elements total).
// This exercises the copy-doubling fill path, which routes through
// runtime.memmove and uses NEON Q-register stores on ARM64.
func BenchmarkRLEDecodeRunHeavyUint64(b *testing.B) {
	const (
		nRuns  = 4
		runLen = 256
		nTotal = nRuns * runLen
	)

	// Build input: 4 distinct values, each repeated 256 times.
	in := make([]uint64, nTotal)
	for r := range nRuns {
		v := uint64(r+1) * 100
		for i := range runLen {
			in[r*runLen+i] = v
		}
	}

	enc := NewEncoder(Fast)
	enc.writePackedRLEUint64Slice(in)
	encoded := enc.buf

	// Locate the start of the RLE body (after stream header + tag byte).
	// peekTag reads the header and leaves d.i pointing at the tag; we skip
	// that tag too so readPackedRLEUint64Slice sees kind + n + runs.
	d := NewDecoderOnBuf(encoded)
	tag, err := d.peekTag()
	if err != nil || tag != tagPackRLE {
		b.Fatalf("unexpected tag %02x err=%v", tag, err)
	}
	d.i++ // consume tag
	bodyStart := d.i

	// Warm-up: verify correctness once.
	out, err := d.readPackedRLEUint64Slice()
	if err != nil {
		b.Fatalf("decode: %v", err)
	}
	if len(out) != nTotal {
		b.Fatalf("expected %d elements, got %d", nTotal, len(out))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.i = bodyStart
		if _, err := d.readPackedRLEUint64Slice(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRLEDecodeRunHeavyInt64 is the signed counterpart.
func BenchmarkRLEDecodeRunHeavyInt64(b *testing.B) {
	const (
		nRuns  = 4
		runLen = 256
		nTotal = nRuns * runLen
	)

	in := make([]int64, nTotal)
	for r := range nRuns {
		v := int64(r+1) * -100
		for i := range runLen {
			in[r*runLen+i] = v
		}
	}

	enc := NewEncoder(Fast)
	enc.writePackedRLEInt64Slice(in)
	encoded := enc.buf

	d := NewDecoderOnBuf(encoded)
	tag, err := d.peekTag()
	if err != nil || tag != tagPackRLE {
		b.Fatalf("unexpected tag %02x err=%v", tag, err)
	}
	d.i++
	bodyStart := d.i

	out, err := d.readPackedRLEInt64Slice()
	if err != nil {
		b.Fatalf("decode: %v", err)
	}
	if len(out) != nTotal {
		b.Fatalf("expected %d elements, got %d", nTotal, len(out))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.i = bodyStart
		if _, err := d.readPackedRLEInt64Slice(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRLEIntoRunHeavy benchmarks the reuse-buffer (Into) variant, which
// avoids per-call allocation and isolates the fill loop latency. This is the
// hot path used by decodeColumnInto for columnar RLE columns.
// 4 runs × 256 elements = 1024 elements (runLen=256 >> 32, exercises NEON path).
func BenchmarkRLEIntoRunHeavy(b *testing.B) {
	const (
		nRuns  = 4
		runLen = 256
		nTotal = nRuns * runLen
	)

	in := make([]uint64, nTotal)
	for r := range nRuns {
		v := uint64(r+1) * 100
		for i := range runLen {
			in[r*runLen+i] = v
		}
	}

	enc := NewEncoder(Fast)
	enc.writePackedRLEUint64Slice(in)
	encoded := enc.buf

	d := NewDecoderOnBuf(encoded)
	tag, err := d.peekTag()
	if err != nil || tag != tagPackRLE {
		b.Fatalf("unexpected tag %02x err=%v", tag, err)
	}
	d.i++
	bodyStart := d.i

	var dst []uint64
	if err := d.readPackedRLEUint64SliceInto(&dst); err != nil {
		b.Fatalf("decode: %v", err)
	}
	if len(dst) != nTotal {
		b.Fatalf("expected %d elements, got %d", nTotal, len(dst))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.i = bodyStart
		if err := d.readPackedRLEUint64SliceInto(&dst); err != nil {
			b.Fatal(err)
		}
	}
}
