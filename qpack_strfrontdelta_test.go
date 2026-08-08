package qdf

import (
	"strconv"
	"testing"
)

func TestFrontDeltaCommonPrefix(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "abc", 0},
		{"abc", "abc", 3},
		{"abcdef", "abcxyz", 3},
		{"abc", "abcdef", 3},
		{"xyz", "abc", 0},
		// Past the 8-byte word step, and straddling it.
		{"abcdefgh", "abcdefgh", 8},
		{"abcdefghi", "abcdefghX", 8},
		{"abcdefghij", "abcdefghij", 10},
		{"aaaaaaaaaaaaaaaaX", "aaaaaaaaaaaaaaaaY", 16},
	} {
		if got := frontDeltaCommonPrefix(tc.a, tc.b); got != tc.want {
			t.Errorf("prefix(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFrontDeltaCommonSuffix(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		skip int
		want int
	}{
		{"", "", 0, 0},
		{"abc", "abc", 0, 3},
		{"xyzabc", "qqabc", 0, 3},
		// skip is the prefix already consumed: the suffix may not reach back
		// into it from either side.
		{"abcdef", "abcxef", 3, 2},
		{"aaaa", "aaaa", 2, 2},
		{"abc", "xbc", 3, 0},
		// Past the word step. These differ only at index 0, so everything
		// after it is shared — the point is that the word-wise scan agrees
		// with the byte-wise one across the 8-byte boundary.
		{"XabcdefghZ", "YabcdefghZ", 0, 9},
		{"XabcdefghijklZ", "YabcdefghijklZ", 0, 13},
	} {
		if got := frontDeltaCommonSuffix(tc.a, tc.b, tc.skip); got != tc.want {
			t.Errorf("suffix(%q,%q,skip=%d) = %d, want %d", tc.a, tc.b, tc.skip, got, tc.want)
		}
	}
}

func TestFrontDeltaProjectDeclines(t *testing.T) {
	// Too few rows to be worth a block header.
	short := []string{"aaaa", "aaab", "aaac"}
	if _, ok := frontDeltaProject(short); ok {
		t.Error("accepted a column below frontDeltaMinElems")
	}

	// No shared prefixes or suffixes: random-looking distinct values.
	rnd := make([]string, 200)
	for i := range rnd {
		rnd[i] = strconv.Itoa(i*2654435761) + "-" + strconv.Itoa(i*40503)
	}
	if _, ok := frontDeltaProject(rnd); ok {
		t.Error("accepted a column with nothing to share")
	}
}

func TestFrontDeltaProjectPicksMode(t *testing.T) {
	// Shared prefix only: the suffix varint would be pure overhead.
	pfx := make([]string, 200)
	for i := range pfx {
		pfx[i] = "/api/v2/resource/" + strconv.Itoa(i)
	}
	mode, ok := frontDeltaProject(pfx)
	if !ok {
		t.Fatal("declined a prefix-sharing column")
	}
	if mode != frontDeltaFrontOnly {
		t.Errorf("mode = %v, want frontDeltaFrontOnly", mode)
	}

	// Shared prefix AND suffix: the extra varint pays for itself.
	both := make([]string, 200)
	for i := range both {
		both[i] = "GET /api/v2/item/" + strconv.Itoa(i*7919) + " HTTP/1.1"
	}
	mode, ok = frontDeltaProject(both)
	if !ok {
		t.Fatal("declined a column sharing both ends")
	}
	if mode != frontDeltaFrontBack {
		t.Errorf("mode = %v, want frontDeltaFrontBack", mode)
	}
}
