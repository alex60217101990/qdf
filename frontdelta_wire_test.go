package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// wantUntouchedWireDigest pins the encoding of the column shapes the
// front-delta codec must NOT claim: a low-cardinality column the dictionary
// takes, and a hex column the alphabet packer takes. If the new codec ever
// intercepts one of them, this digest moves.
const wantUntouchedWireDigest = "f499a903c3fe0e2406bec5608ab9cb8bf97ee63d679858a955a2e8eb3db92de9"

type untouchedRow struct {
	Level string `qdf:"level"`
	Trace string `qdf:"trace"`
	N     int64  `qdf:"n"`
}

func mkUntouchedRows(n int) []untouchedRow {
	const hexdigits = "0123456789abcdef"
	lvl := []string{"info", "warn", "error", "debug"}
	out := make([]untouchedRow, n)
	for i := range out {
		tr := make([]byte, 32)
		for j := range tr {
			tr[j] = hexdigits[(i*31+j*17)%16]
		}
		out[i] = untouchedRow{Level: lvl[i%4], Trace: string(tr), N: int64(i)}
	}
	return out
}

func TestUntouchedColumnsWireIdentity(t *testing.T) {
	h := sha256.New()
	for _, n := range []int{64, 1000, 5000} {
		rows := mkUntouchedRows(n)
		for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
			blob, err := Marshal(rows, o)
			if err != nil {
				t.Fatalf("n=%d %v: %v", n, o, err)
			}
			h.Write(blob)
		}
	}
	got := hex.EncodeToString(h.Sum(nil))
	t.Logf("untouched-column digest: %s", got)
	if got != wantUntouchedWireDigest {
		t.Errorf("digest changed:\n got %s\nwant %s\nthe front-delta codec claimed a column it should not", got, wantUntouchedWireDigest)
	}
}
