package rans

import (
	"bytes"
	"testing"
)

func TestTagRoundTripSingle(t *testing.T) {
	for _, src := range [][]byte{
		{}, {0}, {1, 1, 1, 1}, bytes.Repeat([]byte("hello world "), 100),
	} {
		enc := Encode(nil, src)
		got, err := Decode(enc, len(src))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !bytes.Equal(got, src) {
			t.Fatalf("round-trip mismatch: got %q want %q", got, src)
		}
	}
	enc := Encode(nil, bytes.Repeat([]byte("x"), 50))
	if enc[0] != ransTagSingle {
		t.Fatalf("expected tag 0, got %d", enc[0])
	}
}

func mkSkewed(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte((i / 5) & 0x0F) // few hot symbols
	}
	return b
}

func TestInterleavedRoundTrip(t *testing.T) {
	inputs := [][]byte{
		{},
		{7},
		{1, 2},
		{1, 2, 3},
		{1, 2, 3, 4, 5},
		[]byte("aaaaaaaaaabbbbbbccccd"),
		mkSkewed(4096), mkSkewed(100000),
	}
	for _, N := range []int{4} {
		for _, src := range inputs {
			freq, cum := buildFreqs(src)
			var blob []byte
			blob = append(blob, byte(N))
			blob = appendTable(blob, &freq)
			blob = appendInterleaved(blob, src, &freq, &cum, N)
			got, err := Decode(blob, len(src))
			if err != nil {
				t.Fatalf("N=%d len=%d decode: %v", N, len(src), err)
			}
			if !bytes.Equal(got, src) {
				t.Fatalf("N=%d len=%d mismatch", N, len(src))
			}
		}
	}
}

func TestAppendInterleavedStructure(t *testing.T) {
	src := []byte("the quick brown fox jumps over the lazy dog, again and again")
	freq, cum := buildFreqs(src)
	for _, N := range []int{4} {
		blob := appendInterleaved(nil, src, &freq, &cum, N)
		// blob = N*4 states + (N-1) uvarint lengths + substream bytes; must be non-empty.
		if len(blob) < N*4 {
			t.Fatalf("N=%d: blob too short (%d)", N, len(blob))
		}
	}
}

func TestInterleavedHostile(t *testing.T) {
	src := mkSkewed(50000)
	good := Encode(nil, src) // adaptive → interleaved (≥ interleaveMinBytes)
	if good[0] != ransTagInter4 {
		t.Fatalf("setup: expected interleaved blob, tag=%d", good[0])
	}
	// (a) bad tag
	bad := append([]byte{9}, good[1:]...)
	if _, err := Decode(bad, len(src)); err == nil {
		t.Fatal("bad tag must error")
	}
	// (b) truncation at every length must never panic
	for cut := 1; cut < len(good); cut += 7 {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncation at %d: %v", cut, r)
				}
			}()
			_, _ = Decode(good[:cut], len(src))
		}()
	}
	// (c) flipping framing bytes must error/return, never OOB-panic
	for i := 1; i < min(len(good), 64); i++ {
		flipped := append([]byte(nil), good...)
		flipped[i] ^= 0xFF
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on flip at %d: %v", i, r)
				}
			}()
			_, _ = Decode(flipped, len(src))
		}()
	}
}

func TestEncodeAdaptiveTag(t *testing.T) {
	small := bytes.Repeat([]byte("ab"), 8) // below threshold
	large := mkSkewed(200000)              // above threshold
	if Encode(nil, small)[0] != ransTagSingle {
		t.Fatal("small body must be single-stream (tag 0)")
	}
	if Encode(nil, large)[0] != ransTagInter4 {
		t.Fatalf("large body must be interleaved (tag %d)", ransTagInter4)
	}
	// both round-trip
	for _, src := range [][]byte{small, large} {
		got, err := Decode(Encode(nil, src), len(src))
		if err != nil || !bytes.Equal(got, src) {
			t.Fatalf("round-trip: err=%v", err)
		}
	}
}
