package qdf

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// multiSrc encodes a batch with FOUR string columns that the encoder codes
// differently — a URL column (FSST), a hex column (alpha), a high-entropy
// random column (raw), and a low-cardinality category column (dict) — so the
// batch decode runs its FSST/alpha/raw/dict arms in the SAME slab, each 2nd+
// column starting at a non-zero slab base. This is the coverage the
// single-string-column tests miss: it catches any base/offset error in
// readStringColumnFSSTInto / readStringColumnAlphaInto where a handle would
// otherwise resolve against the wrong slab region.
type multiSrc struct {
	At  time.Time `qdf:"at"`
	URL string    `qdf:"url"` // FSST: shared-substring URLs
	Hex string    `qdf:"hex"` // alpha: 16-symbol hex
	Rnd string    `qdf:"rnd"` // raw: high-entropy ASCII
	Cat string    `qdf:"cat"` // dict: few distinct values
	ID  int64     `qdf:"id"`
}

type multiDoc struct {
	ID  int64 `qdf:"id"`
	At  Time  `qdf:"at"`
	URL Str   `qdf:"url"`
	Hex Str   `qdf:"hex"`
	Rnd Str   `qdf:"rnd"`
	Cat Str   `qdf:"cat"`
}

func mkMultiSrc(n int) []multiSrc {
	const hexChars = "0123456789abcdef"
	const asciiChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	cats := []string{"info", "warn", "error", "debug"}
	r := rand.New(rand.NewSource(7)) //nolint:gosec // deterministic test corpus
	out := make([]multiSrc, n)
	for i := range out {
		hx := make([]byte, 16)
		for j := range hx {
			hx[j] = hexChars[r.Intn(16)]
		}
		rn := make([]byte, 24)
		for j := range rn {
			rn[j] = asciiChars[r.Intn(len(asciiChars))]
		}
		out[i] = multiSrc{
			ID:  int64(i),
			At:  time.Unix(1_700_000_000+int64(i), 0).UTC(),
			URL: fmt.Sprintf("https://example.com/api/v1/resource/%d/detail?ref=%d", i, i*3),
			Hex: string(hx),
			Rnd: string(rn),
			Cat: cats[i%len(cats)],
		}
	}
	return out
}

// TestBatchMultiColumnParity round-trips a four-string-column batch and asserts
// every field decodes byte-identically via UnmarshalBatch — against the source
// AND against the authoritative reflect Unmarshal into the string-twin. Because
// the columns share one slab, columns 2..4 exercise the FSST/alpha direct-into-
// slab arms at a non-zero base; a base/offset bug corrupts a later column's
// strings, which this catches.
func TestBatchMultiColumnParity(t *testing.T) {
	for _, n := range []int{0, 1, 64, 1000} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			src := mkMultiSrc(n)
			data, err := Marshal(src, OptBalanced|OptFSST)
			if err != nil {
				t.Fatal(err)
			}

			if n >= 64 {
				// Confirm the two direct-into-slab arms are actually exercised:
				// FSST (0xF6) and alpha (0xFB) tags both present in the columnar wire.
				if bytes.IndexByte(data, tagColStrFSST) < 0 {
					t.Fatalf("URL column did not FSST-code (no %#x) — FSST arm not exercised", tagColStrFSST)
				}
				if bytes.IndexByte(data, tagColStrAlpha) < 0 {
					t.Fatalf("Hex column did not alpha-pack (no %#x) — alpha arm not exercised", tagColStrAlpha)
				}
			}

			// Authoritative baseline: reflect Unmarshal into the string-twin.
			var want []multiSrc
			if err := Unmarshal(data, &want); err != nil {
				t.Fatalf("reflect Unmarshal: %v", err)
			}

			b, err := UnmarshalBatch[multiDoc](data)
			if err != nil {
				t.Fatalf("UnmarshalBatch: %v", err)
			}
			defer b.Release()
			if len(b.Rows) != n || len(want) != n {
				t.Fatalf("rows batch=%d reflect=%d want=%d", len(b.Rows), len(want), n)
			}
			for i, r := range b.Rows {
				// Compare every string field against BOTH the source and the
				// reflect decode — a base/offset bug shows as a wrong or shifted
				// body in exactly one of the later columns.
				if b.Str(r.URL) != src[i].URL || b.Str(r.URL) != want[i].URL {
					t.Fatalf("row %d URL: batch=%q src=%q reflect=%q", i, b.Str(r.URL), src[i].URL, want[i].URL)
				}
				if b.Str(r.Hex) != src[i].Hex || b.Str(r.Hex) != want[i].Hex {
					t.Fatalf("row %d Hex: batch=%q src=%q reflect=%q", i, b.Str(r.Hex), src[i].Hex, want[i].Hex)
				}
				if b.Str(r.Rnd) != src[i].Rnd || b.Str(r.Rnd) != want[i].Rnd {
					t.Fatalf("row %d Rnd: batch=%q src=%q reflect=%q", i, b.Str(r.Rnd), src[i].Rnd, want[i].Rnd)
				}
				if b.Str(r.Cat) != src[i].Cat || b.Str(r.Cat) != want[i].Cat {
					t.Fatalf("row %d Cat: batch=%q src=%q reflect=%q", i, b.Str(r.Cat), src[i].Cat, want[i].Cat)
				}
				if r.ID != src[i].ID {
					t.Fatalf("row %d ID: batch=%d src=%d", i, r.ID, src[i].ID)
				}
			}
		})
	}
}

// TestBatchMultiColumnTruncated: every prefix of the four-column wire errors,
// never panics — the FSST/alpha arms write into the slab off attacker-controlled
// lengths at a non-zero base, so bounds must hold there too.
func TestBatchMultiColumnTruncated(t *testing.T) {
	src := mkMultiSrc(96)
	data, err := Marshal(src, OptBalanced|OptFSST)
	if err != nil {
		t.Fatal(err)
	}
	for k := range len(data) {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("prefix %d panicked: %v", k, rec)
				}
			}()
			if b, err := UnmarshalBatch[multiDoc](data[:k]); err == nil {
				b.Release()
				t.Fatalf("prefix %d decoded without error", k)
			}
		}()
	}
}
