package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// FuzzPrimitivesRoundTrip flips every byte of a known-good encoding to make
// sure the decoder never panics, no matter how mangled the input. Returning
// an error is fine; crashing is not.
func FuzzDecoder_NeverPanics(f *testing.F) {
	// Seed corpus: a handful of well-formed encodings.
	good := []any{
		"hello world",
		[]int{1, 2, 3, 4, 5},
		map[string]int{"a": 1, "b": 2},
		Outer{Name: "alice", Age: 30, Active: true, Score: 3.14, Tags: []string{"x"}, Meta: map[string]string{"k": "v"}, Inner: Inner{X: 1, Y: 1.5}, Buf: []byte{1, 2, 3}, Counts: [3]int32{1, 2, 3}},
	}
	for _, v := range good {
		b, err := Marshal(v, OptSpeed)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
		bd, err := Marshal(v, OptBalanced)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(bd)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// Try decoding into a few different shapes. None should panic.
		var s string
		_ = Unmarshal(data, &s)
		var i int
		_ = Unmarshal(data, &i)
		var m map[string]any
		_ = Unmarshal(data, &m)
		var slc []any
		_ = Unmarshal(data, &slc)
		var o Outer
		_ = Unmarshal(data, &o)
	})
}

// FuzzRoundTrip generates random payloads via standard types and verifies
// round-trip equality through both Fast and Dense modes.
func FuzzRoundTrip_StringSlice(f *testing.F) {
	f.Add(uint8(3), "hello", "world", "foo")
	f.Fuzz(func(t *testing.T, n uint8, a, b, c string) {
		in := make([]string, 0, int(n)+3)
		for range n {
			in = append(in, a, b, c)
		}
		// Fast
		buf, err := Marshal(in, OptSpeed)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("fast mismatch: %v vs %v", in, out)
		}
		// Dense
		buf2, err := Marshal(in, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		var out2 []string
		if err := Unmarshal(buf2, &out2); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(in, out2) {
			t.Fatalf("dense mismatch: %v vs %v", in, out2)
		}
		// Dense should be ≤ Fast in size for repetitive input
		// (but not strictly when n=0/1).
		if n >= 2 && len(buf2) > len(buf) {
			t.Logf("warning: dense not smaller than fast at n=%d: fast=%d dense=%d", n, len(buf), len(buf2))
		}
	})
}

func TestDecoder_TruncatedInput(t *testing.T) {
	// Encode a Flat struct, then try decoding every prefix < full length.
	// Every short read must return an error, never panic.
	in := Outer{Name: "alice", Age: 30, Tags: []string{"x", "y", "z"}}
	full, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	for i := range full {
		var out Outer
		// Recover from any panic for diagnostics.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic decoding prefix len=%d: %v", i, r)
				}
			}()
			_ = Unmarshal(full[:i], &out)
		}()
	}
}

func TestDecoder_BadMagic(t *testing.T) {
	bad := []byte{'X', 'Y', 'Z', 0x01, 0x00, tagNil}
	var out any
	if err := Unmarshal(bad, &out); err == nil {
		t.Fatalf("expected error on bad magic, got nil")
	}
}

func TestDecoder_BadVersion(t *testing.T) {
	bad := []byte{'Q', 'D', 'F', 0xFF, 0x00, tagNil}
	var out any
	if err := Unmarshal(bad, &out); err == nil {
		t.Fatalf("expected error on bad version, got nil")
	}
}

func TestEdgeCases_BoundaryInts(t *testing.T) {
	cases := []int64{
		0, 1, -1, 127, 128, -128, -129,
		32767, 32768, -32768, -32769,
		2147483647, 2147483648, -2147483648, -2147483649,
		9223372036854775807, -9223372036854775808,
	}
	for _, v := range cases {
		b, err := Marshal(v, OptSpeed)
		if err != nil {
			t.Fatalf("encode %d: %v", v, err)
		}
		var out int64
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("decode %d: %v (bytes=%x)", v, err, b)
		}
		if out != v {
			t.Fatalf("int round-trip mismatch: in=%d out=%d", v, out)
		}
	}
}

func TestEdgeCases_HugeString(t *testing.T) {
	cases := []int{0, 1, 31, 32, 255, 256, 65535, 65536, 1 << 20}
	for _, n := range cases {
		s := bytes.Repeat([]byte{'a'}, n)
		b, err := Marshal(string(s), OptSpeed)
		if err != nil {
			t.Fatalf("encode len=%d: %v", n, err)
		}
		var out string
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("decode len=%d: %v", n, err)
		}
		if len(out) != n || out != string(s) {
			t.Fatalf("string len=%d: round-trip mismatch", n)
		}
	}
}

func TestEdgeCases_Unicode(t *testing.T) {
	cases := []string{
		"",
		"a",
		"héllo wörld",
		"日本語",
		"🐱🐶",
		"mixed: ascii + 中文 + emoji 🚀",
		string([]byte{0xff, 0xfe, 0xfd}), // invalid UTF-8 — we still must round-trip
	}
	for _, s := range cases {
		b, err := Marshal(s, OptSpeed)
		if err != nil {
			t.Fatalf("encode %q: %v", s, err)
		}
		var out string
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("decode %q: %v", s, err)
		}
		if out != s {
			t.Fatalf("unicode round-trip mismatch: %q vs %q", s, out)
		}
	}
}

// TestInterop_FastEncodedDenseDecoded checks that data written by a Fast
// encoder decodes correctly through a fresh Decoder (which auto-detects
// the mode from header flags), and vice versa.
func TestInterop_ModesCrossDecode(t *testing.T) {
	in := Outer{Name: "alice", Age: 30, Tags: []string{"x"}, Counts: [3]int32{1, 2, 3}}
	// Fast → any decoder
	fast, _ := Marshal(in, OptSpeed)
	var outFast Outer
	if err := Unmarshal(fast, &outFast); err != nil {
		t.Fatalf("decode fast: %v", err)
	}
	if outFast.Name != in.Name {
		t.Fatalf("fast cross-decode mismatch")
	}
	// Dense → any decoder
	dense, _ := Marshal(in, OptBalanced)
	var outDense Outer
	if err := Unmarshal(dense, &outDense); err != nil {
		t.Fatalf("decode dense: %v", err)
	}
	if outDense.Name != in.Name {
		t.Fatalf("dense cross-decode mismatch")
	}
}
