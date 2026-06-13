package qdf

import (
	"bytes"
	"strconv"
	"testing"
)

type rawTokRow struct {
	Seq int64  `qdf:"seq"`
	TS  int64  `qdf:"ts"` // second scalar column tips the columnar probe
	Tok string `qdf:"tok"`
}

// TestStrRaw_NeverLargerOnShortStrings pins the never-larger fix for the
// tagColStrRaw codec on a high-cardinality SHORT-string column. WriteString
// emits sub-minIntern strings inline (no intern, no dedup); the old estimate
// modelled them as interned and so fired the bulk form even though the bulk
// header made it larger than the per-value path. After the fix the estimate
// matches reality: for a column where bulk would bloat, it must decline.
func TestStrRaw_NeverLargerOnShortStrings(t *testing.T) {
	const n = 200
	rows := make([]rawTokRow, n)
	for i := range rows {
		// distinct 2-char tokens (len 2 < minIntern 4) — base-36 of i, padded.
		s := strconv.FormatInt(int64(i), 36)
		for len(s) < 2 {
			s = "0" + s
		}
		rows[i] = rawTokRow{Seq: int64(i), TS: int64(i * 1000), Tok: s}
	}
	buf, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// The bulk form must NOT have been chosen here (it would be larger by the
	// bulk header than the inline per-value path).
	if bytes.IndexByte(buf, tagColStrRaw) >= 0 {
		t.Fatalf("tagColStrRaw fired on a short-string column where it bloats the wire (%d bytes)", len(buf))
	}
	var out []rawTokRow
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out) != n {
		t.Fatalf("len %d != %d", len(out), n)
	}
	for i := range out {
		if out[i] != rows[i] {
			t.Fatalf("row %d: %+v != %+v", i, out[i], rows[i])
		}
	}
}

// TestStrRaw_FiresOnLongHighCardStrings is the no-regression counterpart: the
// bulk form must STILL be chosen for a high-cardinality long-string column,
// where it is wire-safe and saves n decode allocations.
func TestStrRaw_FiresOnLongHighCardStrings(t *testing.T) {
	const n = 200
	rows := make([]rawTokRow, n)
	for i := range rows {
		// Long, high-cardinality, NON-substring-sharing tokens (reversed digits
		// + per-row salt) so dict/FSST can't compress and the raw bulk form wins.
		s := strconv.FormatInt(int64(i*2654435761&0x7fffffff), 36) + "-" + strconv.FormatInt(int64(i)*0x9e3779b1&0x7fffffff, 36) + "-zzqxij"
		rows[i] = rawTokRow{Seq: int64(i), TS: int64(i * 1000), Tok: s}
	}
	buf, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if bytes.IndexByte(buf, tagColStrRaw) < 0 {
		t.Fatalf("tagColStrRaw did NOT fire on a long high-cardinality column (regression: lost the decode-alloc win)")
	}
	var out []rawTokRow
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for i := range out {
		if out[i] != rows[i] {
			t.Fatalf("row %d mismatch", i)
		}
	}
}
