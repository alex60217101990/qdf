package qdf

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/alex60217101990/qdf/internal/internarena"
)

// internBatch drives n distinct large keys through the encoder intern table,
// growing the arena past the default soft cap, then resets once. Returns the
// arena bytes resident *after* the reset — i.e. what the adaptive policy chose
// to keep warm for the next batch.
func internBatch(e *encState, n int) int {
	for i := range n {
		// ~100-byte distinct keys: n=5000 => ~500 KiB, well past the 256 KiB
		// DefaultRetainBytes cap, so the spike chunks are what's at stake.
		key := "host-" + strconv.Itoa(i) + "-region-eu-west-1-az-1-service-telemetry-collector-shard"
		e.lookupOrAssign(key)
	}
	e.reset()
	return e.arena.ResidentBytes()
}

// TestStreamArenaRetainAcrossLargeBatches proves the adaptive arena retention:
// while a steady high-cardinality (streaming-shaped) workload keeps
// retainStreak pinned at 0, the encoder keeps its grown arena slabs resident
// across reset() so the next same-shaped batch reuses them instead of
// regrowing — the streaming-encode allocation lever. Once the burst subsides
// (retainReleaseStreak consecutive small batches), the arena falls back to the
// default soft cap and sheds the spike memory.
func TestStreamArenaRetainAcrossLargeBatches(t *testing.T) {
	const bigN = 5000 // > maxRetainedIDs (4096) => retainStreak stays 0.

	e := newEncState()

	// First large batch grows the arena; reset retains it (release == false).
	resident := internBatch(e, bigN)
	if resident <= internarena.DefaultRetainBytes {
		t.Fatalf("after first large batch: arena resident=%d, want > DefaultRetainBytes=%d (spike chunks should be retained, not dropped)",
			resident, internarena.DefaultRetainBytes)
	}
	peak := resident

	// Steady large batches must keep the arena resident — no per-batch
	// drop-and-regrow. Resident stays at the single-batch peak (cursor rolls
	// to 0 each reset and reuses the same slabs), never shedding.
	for b := range 5 {
		resident = internBatch(e, bigN)
		if resident < peak {
			t.Fatalf("steady large batch %d: arena resident=%d dropped below peak=%d; spike chunks were shed mid-stream (regrow alloc)",
				b, resident, peak)
		}
	}

	// Burst subsides: retainReleaseStreak consecutive SMALL batches must drive
	// release and shrink the arena back under the default soft cap.
	var small int
	for b := range retainReleaseStreak {
		small = internBatch(e, 4) // tiny load => streak advances toward release.
		_ = b
	}
	if small > internarena.DefaultRetainBytes {
		t.Fatalf("after %d small batches: arena resident=%d, want <= DefaultRetainBytes=%d (burst subsided, spike memory should be released)",
			retainReleaseStreak, small, internarena.DefaultRetainBytes)
	}
}

// TestStreamArenaRetainWireIdentical locks the invariant that adaptive arena
// retention is a memory-only optimization: the encoded bytes for a fixed batch
// must be byte-identical whether or not the prior batch left spike slabs warm.
// Encoding the same batch on a fresh stream and after a large warm-up batch (so
// the arena is retained) must produce the same wire output — intern ids come
// from internLoad (reset to 0 each batch), independent of slab layout.
func TestStreamArenaRetainWireIdentical(t *testing.T) {
	type row struct {
		Name string
		ID   string
		N    int64
	}
	batch := make([]row, 64)
	for i := range batch {
		k := strconv.Itoa(i)
		batch[i] = row{Name: "service-instance-" + k + ".prod.internal", ID: "uuid-" + k, N: int64(i)}
	}

	encodeBatch := func(warmup bool) []byte {
		var buf bytes.Buffer
		enc := NewStreamEncoder(&buf, Dense)
		if warmup {
			// Large prior batch grows + retains the arena slabs, then reset.
			enc.Reset(&buf)
			for i := range 6000 {
				_ = enc.Encode(row{Name: "warm-" + strconv.Itoa(i) + "-padding-to-grow-the-arena-past-cap", ID: strconv.Itoa(i), N: int64(i)})
			}
			buf.Reset()
		}
		enc.Reset(&buf)
		for i := range batch {
			if err := enc.Encode(batch[i]); err != nil {
				t.Fatal(err)
			}
		}
		return append([]byte(nil), buf.Bytes()...)
	}

	cold := encodeBatch(false)
	warm := encodeBatch(true)
	if !bytes.Equal(cold, warm) {
		t.Fatalf("wire output differs with arena retention: cold=%d bytes, warm=%d bytes — retention must not affect encoding", len(cold), len(warm))
	}
}
