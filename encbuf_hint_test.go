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
const wantHintWireDigest = "bb7d614b06b3fea25eef90638abeef843318c40573df68e5c441d37044433ecf"

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
func TestEncBufHintDecays(t *testing.T) {
	var e Encoder

	// A large message detaches and sets the mark.
	big := make([]byte, 4<<20)
	e.noteDetached(big)
	peak := e.bufHint
	if peak != len(big) {
		t.Fatalf("hint did not adopt the large message: got %d want %d", peak, len(big))
	}

	const smallLen = 512
	// Small messages that fit must walk it back down. Each iteration mirrors
	// what resetForReuse does: the next buffer is allocated at exactly the
	// current hint, so its capacity tracks the hint rather than staying pinned
	// at the peak.
	for range 64 {
		e.noteDetached(make([]byte, smallLen, e.bufHint))
	}
	if e.bufHint >= peak {
		t.Fatalf("hint did not decay: still %d after 64 small messages (peak %d)", e.bufHint, peak)
	}
	if e.bufHint < smallLen {
		t.Errorf("hint decayed below what the data used: %d < %d", e.bufHint, smallLen)
	}
	t.Logf("hint %d -> %d", peak, e.bufHint)

	// And it must rise again at once when a message outgrows it.
	e.noteDetached(make([]byte, 8<<20))
	if e.bufHint != 8<<20 {
		t.Errorf("hint did not rise to a larger message: got %d", e.bufHint)
	}

	// An outlier must never be adopted past the pool's own retention ceiling.
	e.noteDetached(make([]byte, maxPooledBuf*4))
	if e.bufHint > maxPooledBuf {
		t.Errorf("hint exceeded maxPooledBuf: %d", e.bufHint)
	}
}
