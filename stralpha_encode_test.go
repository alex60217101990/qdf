package qdf

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
)

type saRow struct {
	Seq   int64             `qdf:"seq"`
	Trace string            `qdf:"trace"`
	Tags  map[string]string `qdf:"tags"`
}

// writeN drives the field writer directly.
//
// Firing counts are asserted here rather than through Marshal because the
// container decision is upstream of this codec: a payload whose string column
// the columnar probe accepts never reaches the row-major writer at all, and an
// assertion through Marshal would be measuring the probe, not the packer.
// End-to-end behavior is covered by the round-trip tests below.
func writeN(t *testing.T, vals []string) (wk, decl, ref int64) {
	t.Helper()
	e := NewEncoderWith(OptBalanced | OptStringAlphabet)
	var fs strFieldState
	w0, d0, r0 := strAlphaEmittedWK.Load(), strAlphaEmittedDecl.Load(), strAlphaEmittedRef.Load()
	for _, v := range vals {
		e.writeStringField(v, &fs)
	}
	return strAlphaEmittedWK.Load() - w0, strAlphaEmittedDecl.Load() - d0, strAlphaEmittedRef.Load() - r0
}

func TestStrAlphaWellKnownFiresWithoutDeclaringATable(t *testing.T) {
	vals := make([]string, 512)
	for i := range vals {
		vals[i] = fmt.Sprintf("%032x", uint64(i)*11400714819323198485)
	}
	wk, decl, ref := writeN(t, vals)
	// The first value has no base and takes the plain path; every later one is
	// packed against the stateless hex alphabet.
	if wk < int64(len(vals))-1 {
		t.Fatalf("well-known fired %d times on %d hex values", wk, len(vals))
	}
	if decl != 0 || ref != 0 {
		t.Fatalf("a well-known alphabet declared a table: decl=%d ref=%d", decl, ref)
	}
}

func TestStrAlphaDeclaresOnceThenReferences(t *testing.T) {
	// Lowercase letters only, and deliberately without a shared prefix: a
	// prefix would go to the delta, which is the correct codec for it. This
	// shape matches the 36-symbol lowercase-and-digits set at six bits, so the
	// win only exists if the field learns the 26 symbols it really uses.
	vals := make([]string, 512)
	seed := uint64(1)
	for i := range vals {
		b := make([]byte, 24)
		for j := range b {
			seed = seed*6364136223846793005 + 1442695040888963407
			b[j] = byte('a' + (seed>>33)%26)
		}
		vals[i] = string(b)
	}
	_, decl, ref := writeN(t, vals)
	if decl != 1 {
		t.Fatalf("declared the table %d times, want exactly 1", decl)
	}
	if ref == 0 {
		t.Fatal("no value referenced the declared table")
	}
}

// A field whose character set is too wide must give up, and stay given up.
func TestStrAlphaGivesUpOnAWideField(t *testing.T) {
	vals := make([]string, 512)
	for i := range vals {
		b := make([]byte, 40)
		for j := range b {
			b[j] = byte(32 + (i*31+j*17)%95) // 95 printable symbols
		}
		vals[i] = string(b)
	}
	wk, decl, ref := writeN(t, vals)
	if wk != 0 || decl != 0 || ref != 0 {
		t.Fatalf("alpha fired on a 95-symbol field: wk=%d decl=%d ref=%d", wk, decl, ref)
	}
}

// The gate that stops a wide field from paying the scan forever must be doing
// something: break it and the field must start being offered again. Without
// this the previous test passes whether or not the gate ever engages.
func TestStrAlphaGiveUpGuardIsNotVacuous(t *testing.T) {
	e := NewEncoderWith(OptBalanced | OptStringAlphabet)
	var fs strFieldState
	for i := range 64 {
		b := make([]byte, 40)
		for j := range b {
			b[j] = byte(32 + (i*31+j*17)%95)
		}
		e.writeStringField(string(b), &fs)
	}
	if !fs.alphaMuted && !fs.alphaOff {
		t.Fatal("a 95-symbol field was still being offered to the codec")
	}
	fs.alphaOff, fs.alphaMuted, fs.alphaProbe = false, false, 0
	w0 := strAlphaEmittedWK.Load()
	for i := range 8 {
		e.writeStringField(fmt.Sprintf("%032x", uint64(i)*2654435761), &fs)
	}
	if strAlphaEmittedWK.Load() == w0 {
		t.Fatal("clearing the gate changed nothing — it is not what gates the codec")
	}
}

