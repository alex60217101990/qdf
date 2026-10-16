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
// modeled them as interned and so fired the bulk form even though the bulk
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
	// Long, high-cardinality tokens with nothing for another codec to take:
	// printable bytes from a ~90-symbol alphabet (alpha bails past 64), no
	// shared prefix or suffix (front-delta declines), no repeats (dict bails).
	// The previous tokens all ended in "-zzqxij", a shared suffix front-delta
	// rightly took once the picker started choosing the smallest codec.
	x := uint64(0x9E3779B97F4A7C15)
	for i := range rows {
		var b [24]byte
		for j := range b {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
			b[j] = byte(33 + x%90)
		}
		rows[i] = rawTokRow{Seq: int64(i), TS: int64(i * 1000), Tok: string(b[:])}
	}
	// The picker's verdict, read at the block's own first byte. Searching the
	// whole Marshal output for the tag byte, as this used to, passed with the
	// raw codec disabled: 0xF8 turns up in the integer columns.
	toks := make([]string, n)
	for i := range rows {
		toks[i] = rows[i].Tok
	}
	e := NewEncoderWith(OptBalanced)
	e.writeHeader()
	at := len(e.buf)
	e.writeStringColumn(toks)
	if got := e.buf[at]; got != tagColStrRaw {
		t.Fatalf("tok column is block %#x, want tagColStrRaw: lost the one-slab decode on a long high-cardinality column", got)
	}

	buf, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
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
