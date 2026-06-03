package fsst

import (
	"bytes"
	"testing"
)

func FuzzFSSTRoundTrip(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("GET /a/b/c HTTP/1.1"))
	f.Add(bytes.Repeat([]byte{0xFF}, 20))
	f.Fuzz(func(t *testing.T, in []byte) {
		// Train on the single input split into pseudo-rows so the table is
		// data-dependent, then round-trip the whole thing.
		tbl := BuildSymbolTable([][]byte{in})
		out := tbl.Decompress(tbl.Compress(in, nil), nil)
		if !bytes.Equal(out, in) {
			t.Fatalf("round-trip mismatch len(in)=%d len(out)=%d", len(in), len(out))
		}
	})
}

func FuzzFSSTDecodeNoPanic(f *testing.F) {
	f.Add([]byte{0x02, 0x02, 'a', 'b', 0x01, 'c', 0x00, escapeCode})
	f.Fuzz(func(t *testing.T, blob []byte) {
		// Treat blob as table||codes; must never panic or OOM.
		tbl, n, err := UnmarshalSymbolTable(blob)
		if err != nil {
			return
		}
		if n > len(blob) {
			t.Fatal("consumed past end")
		}
		out := tbl.Decompress(blob[n:], nil)
		if len(out) > 8*len(blob[n:])+len(blob) {
			t.Fatalf("decompressed expansion implausible: %d from %d", len(out), len(blob)-n)
		}
	})
}
