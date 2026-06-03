package fsst

import (
	"bytes"
	"testing"
)

// newTestTable builds a SymbolTable from explicit symbols, in code order.
func newTestTable(syms ...string) *SymbolTable {
	b := make([][]byte, len(syms))
	for i, s := range syms {
		b[i] = []byte(s)
	}
	return newSymbolTable(b)
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	tbl := newTestTable("http://", "www.", ".com", "/index")
	in := []byte("http://www.example.com/index")
	comp := tbl.Compress(in, nil)
	if len(comp) >= len(in) {
		t.Logf("no shrink (ok for this assertion): %d -> %d", len(in), len(comp))
	}
	out := tbl.Decompress(comp, nil)
	if !bytes.Equal(out, in) {
		t.Fatalf("round-trip mismatch:\n in=%q\nout=%q", in, out)
	}
}

func TestRoundTripByteRange(t *testing.T) {
	tbl := newTestTable("ab", "abc") // arbitrary symbols
	for n := 0; n < 256; n++ {
		in := make([]byte, n)
		for i := range in {
			in[i] = byte((i * 7) ^ n)
		}
		out := tbl.Decompress(tbl.Compress(in, nil), nil)
		if !bytes.Equal(out, in) {
			t.Fatalf("n=%d mismatch", n)
		}
	}
}

func TestRoundTripEscapeByte(t *testing.T) {
	tbl := newTestTable("x")
	in := bytes.Repeat([]byte{0xFF}, 40) // 0xFF is the escape code value
	out := tbl.Decompress(tbl.Compress(in, nil), nil)
	if !bytes.Equal(out, in) {
		t.Fatal("escape-byte round-trip failed")
	}
}

// --- Phase 2 tests (appended) ---

func TestBuildSymbolTableShrinksAndRoundTrips(t *testing.T) {
	samples := make([][]byte, 0, 200)
	for i := 0; i < 200; i++ {
		samples = append(samples, []byte("GET /api/v1/users/"+string(rune('a'+i%26))+" HTTP/1.1"))
	}
	tbl := BuildSymbolTable(samples)
	var total, comp int
	for _, s := range samples {
		total += len(s)
		comp += len(tbl.Compress(s, nil))
		out := tbl.Decompress(tbl.Compress(s, nil), nil)
		if string(out) != string(s) {
			t.Fatalf("round-trip failed for %q", s)
		}
	}
	if comp >= total {
		t.Fatalf("expected shrink on repetitive corpus: total=%d comp=%d", total, comp)
	}
}

func TestBuildSymbolTableDeterministic(t *testing.T) {
	samples := [][]byte{[]byte("alpha-beta"), []byte("beta-gamma"), []byte("alpha-gamma")}
	a := BuildSymbolTable(samples)
	b := BuildSymbolTable(samples)
	if len(a.symbols) != len(b.symbols) {
		t.Fatal("symbol count differs")
	}
	for i := range a.symbols {
		if a.symbols[i] != b.symbols[i] {
			t.Fatalf("symbol %d differs", i)
		}
	}
}

// --- Phase 3 tests (appended) ---

func TestSymbolTableMarshalRoundTrip(t *testing.T) {
	tbl := newTestTable("http://", "x", ".com", "12345678") // incl. 8-byte symbol
	b := tbl.MarshalTo(nil)
	got, n, err := UnmarshalSymbolTable(b)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(b) {
		t.Fatalf("consumed %d want %d", n, len(b))
	}
	if len(got.symbols) != len(tbl.symbols) {
		t.Fatalf("symbol count %d want %d", len(got.symbols), len(tbl.symbols))
	}
	for i := range tbl.symbols {
		if got.symbols[i] != tbl.symbols[i] {
			t.Fatalf("symbol %d mismatch: got %v want %v", i, got.symbols[i], tbl.symbols[i])
		}
	}
}

func TestUnmarshalSymbolTableRejectsBad(t *testing.T) {
	cases := [][]byte{
		{0x01},             // count=1 but no symbol bytes
		{0x01, 0x00},       // symLen=0 illegal
		{0x01, 0x09, 1},    // symLen=9 > maxSymLen
		{0x01, 0x03, 1, 2}, // symLen=3 but only 2 bytes follow
	}
	for i, b := range cases {
		if _, _, err := UnmarshalSymbolTable(b); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}
