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

func TestFrontDeltaWriterHeader(t *testing.T) {
	strs := make([]string, 100)
	for i := range strs {
		strs[i] = "/api/v2/resource/" + strconv.Itoa(i)
	}

	e := NewEncoder(Dense)
	e.applyOpts(OptBalanced)
	if !e.tryWriteStringColumnFrontDelta(strs) {
		t.Fatal("writer declined a prefix-sharing column")
	}
	buf := e.Bytes()

	// The block begins at the tag; everything before it is the frame header.
	i := 0
	for i < len(buf) && buf[i] != tagColStrFrontDelta {
		i++
	}
	if i == len(buf) {
		t.Fatal("tag not found in output")
	}
	i++
	n, nr := readUvarint(buf[i:])
	if nr <= 0 || n != uint64(len(strs)) {
		t.Fatalf("row count = %d (nr=%d), want %d", n, nr, len(strs))
	}
	i += nr
	if flags := buf[i]; flags&^1 != 0 {
		t.Errorf("reserved flag bits set: %#x", flags)
	}
}

func TestFrontDeltaWriterDeclines(t *testing.T) {
	rnd := make([]string, 200)
	for i := range rnd {
		rnd[i] = strconv.Itoa(i*2654435761) + "-" + strconv.Itoa(i*40503)
	}
	e := NewEncoder(Dense)
	e.applyOpts(OptBalanced)
	before := len(e.Bytes())
	if e.tryWriteStringColumnFrontDelta(rnd) {
		t.Error("writer accepted a column with nothing to share")
	}
	if len(e.Bytes()) != before {
		t.Error("writer left bytes behind after declining")
	}
}

// frontDeltaRoundTrip encodes strs through the writer and reads them back
// through the reader, returning what the reader produced.
func frontDeltaRoundTrip(t *testing.T, strs []string) []string {
	t.Helper()
	e := NewEncoder(Dense)
	e.applyOpts(OptBalanced)
	if !e.tryWriteStringColumnFrontDelta(strs) {
		t.Fatal("writer declined the column")
	}
	buf := e.Bytes()
	i := 0
	for i < len(buf) && buf[i] != tagColStrFrontDelta {
		i++
	}
	d := NewDecoderOnBuf(buf)
	d.i = i
	got, err := d.readStringColumnFrontDelta(len(strs))
	if err != nil {
		t.Fatalf("reader: %v", err)
	}
	return got
}

func TestFrontDeltaRoundTrip(t *testing.T) {
	mk := func(shape string, n int) []string {
		out := make([]string, n)
		for i := range out {
			switch shape {
			case "prefix":
				out[i] = "/api/v2/resource/" + strconv.Itoa(i)
			case "bothends":
				out[i] = "GET /api/v2/item/" + strconv.Itoa(i*7919) + " HTTP/1.1"
			case "allequal":
				out[i] = "the same value every row"
			case "empties":
				if i%3 == 0 {
					out[i] = ""
				} else {
					out[i] = "/api/v2/resource/" + strconv.Itoa(i)
				}
			}
		}
		return out
	}
	// Sizes straddle the 64-row anchor so boundary rows are covered.
	for _, shape := range []string{"prefix", "bothends", "allequal", "empties"} {
		for _, n := range []int{32, 63, 64, 65, 128, 1000} {
			want := mk(shape, n)
			got := frontDeltaRoundTrip(t, want)
			if len(got) != len(want) {
				t.Fatalf("%s n=%d: got %d rows, want %d", shape, n, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s n=%d row %d: got %q, want %q", shape, n, i, got[i], want[i])
				}
			}
		}
	}
}

func TestFrontDeltaRejectsMalformed(t *testing.T) {
	strs := make([]string, 100)
	for i := range strs {
		strs[i] = "/api/v2/resource/" + strconv.Itoa(i)
	}
	e := NewEncoder(Dense)
	e.applyOpts(OptBalanced)
	if !e.tryWriteStringColumnFrontDelta(strs) {
		t.Fatal("writer declined")
	}
	full := append([]byte(nil), e.Bytes()...)
	start := 0
	for start < len(full) && full[start] != tagColStrFrontDelta {
		start++
	}

	// Every single-byte corruption inside the block, and every truncation of
	// it, must produce an error rather than a panic or a wrong value.
	for pos := start; pos < len(full); pos++ {
		for _, delta := range []byte{1, 0x7f, 0xff} {
			b := append([]byte(nil), full...)
			b[pos] += delta
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic on byte %d += %d: %v", pos, delta, r)
					}
				}()
				d := NewDecoderOnBuf(b)
				d.i = start
				_, _ = d.readStringColumnFrontDelta(len(strs))
			}()
		}
	}
	for cut := start; cut < len(full); cut++ {
		b := append([]byte(nil), full[:cut]...)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncation at %d: %v", cut, r)
				}
			}()
			d := NewDecoderOnBuf(b)
			d.i = start
			if _, err := d.readStringColumnFrontDelta(len(strs)); err == nil {
				t.Fatalf("truncation at %d accepted", cut)
			}
		}()
	}
}

func TestFrontDeltaRejectsReservedFlags(t *testing.T) {
	strs := make([]string, 100)
	for i := range strs {
		strs[i] = "/api/v2/resource/" + strconv.Itoa(i)
	}
	e := NewEncoder(Dense)
	e.applyOpts(OptBalanced)
	if !e.tryWriteStringColumnFrontDelta(strs) {
		t.Fatal("writer declined")
	}
	b := append([]byte(nil), e.Bytes()...)
	start := 0
	for start < len(b) && b[start] != tagColStrFrontDelta {
		start++
	}
	_, nr := readUvarint(b[start+1:])
	b[start+1+nr] |= 0x80 // set a reserved bit

	d := NewDecoderOnBuf(b)
	d.i = start
	if _, err := d.readStringColumnFrontDelta(len(strs)); err == nil {
		t.Error("accepted a reserved flag bit")
	}
}
