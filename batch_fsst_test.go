package qdf

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

// fsstSrc encodes the wire; fsstDoc is the pointer-free batch target. Body
// carries high-cardinality, heavy-shared-substring URLs so the columnar string
// codec picks FSST (tagColStrFSST); At is a narrow-range timestamp that keeps
// the struct columnar (so the batch fast path — decodeBatchColumnar ->
// scatterBatchColumn(bfStr) -> readStringColumnHandles — is reached, exercising
// the direct-into-slab FSST arm this test covers).
type fsstSrc struct {
	At   time.Time `qdf:"at"`
	Body string    `qdf:"body"`
	ID   int64     `qdf:"id"`
}

type fsstDoc struct {
	ID   int64 `qdf:"id"`
	At   Time  `qdf:"at"`
	Body Str   `qdf:"body"`
}

func mkFSSTSrc(n int) []fsstSrc {
	out := make([]fsstSrc, n)
	for i := range out {
		out[i] = fsstSrc{
			ID:   int64(i),
			At:   time.Unix(1_700_000_000+int64(i), 0).UTC(),
			Body: fmt.Sprintf("https://example.com/api/v1/users/%d/profile?ref=%d&session=abcdef0123456789", i, i*7),
		}
	}
	return out
}

// TestBatchFSSTDirectParity checks the direct-into-slab FSST arm
// (readStringColumnFSSTInto) decodes byte-identically to the source, across an
// FSST-coded columnar batch. For n large enough to compress, it also asserts
// the wire is a columnar struct carrying an FSST string column, so the test
// genuinely exercises the new arm rather than a raw/row-major fallback.
func TestBatchFSSTDirectParity(t *testing.T) {
	for _, n := range []int{0, 1, 64, 1000} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			src := mkFSSTSrc(n)
			data, err := Marshal(src, OptBalanced|OptFSST)
			if err != nil {
				t.Fatal(err)
			}

			if n >= 64 {
				// Columnar wire (batch fast path) carrying an FSST string column.
				d := &Decoder{buf: data}
				tag, perr := d.peekTag()
				if perr != nil {
					t.Fatalf("peekTag: %v", perr)
				}
				if tag != tagColStruct {
					t.Fatalf("want columnar wire (tagColStruct %#x) so the batch FSST arm runs, got %#x", tagColStruct, tag)
				}
				if bytes.IndexByte(data, tagColStrFSST) < 0 {
					t.Fatalf("Body column did not FSST-code (no %#x tag in wire) — test would not exercise readStringColumnFSSTInto", tagColStrFSST)
				}
			}

			b, err := UnmarshalBatch[fsstDoc](data)
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
				if got := b.Str(r.Body); got != src[i].Body {
					t.Fatalf("row %d body = %q, want %q", i, got, src[i].Body)
				}
				if !b.TimeOf(r.At).Equal(src[i].At) {
					t.Fatalf("row %d at = %v, want %v", i, b.TimeOf(r.At), src[i].At)
				}
			}
		})
	}
}

// TestBatchFSSTDirectTruncated feeds every prefix of an FSST-coded batch wire
// through UnmarshalBatch: each must return an error, never panic (the
// direct-into-slab path parses a symbol table + per-row compressed lengths off
// attacker-controlled bytes; every bound must hold).
func TestBatchFSSTDirectTruncated(t *testing.T) {
	src := mkFSSTSrc(128)
	data, err := Marshal(src, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	for k := range len(data) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("prefix len %d panicked: %v", k, r)
				}
			}()
			if b, err := UnmarshalBatch[fsstDoc](data[:k]); err == nil {
				b.Release()
				t.Fatalf("prefix len %d decoded without error", k)
			}
		}()
	}
}

// BenchmarkBatchFSSTDecode measures the FSST batch decode after the
// direct-into-slab change: the temp make([]byte,0,dt64) scratch + one full copy
// pass the general readStringColumnHandles arm used to pay are gone, so allocs/op
// and B/op should sit well below the pre-change ~14 allocs / ~112 KB the headroom
// probe recorded (the residual is the FSST symbol-table decode, out of scope).
func BenchmarkBatchFSSTDecode(b *testing.B) {
	src := mkFSSTSrc(1000)
	data, err := Marshal(src, OptBalanced|OptFSST)
	if err != nil {
		b.Fatal(err)
	}
	warm, err := UnmarshalBatch[fsstDoc](data)
	if err != nil {
		b.Fatal(err)
	}
	warm.Release()

	b.ReportAllocs()
	for b.Loop() {
		bat, err := UnmarshalBatch[fsstDoc](data)
		if err != nil {
			b.Fatal(err)
		}
		bat.Release()
	}
}
