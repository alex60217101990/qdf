package qdf

import (
	"strings"
	"testing"
)

func TestStrDeltaRoundTrip(t *testing.T) {
	cases := []struct{ base, s string }{
		{"", "hello"},
		{"hello", "hello"},
		{"hello", "help"},
		{"/api/v1/users/1000", "/api/v1/users/1001"},
		{"abc", ""},
		{"", ""},
		{strings.Repeat("x", 300), strings.Repeat("x", 300) + "y"},
	}
	for _, c := range cases {
		buf := appendStrDelta(c.base, c.s)
		if len(buf) != strDeltaCost(c.base, c.s) {
			t.Fatalf("base=%q s=%q: wrote %d bytes, cost said %d",
				c.base, c.s, len(buf), strDeltaCost(c.base, c.s))
		}
		got, n, err := readStrDelta(buf, c.base)
		if err != nil {
			t.Fatalf("base=%q s=%q: %v", c.base, c.s, err)
		}
		if got != c.s {
			t.Fatalf("base=%q: got %q want %q", c.base, got, c.s)
		}
		if n != len(buf) {
			t.Fatalf("base=%q s=%q: consumed %d of %d", c.base, c.s, n, len(buf))
		}
	}
}

// A prefix longer than the base would read out of bounds. The reader must
// reject it rather than slice past the end.
func TestStrDeltaRejectsOverlongPrefix(t *testing.T) {
	buf := appendStrDelta("abcdef", "abcdefgh")
	bad := append([]byte(nil), buf...)
	bad[1] = 200 // buf[0] is the tag, buf[1] the prefix varint
	if _, _, err := readStrDelta(bad, "abc"); err == nil {
		t.Fatal("reader accepted a prefix longer than the base")
	}
}

func TestStrDeltaRejectsTruncation(t *testing.T) {
	full := appendStrDelta("/api/v1/users/1000", "/api/v1/users/1001")
	for i := range full {
		if _, _, err := readStrDelta(full[:i], "/api/v1/users/1000"); err == nil {
			t.Fatalf("reader accepted a %d-byte truncation of a %d-byte value", i, len(full))
		}
	}
}

// Every single-byte corruption must be rejected or decode to something the
// reader can account for — never read out of bounds and never panic.
func TestStrDeltaSurvivesEverySingleByteCorruption(t *testing.T) {
	base := "/api/v1/tenants/9f3a/users/100000"
	full := appendStrDelta(base, "/api/v1/tenants/9f3a/users/100001")
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
				_, _, _ = readStrDelta(m, base)
			}()
		}
	}
}