// Round-trip through the public API on the shapes the codec targets and the
// shapes it must decline, under every option combination that changes the
// string path.
func TestStrAlphaRoundTripsEveryShape(t *testing.T) {
	shapes := map[string]func(i int) string{
		"hex32":   func(i int) string { return fmt.Sprintf("%032x", uint64(i)*2654435761) },
		"decimal": func(i int) string { return fmt.Sprintf("%018d", i*7919) },
		"path":    func(i int) string { return fmt.Sprintf("/api/v1/tenants/x/users/%d", i) },
		"wide": func(i int) string {
			b := make([]byte, 40)
			for j := range b {
				b[j] = byte(32 + (i*31+j*17)%95)
			}
			return string(b)
		},
		"short":    func(i int) string { return strconv.Itoa(i % 7) },
		"constant": func(i int) string { return "/healthz/ready/probe" },
		"empty":    func(i int) string { return "" },
		"mixed": func(i int) string {
			if i%3 == 0 {
				return fmt.Sprintf("%032x", uint64(i))
			}
			return fmt.Sprintf("mixed value with spaces %d", i)
		},
	}
	// Both with and without the bit: the codec must be invisible when it is off
	// and correct when it is on, and every value must survive either way.
	opts := []Options{
		OptBalanced | OptStringAlphabet,
		OptBalanced | OptStringAlphabet | OptCanonical,
		OptBalanced | OptStringAlphabet&^OptMTF,
		OptBalanced | OptStringAlphabet&^OptPairPred,
		OptCompression | OptStringAlphabet,
		OptBalanced,
		OptCompression,
		OptSpeed,
	}
	for name, f := range shapes {
		rows := make([]saRow, 700)
		for i := range rows {
			rows[i] = saRow{Seq: int64(i), Trace: f(i), Tags: map[string]string{"k": "v"}}
		}
		for _, o := range opts {
			b, err := Marshal(rows, o)
			if err != nil {
				t.Fatalf("%s opts=%v: %v", name, o, err)
			}
			var got []saRow
			if err := Unmarshal(b, &got); err != nil {
				t.Fatalf("%s opts=%v: %v", name, o, err)
			}
			for i := range rows {
				if got[i].Trace != rows[i].Trace || got[i].Seq != rows[i].Seq {
					t.Fatalf("%s opts=%v row %d: got %+v want %+v", name, o, i, got[i], rows[i])
				}
			}
			// Dynamic decode walks the same bytes with no target type.
			var a any
			if err := Unmarshal(b, &a); err != nil {
				t.Fatalf("%s opts=%v dynamic: %v", name, o, err)
			}
		}
	}
}

// The bit must be inert wherever an entropy coder runs, and this test is what
// keeps it that way.
//
// Packing to five bits destroys the byte-level skew rANS and FSST feed on, so
// the two together lose to either alone: measured across ten identifier shapes,
// a dashed UUID field cost 17.5% MORE wire under OptCompression with the bit
// than without it and a MAC-address field 18.2% more, while the best case was
// -0.1%. Never better, sometimes much worse — so the encoder ignores the bit
// rather than honoring it, and the wire must come out byte-identical.
func TestStrAlphaIsInertUnderEntropyCoding(t *testing.T) {
	type row struct {
		Seq  int64 `qdf:"seq"`
		Rows []struct {
			A string `qdf:"a"`
			B string `qdf:"b"`
		} `qdf:"rows"`
	}
	rows := make([]row, 600)
	seed := uint64(0x9E3779B97F4A7C15)
	hex := func() string {
		b := make([]byte, 32)
		for j := range b {
			seed = seed*6364136223846793005 + 1442695040888963407
			b[j] = "0123456789abcdef"[(seed>>33)%16]
		}
		return string(b)
	}
	for i := range rows {
		rows[i].Seq = int64(i)
		rows[i].Rows = make([]struct {
			A string `qdf:"a"`
			B string `qdf:"b"`
		}, 2)
		for k := range rows[i].Rows {
			rows[i].Rows[k].A, rows[i].Rows[k].B = hex(), hex()
		}
	}

	for _, o := range []Options{OptCompression, OptBalanced | OptRANS, OptBalanced | OptFSST} {
		off, err := Marshal(rows, o)
		if err != nil {
			t.Fatal(err)
		}
		on, err := Marshal(rows, o|OptStringAlphabet)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(off, on) {
			t.Fatalf("opts=%v: the bit changed the wire under entropy coding (%d vs %d bytes)",
				o, len(off), len(on))
		}
	}

	// And the guard must not be vacuous: without an entropy coder the same
	// payload has to get materially smaller, or the test above would pass on a
	// codec that never runs at all.
	plain, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	packed, err := Marshal(rows, OptBalanced|OptStringAlphabet)
	if err != nil {
		t.Fatal(err)
	}
	if len(packed) >= len(plain)*9/10 {
		t.Fatalf("hex ids packed to %d bytes against %d — the codec is not running", len(packed), len(plain))
	}
}
