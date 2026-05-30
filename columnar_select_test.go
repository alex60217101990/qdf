package qdf

import (
	"bytes"
	"strconv"
	"testing"
)

type selFull struct {
	A int64  `qdf:"a"`
	B string `qdf:"b"`
	C int32  `qdf:"c"`
	D bool   `qdf:"d"`
	// E is a high-cardinality (distinct-per-row) string column. The subset
	// type below excludes it, so a subset decode skips a column that would
	// otherwise allocate a string per row — making the alloc saving robust
	// (numeric columns alone decode into reused scratch buffers => no saving).
	E string `qdf:"e"`
}

func mkSelFull(n int) []selFull {
	out := make([]selFull, n)
	for i := range out {
		out[i] = selFull{
			A: int64(i),
			B: []string{"x", "y", "z"}[i%3],
			C: int32(i % 7),
			D: i%2 == 0,
			E: "evt-" + strconv.Itoa(i),
		}
	}
	return out
}

func TestSelect_IndexFullRoundTrip(t *testing.T) {
	rows := mkSelFull(500)
	enc, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var got []selFull
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows) {
		t.Fatalf("len %d != %d", len(got), len(rows))
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d: %+v != %+v", i, got[i], rows[i])
		}
	}
	plain, _ := Marshal(rows, OptBalanced)
	if len(enc) <= len(plain) {
		t.Fatalf("indexed wire %d should be larger than plain %d (carries the index)", len(enc), len(plain))
	}
}

type selSubset struct {
	B string `qdf:"b"`
	D bool   `qdf:"d"`
}

func TestSelect_TypedSubsetSkips(t *testing.T) {
	rows := mkSelFull(500)
	enc, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var got []selSubset
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows) {
		t.Fatalf("len %d != %d", len(got), len(rows))
	}
	for i := range rows {
		if got[i].B != rows[i].B || got[i].D != rows[i].D {
			t.Fatalf("row %d: %+v vs full %+v", i, got[i], rows[i])
		}
	}
	full := testing.AllocsPerRun(20, func() {
		var f []selFull
		_ = Unmarshal(enc, &f)
	})
	sub := testing.AllocsPerRun(20, func() {
		var s []selSubset
		_ = Unmarshal(enc, &s)
	})
	if sub >= full {
		t.Fatalf("subset decode allocs %.0f not below full %.0f (columns not skipped)", sub, full)
	}
}

func TestSelect_FallbackNoIndex(t *testing.T) {
	rows := mkSelFull(200)
	enc, _ := Marshal(rows, OptBalanced) // NO OptColumnIndex
	var got []selSubset
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i].B != rows[i].B || got[i].D != rows[i].D {
			t.Fatalf("row %d fallback mismatch: %+v", i, got[i])
		}
	}
}

func TestSelect_MalformedIndexNoPanic(t *testing.T) {
	rows := mkSelFull(64)
	enc, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	// Flip every byte; selective AND full decode must never panic/hang.
	for i := range enc {
		m := append([]byte(nil), enc...)
		m[i] ^= 0xFF
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on corrupted byte %d: %v", i, r)
				}
			}()
			var s []selSubset
			_ = Unmarshal(m, &s)
			var f []selFull
			_ = Unmarshal(m, &f)
			var mp []map[string]any
			_ = UnmarshalColumns(m, &mp, "b", "d")
		}()
	}
}

func TestSelect_UnmarshalColumnsMap(t *testing.T) {
	rows := mkSelFull(300)
	enc, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := UnmarshalColumns(enc, &out, "b", "d"); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(rows) {
		t.Fatalf("len %d != %d", len(out), len(rows))
	}
	for i := range rows {
		if out[i]["b"].(string) != rows[i].B || out[i]["d"].(bool) != rows[i].D {
			t.Fatalf("row %d: %v", i, out[i])
		}
		if _, present := out[i]["a"]; present {
			t.Fatalf("row %d: column a should have been skipped", i)
		}
	}
}

func TestSelect_StreamEncoderNoCorruption(t *testing.T) {
	var buf bytes.Buffer
	se := NewStreamEncoderWith(&buf, OptBalanced|OptColumnIndex)
	b1, b2 := mkSelFull(50), mkSelFull(60)
	for _, b := range [][]selFull{b1, b2} {
		if err := se.Encode(b); err != nil {
			t.Fatal(err)
		}
		if err := se.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	sd := NewStreamDecoder(&buf)
	var g1, g2 []selFull
	if err := sd.Decode(&g1); err != nil {
		t.Fatalf("decode 1: %v", err)
	}
	if err := sd.Decode(&g2); err != nil {
		t.Fatalf("decode 2 (stream corruption): %v", err)
	}
	if len(g1) != len(b1) || len(g2) != len(b2) {
		t.Fatalf("lens %d/%d vs %d/%d", len(g1), len(g2), len(b1), len(b2))
	}
	for i := range b2 {
		if g2[i] != b2[i] {
			t.Fatalf("b2 row %d corrupted: %+v != %+v", i, g2[i], b2[i])
		}
	}
}
