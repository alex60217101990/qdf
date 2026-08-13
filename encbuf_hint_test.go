package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

type hintRow struct {
	A string
	B int64
	C float64
	D []string
	E map[string]string
	F []float64
	G bool
}

func mkHintRows(n int) []hintRow {
	out := make([]hintRow, n)
	svc := []string{"api", "worker", "cron", "db"}
	for i := range out {
		out[i] = hintRow{
			A: "/v1/resource/" + svc[i%4],
			B: int64(i * 7),
			C: float64(i) * 1.25,
			D: []string{svc[i%4], svc[(i+1)%4]},
			E: map[string]string{"svc": svc[i%4], "lvl": "info"},
			F: []float64{float64(i), float64(i) * 0.5},
			G: i%3 == 0,
		}
	}
	return out
}

// wantHintWireDigest pins the encoding of the corpus below. The output-buffer
// size hint decides how the buffer is allocated and must never touch a byte of
// what is written into it, so this digest is the same with the hint and with it
// removed — verified both ways when it was introduced.
//
// A change here means the wire format moved, which breaks every already-encoded
// payload. Update it only alongside a note saying why the break is intended.
const wantHintWireDigest = "b63ff32fee7a4611d47927ed0aa51054ba114acde8ca8e60fd29977e1f0ae844"

func TestEncBufHintWireIdentity(t *testing.T) {
	h := sha256.New()
	// Sizes straddle marshalDetachThreshold so both the clone path and the
	// detach path — the one the hint touches — are covered.
	for _, n := range []int{16, 1024, 8192, 40000} {
		v := mkHintRows(n)
		// OptCanonical throughout: the payload carries a map[string]string, and
		// without it Go's randomised map iteration changes the bytes every run,
		// which would leave this digest unable to fail for a real reason.
		for _, base := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
			o := base | OptCanonical
			blob, err := Marshal(v, o)
			if err != nil {
				t.Fatalf("n=%d opts=%v: %v", n, o, err)
			}
			h.Write(blob)
			ab, err := AppendMarshal(nil, v, o) // shares the detach sites
			if err != nil {
				t.Fatalf("append n=%d opts=%v: %v", n, o, err)
			}
			h.Write(ab)
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != wantHintWireDigest {
		t.Errorf("wire digest changed:\n got %s\nwant %s\nsee the note on wantHintWireDigest", got, wantHintWireDigest)
	}
}

// TestEncBufHintDecays pins the half of noteDetached that is easy to lose: the
// hint must follow the workload DOWN as well as up. Without the decay, one large
// message sets a high-water mark that every later small message allocates at,
// handing the caller a small slice backed by a large array for as long as that
// encoder lives.
//
// Driven directly rather than through Marshal: the encoder pool hands out
// whichever instance it likes, so a payload-level test mostly measures encoders
// that never saw the large message and passes whether the decay works or not.
// TestEncBufHintTracksTheWorkload pins what noteDetached and resetForReuse
// promise each other. Driven directly rather than through Marshal — the encoder
// pool hands out whichever instance it likes, so a payload-level test mostly
// measures encoders that never saw the message it set up and passes either way.
func TestEncBufHintTracksTheWorkload(t *testing.T) {
	var e Encoder

	// The hint is the capacity the message needed, not the bytes it delivered:
	// the codec picker writes candidate encodings and rewinds the losers, so the
	// peak sits above the output and sizing to the output alone would make every
	// message grow once.
	const big = 1 << 20
	e.noteDetached(make([]byte, big/2, big))
	if e.bufHint != big {
		t.Fatalf("hint did not adopt what the message needed: got %d want %d", e.bufHint, big)
	}

	// And it follows the workload down, or one large message would leave every
	// later small one allocating at its size — handing the caller a small slice
	// backed by a large array for as long as that encoder lives.
	const small = 4096
	e.noteDetached(make([]byte, 512, small))
	if e.bufHint != small {
		t.Errorf("hint did not follow the workload down: got %d want %d", e.bufHint, small)
	}
}

// TestEncBufHintSkipsLargePayloads pins the bound on where pre-allocating pays.
// slices.Grow serves a large request with one right-sized allocation, so above
// maxHintedBuf the doubling chain is already short and a pre-allocation is
// either overshoot or an undershoot that doubles on top — measured 18.3 MB/op
// unhinted against 21.9 hinted on a 17.9 MB batch.
func TestEncBufHintSkipsLargePayloads(t *testing.T) {
	var e Encoder

	e.bufHint = maxHintedBuf
	e.buf = nil
	e.resetForReuse()
	if cap(e.buf) != maxHintedBuf {
		t.Errorf("a hint at the bound should still pre-allocate: cap %d want %d", cap(e.buf), maxHintedBuf)
	}

	e.bufHint = maxHintedBuf + 1
	e.buf = nil
	e.resetForReuse()
	if cap(e.buf) != 0 {
		t.Errorf("a hint past the bound should not pre-allocate: cap %d want 0", cap(e.buf))
	}
}

func TestWideScratchSurvivesLargeColumns(t *testing.T) {
	type wideCol struct {
		Data []int32 `qdf:"data"`
	}
	mk := func(n int) wideCol {
		d := make([]int32, n)
		for i := range d {
			d[i] = int32(i * 3 % 100000)
		}
		return wideCol{Data: d}
	}

	// The property is what putEnc does with the widening scratch on release:
	// keep it when it is within the retention ceiling, drop it when it is a
	// one-off spike. Asserted on the scratch itself.
	//
	// It used to be inferred from a MemStats delta over 100 pooled Marshal
	// calls, and that measurement could not see what it claimed to. In a full
	// suite it failed about one run in three, blaming the scratch, while a
	// memory profile of a failing run put widenI64 at 2.6% of allocation and
	// resetForReuse at all the rest: the buffer HINT (encoder.go:524)
	// pre-allocates cap(previous output) whenever a pooled encoder comes back
	// with an empty buffer, so a hint left by a neighbouring test dominated the
	// figure. Which encoder the pool returned depended on GC timing, so the
	// verdict inverted with GOGC — off failed 5/5, GOGC=1 passed 3/3, the
	// opposite of what a scratch-retention test should do.
	release := func(n int) (retained int, ceiling int) {
		enc := NewEncoderWith(OptBalanced)
		if err := enc.EncodeValue(mk(n)); err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if cap(enc.wideI64) == 0 {
			t.Fatalf("n=%d: encoding an []int32 column did not widen anything — the test "+
				"is no longer exercising the path it names", n)
		}
		putEnc(enc, &encPool)
		return cap(enc.wideI64), maxRetainedWideScratch
	}

	// Under the ceiling: kept, so the next message of the same shape reuses it.
	const largeN = 100000 // over the former 1<<14 ceiling, under the current 1<<17
	if got, ceiling := release(largeN); got == 0 {
		t.Errorf("a %d-element column released its widening scratch (ceiling %d) — every "+
			"message of this shape now rebuilds it", largeN, ceiling)
	}

	// Over the ceiling: dropped, so one spike does not pin megabytes on a
	// pooled encoder forever. Without this arm the test would pass with the
	// retention check deleted and the scratch simply kept unconditionally.
	if got, ceiling := release(maxRetainedWideScratch + 1); got != 0 {
		t.Errorf("a spike-sized column kept a %d-element widening scratch (ceiling %d) — "+
			"it is pinned on the pooled encoder", got, ceiling)
	}
}
