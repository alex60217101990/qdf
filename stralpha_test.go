package qdf

import (
	"strings"
	"testing"
)

func TestStrAlphaWellKnownRoundTrip(t *testing.T) {
	cases := []string{
		"4bf92f3577b34da6a3ce929d0e0e4736", // lowercase hex
		"00F067AA0BA902B7",                 // uppercase hex
		"1234567890",                       // decimal
		"abc-XYZ_09",                       // base64url
	}
	for _, s := range cases {
		id := strAlphaWellKnown(s)
		if id == 0 {
			t.Fatalf("%q matched no well-known alphabet", s)
		}
		buf := appendStrAlphaWellKnown(nil, id, s)
		got, n, err := readStrAlpha(buf, nil, nil)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if got != s {
			t.Fatalf("got %q want %q", got, s)
		}
		if n != len(buf) {
			t.Fatalf("%q: consumed %d of %d", s, n, len(buf))
		}
		// Not every match is a win: a short value over a wide alphabet costs
		// more than it saves, and the encoder's cost gate is what declines it.
		// What must hold is that the length the form reports is the length it
		// writes, so the gate is comparing against the truth.
		set := strAlphaSets[id]
		if want := strAlphaCost(len(s), set.bits, id, 0); want != len(buf) {
			t.Fatalf("%q: strAlphaCost says %d bytes, append wrote %d", s, want, len(buf))
		}
	}
}

func TestStrAlphaWellKnownRejectsForeignCharacters(t *testing.T) {
	if id := strAlphaWellKnown("4bf92f35 77b34da6"); id != 0 {
		t.Fatalf("a space matched well-known alphabet %d", id)
	}
	if id := strAlphaWellKnown(""); id != 0 {
		t.Fatal("the empty string matched a well-known alphabet")
	}
}

// The word-wise membership test must agree with a plain byte loop on every
// input, including the ones that fail — the failing case is the common one and
// the whole reason the scan is written this way.
func TestStrAlphaWellKnownMatchesTheByteLoop(t *testing.T) {
	ref := func(s string) uint8 {
		if len(s) == 0 {
			return 0
		}
		for _, id := range strAlphaOrder {
			set := strAlphaSets[id]
			ok := true
			for i := range len(s) {
				if !set.member[s[i]] {
					ok = false
					break
				}
			}
			if ok {
				return id
			}
		}
		return 0
	}
	seed := uint64(1)
	next := func() byte {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return byte(seed)
	}
	for n := range 40 {
		for range 200 {
			b := make([]byte, n)
			for i := range b {
				// Bias towards the alphabets so both outcomes are common.
				switch next() % 4 {
				case 0:
					b[i] = "0123456789abcdef"[int(next())%16]
				case 1:
					b[i] = "0123456789"[int(next())%10]
				case 2:
					b[i] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"[int(next())%64]
				default:
					b[i] = next()
				}
			}
			s := string(b)
			if got, want := strAlphaWellKnown(s), ref(s); got != want {
				t.Fatalf("%q: word-wise says %d, byte loop says %d", s, got, want)
			}
		}
	}
}

func TestStrAlphaDeclaredRoundTrip(t *testing.T) {
	vals := []string{"/api/v1/users", "/api/v1/orders", "/api/v1/carts"}
	var seen [256]bool
	var code [256]uint8
	var alphabet []byte
	for _, v := range vals {
		for i := range len(v) {
			if c := v[i]; !seen[c] {
				seen[c] = true
				code[c] = uint8(len(alphabet))
				alphabet = append(alphabet, c)
			}
		}
	}
	var tbl strAlphaTable

	buf := appendStrAlphaDeclared(nil, alphabet, &code, vals[0])
	got, n, err := readStrAlpha(buf, &tbl, nil) // the declaration carries its own table
	if err != nil {
		t.Fatal(err)
	}
	if got != vals[0] || n != len(buf) {
		t.Fatalf("declared: got %q (%d of %d bytes)", got, n, len(buf))
	}
	if len(tbl.alphabet) != len(alphabet) {
		t.Fatalf("the declaration did not record its table: %d symbols", len(tbl.alphabet))
	}

	ref := appendStrAlphaRef(nil, &code, bitsForDistinct(len(alphabet)), vals[1])
	got, n, err = readStrAlpha(ref, &tbl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != vals[1] || n != len(ref) {
		t.Fatalf("ref: got %q (%d of %d bytes)", got, n, len(ref))
	}
}

// A reference form without a declared table is a malformed stream, not a panic.
func TestStrAlphaRefWithoutTableIsRejected(t *testing.T) {
	var code [256]uint8
	code['a'], code['b'] = 0, 1
	ref := appendStrAlphaRef(nil, &code, 1, "abab")
	if _, _, err := readStrAlpha(ref, nil, nil); err == nil {
		t.Fatal("a reference form decoded without a declared table")
	}
	var empty strAlphaTable
	if _, _, err := readStrAlpha(ref, &empty, nil); err == nil {
		t.Fatal("a reference form decoded against an empty table")
	}
}

// Every truncation and every single-byte corruption must be rejected without
// panicking and without reading past the buffer.
func TestStrAlphaSurvivesTruncationAndCorruption(t *testing.T) {
	s := strings.Repeat("4bf92f35", 4)
	full := appendStrAlphaWellKnown(nil, strAlphaWellKnown(s), s)
	for i := range full {
		if _, _, err := readStrAlpha(full[:i], nil, nil); err == nil {
			t.Fatalf("accepted a %d-byte truncation of %d", i, len(full))
		}
	}
	for i := range full {
		for b := range 256 {
			if byte(b) == full[i] {
				continue
			}
			m := append([]byte(nil), full...)
			m[i] = byte(b)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic on byte %d = 0x%02x: %v", i, b, r)
					}
				}()
				var tbl strAlphaTable
				_, _, _ = readStrAlpha(m, &tbl, nil)
			}()
		}
	}
}

// A declaration whose table is longer than the buffer, or wider than the packer
// supports, must be rejected before anything is indexed by it.
func TestStrAlphaRejectsImpossibleTables(t *testing.T) {
	for _, a := range []byte{0, 1, qpackStrAlphaMaxAlphabet + 1, 200} {
		buf := []byte{tagStrAlpha, strAlphaSelDeclare, a}
		buf = append(buf, make([]byte, 8)...)
		var tbl strAlphaTable
		if _, _, err := readStrAlpha(buf, &tbl, nil); err == nil {
			t.Fatalf("accepted a declaration claiming %d symbols", a)
		}
	}
}

// The packed form must actually be smaller where it is meant to be: a long
// value over a narrow alphabet. This is the case the codec exists for.
func TestStrAlphaPacksNarrowAlphabetsSmaller(t *testing.T) {
	for _, s := range []string{
		"4bf92f3577b34da6a3ce929d0e0e4736", // 32 hex chars, 4 bits each
		"00f067aa0ba902b7",                 // 16 hex chars
		"170000000012345678",               // 18 decimal digits, 4 bits each
	} {
		id := strAlphaWellKnown(s)
		if id == 0 {
			t.Fatalf("%q matched no well-known alphabet", s)
		}
		buf := appendStrAlphaWellKnown(nil, id, s)
		raw := uvarintLen(uint64(len(s))) + len(s)
		if len(buf) >= raw {
			t.Fatalf("%q: packed to %d bytes against a %d-byte raw form", s, len(buf), raw)
		}
		got, _, err := readStrAlpha(buf, nil, nil)
		if err != nil || got != s {
			t.Fatalf("%q: round-trip gave %q, %v", s, got, err)
		}
	}
}
