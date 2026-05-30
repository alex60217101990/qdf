package qdf

import "testing"

type selFull struct {
	A int64  `qdf:"a"`
	B string `qdf:"b"`
	C int32  `qdf:"c"`
	D bool   `qdf:"d"`
}

func mkSelFull(n int) []selFull {
	out := make([]selFull, n)
	for i := range out {
		out[i] = selFull{A: int64(i), B: []string{"x", "y", "z"}[i%3], C: int32(i % 7), D: i%2 == 0}
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
