package qdf

import (
	"io"
	"strconv"
	"testing"
)

// adRow is a small high-cardinality record: every batch carries distinct,
// long-ish string keys/values (host names, SIDs, paths) — the AD/telemetry
// shape where first-occurrence interning dominates encode allocation.
type adRow struct {
	Host    string
	SID     string
	Path    string
	Service string
	Value   int64
}

func makeADBatch(n, batchSeed int) []adRow {
	rows := make([]adRow, n)
	for i := range n {
		k := strconv.Itoa(batchSeed*1_000_000 + i)
		rows[i] = adRow{
			Host:    "WIN-DC-" + k + ".corp.internal.example.com",
			SID:     "S-1-5-21-3623811015-3361044348-30300820-" + k,
			Path:    "C:\\Windows\\System32\\config\\systemprofile\\AppData\\Local\\" + k,
			Service: "telemetry-collector-shard-" + k,
			Value:   int64(i),
		}
	}
	return rows
}

// BenchmarkStreamEncodeADRetain streams many same-shaped high-cardinality
// batches through one reused StreamEncoder, resetting per batch (the
// qdf-bench / per-request pattern). Reports encode allocation per value. With
// adaptive arena retention the spike slabs survive the per-batch Reset, so a
// steady stream stops paying a full arena regrow every batch.
func BenchmarkStreamEncodeADRetain(b *testing.B) {
	const rowsPerBatch = 4000 // ~400 KiB interned/batch: past the 256 KiB cap.

	// Distinct batches so keys never repeat across batches (worst case for
	// interning — every batch is a fresh first-occurrence wave).
	batches := make([][]adRow, 16)
	for i := range batches {
		batches[i] = makeADBatch(rowsPerBatch, i+1)
	}

	enc := NewStreamEncoder(io.Discard, Dense)

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		batch := batches[i%len(batches)]
		enc.Reset(io.Discard)
		for j := range batch {
			if err := enc.Encode(batch[j]); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(rowsPerBatch), "rows/batch")
}
