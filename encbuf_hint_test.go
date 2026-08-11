package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime"
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
const wantHintWireDigest = "d3afb72c4e63c7c09a4136f28406686c0639eb3ebb158968886b9181afca2511"

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

	measure := func(v wideCol) uint64 {
		for range 50 { // let the pooled scratch settle
			if _, err := Marshal(v, OptBalanced); err != nil {
				t.Fatal(err)
			}
		}
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		const runs = 100
		for range runs {
			if _, err := Marshal(v, OptBalanced); err != nil {
				t.Fatal(err)
			}
		}
		runtime.ReadMemStats(&b)
		return (b.TotalAlloc - a.TotalAlloc) / runs
	}

	const largeN = 100000 // over the former 1<<14 ceiling, under the current 1<<17
	small := measure(mk(8192))
	large := measure(mk(largeN))

	// The scratch is []uint64, so dropping and rebuilding it costs exactly 8
	// bytes per element per message. Bound the whole message by that figure:
	// well above what the column legitimately allocates, well below what it
	// costs to rebuild the scratch on top of that.
	//
	// The bound is absolute rather than a multiple of the small column because
	// -race inflates every allocation, and by different factors on different
	// runners. Measured per element: 2.1 B fixed without -race, up to 6.0 B
	// fixed under it, against 10.2 B with the ceiling back at 1<<14 — so 8 B
	// separates the two regimes in both modes.
	if limit := uint64(8 * largeN); large > limit {
		t.Errorf("a %d-element column allocates %d B/op (limit %d, small column %d) — the widening scratch is being dropped every message",
			largeN, large, limit, small)
	}
	t.Logf("8192 elems: %d B/op   %d elems: %d B/op (%.1f B/elem)", small, largeN, large, float64(large)/largeN)
}
