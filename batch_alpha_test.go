package qdf

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// alphaSrc encodes the wire; alphaDoc is the pointer-free batch target. Span is
// a high-cardinality restricted-alphabet (lowercase-hex, 16 symbols) column the
// alpha codec packs (tagColStrAlpha); ID/At keep the struct columnar so the
// batch fast path — decodeBatchColumnar -> scatterBatchColumn(bfStr) ->
// readStringColumnHandles — reaches the direct-into-slab alpha arm under test.
type alphaSrc struct {
	At   time.Time `qdf:"at"`
	Span string    `qdf:"span"`
	ID   int64     `qdf:"id"`
}

type alphaDoc struct {
	ID   int64 `qdf:"id"`
	At   Time  `qdf:"at"`
	Span Str   `qdf:"span"`
}

// mkAlphaSrc builds n rows. fixed=true gives a uniform 16-char span (the
// fixed-length + a==16 SIMD DecodeHex4 path); fixed=false varies the length
// 8..23 (the variable-length lenStart re-scan path). Both use the 16-symbol
// hex alphabet so the column alpha-packs either way.
func mkAlphaSrc(n int, fixed bool) []alphaSrc {
	const hexChars = "0123456789abcdef"
	r := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test corpus
	out := make([]alphaSrc, n)
	for i := range out {
		ln := 16
		if !fixed {
			ln = 8 + r.Intn(16)
		}
		b := make([]byte, ln)
		for j := range b {
			// Random over the 16-symbol hex alphabet: high cardinality (so dict
			// can't dedup it) restricted to <= 64 distinct chars (so alpha packs it).
			b[j] = hexChars[r.Intn(16)]
		}
		out[i] = alphaSrc{
			ID:   int64(i),
			At:   time.Unix(1_700_000_000+int64(i), 0).UTC(),
			Span: string(b),
		}
	}
	return out
}

// TestBatchAlphaDirectParity checks readStringColumnAlphaInto decodes
// byte-identically to the source across an alpha-packed columnar batch, on both
// the fixed- and variable-length span layouts. For n large enough to pack it
// also asserts the wire is a columnar struct carrying an alpha column (0xFB),
// so the test genuinely exercises the new arm.
func TestBatchAlphaDirectParity(t *testing.T) {
	for _, fixed := range []bool{true, false} {
		for _, n := range []int{0, 1, 64, 1000} {
			t.Run(fmt.Sprintf("fixed=%v/n=%d", fixed, n), func(t *testing.T) {
				src := mkAlphaSrc(n, fixed)
				data, err := Marshal(src, OptBalanced)
				if err != nil {
					t.Fatal(err)
				}

				if n >= 64 {
					d := &Decoder{buf: data}
					tag, perr := d.peekTag()
					if perr != nil {
						t.Fatalf("peekTag: %v", perr)
					}
					if tag != tagColStruct {
						t.Fatalf("want columnar wire (tagColStruct %#x), got %#x", tagColStruct, tag)
					}
					// Printable-ASCII bodies never contain 0xFB, so its presence
					// marks the alpha column tag (same discriminator alphapack_test uses).
					if bytes.IndexByte(data, tagColStrAlpha) < 0 {
						t.Fatalf("Span did not alpha-pack (no %#x tag) — would not exercise readStringColumnAlphaInto", tagColStrAlpha)
					}
				}

				b, err := UnmarshalBatch[alphaDoc](data)
				if err != nil {
					t.Fatalf("UnmarshalBatch: %v", err)
				}
				defer b.Release()
				if len(b.Rows) != n {
					t.Fatalf("rows = %d, want %d", len(b.Rows), n)
				}
				for i, r := range b.Rows {
					if r.ID != src[i].ID {
						t.Fatalf("row %d id = %d, want %d", i, r.ID, src[i].ID)
					}
					if got := b.Str(r.Span); got != src[i].Span {
						t.Fatalf("row %d span = %q, want %q", i, got, src[i].Span)
					}
					if !b.TimeOf(r.At).Equal(src[i].At) {
						t.Fatalf("row %d at = %v, want %v", i, b.TimeOf(r.At), src[i].At)
					}
				}
			})
		}
	}
}

// TestBatchAlphaDirectTruncated feeds every prefix of an alpha-packed batch wire
// through UnmarshalBatch: each must error, never panic (readStringColumnAlphaInto
// parses an alphabet + per-row lengths + a bit-packed body off attacker-controlled
// bytes and writes unpacked chars into the slab — every bound must hold).
func TestBatchAlphaDirectTruncated(t *testing.T) {
	for _, fixed := range []bool{true, false} {
		src := mkAlphaSrc(128, fixed)
		data, err := Marshal(src, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		for k := range data {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("fixed=%v prefix %d panicked: %v", fixed, k, r)
					}
				}()
				if b, err := UnmarshalBatch[alphaDoc](data[:k]); err == nil {
					b.Release()
					t.Fatalf("fixed=%v prefix %d decoded without error", fixed, k)
				}
			}()
		}
	}
}

// BenchmarkBatchAlphaDecode measures the alpha batch decode after the
// direct-into-slab change: the temp make([]byte, totalChars) scratch + one copy
// pass the general readStringColumnHandles arm used to pay are gone.
func BenchmarkBatchAlphaDecode(b *testing.B) {
	src := mkAlphaSrc(1000, true)
	data, err := Marshal(src, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	warm, err := UnmarshalBatch[alphaDoc](data)
	if err != nil {
		b.Fatal(err)
	}
	warm.Release()

	b.ReportAllocs()
	for b.Loop() {
		bat, err := UnmarshalBatch[alphaDoc](data)
		if err != nil {
			b.Fatal(err)
		}
		bat.Release()
	}
}
