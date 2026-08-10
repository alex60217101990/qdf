package qdf

import (
	"strconv"
	"testing"
)

type sdWide struct {
	A string `qdf:"a"`
	B string `qdf:"b"`
}

type sdNarrow struct {
	B string `qdf:"b"`
}

// The wire carries a field the target struct does not declare. The decoder
// skips it — and its base must still advance, or the NEXT value of that field
// rebuilds against a stale prefix. The types still line up, so the failure is a
// wrong string rather than an error.
func TestStrDeltaSkippedFieldAdvancesTheBase(t *testing.T) {
	rows := make([]sdWide, 512)
	for i := range rows {
		rows[i] = sdWide{
			A: "/prefix/aaaaaaaaaaaaaaaaaaaa/" + strconv.Itoa(i),
			B: "/prefix/bbbbbbbbbbbbbbbbbbbb/" + strconv.Itoa(i),
		}
	}
	before := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if fired := strDeltaEmitted.Load() - before; fired == 0 {
		t.Fatal("delta never fired — this test would not exercise the skip path")
	}
	var got []sdNarrow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows) {
		t.Fatalf("decoded %d rows, want %d", len(got), len(rows))
	}
	for i := range rows {
		if got[i].B != rows[i].B {
			t.Fatalf("row %d: got %q want %q", i, got[i].B, rows[i].B)
		}
	}
}

// The other direction: the struct declares a field the wire does not carry.
func TestStrDeltaRoundTripsWiderTarget(t *testing.T) {
	rows := make([]sdNarrow, 512)
	for i := range rows {
		rows[i] = sdNarrow{B: "/prefix/bbbbbbbbbbbbbbbbbbbb/" + strconv.Itoa(i)}
	}
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var got []sdWide
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i].B != rows[i].B {
			t.Fatalf("row %d: got %q want %q", i, got[i].B, rows[i].B)
		}
		if got[i].A != "" {
			t.Fatalf("row %d: absent field decoded as %q", i, got[i].A)
		}
	}
}

// Delta values and intern refs interleave on one field. The base has to advance
// across both, and the state chain has to survive the mixture.
func TestStrDeltaInterleavesWithInternRefs(t *testing.T) {
	rows := make([]sdRow, 2048)
	for i := range rows {
		switch i % 3 {
		case 0:
			rows[i] = sdRow{Seq: int64(i), URL: "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+i)}
		case 1:
			rows[i] = sdRow{Seq: int64(i), URL: "/healthz/ready/probe/endpoint"}
		default:
			rows[i] = sdRow{Seq: int64(i), URL: "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+i-1)}
		}
	}
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var got []sdRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d: got %+v want %+v", i, got[i], rows[i])
		}
	}
}

// Nested structs must not share a base with their parent.
func TestStrDeltaNestedStructs(t *testing.T) {
	type inner struct {
		Path string `qdf:"path"`
	}
	type outer struct {
		Host string `qdf:"host"`
		In   inner  `qdf:"in"`
	}
	rows := make([]outer, 512)
	for i := range rows {
		rows[i] = outer{
			Host: "host-eu-central-1-node-" + strconv.Itoa(i),
			In:   inner{Path: "/very/long/shared/path/segment/" + strconv.Itoa(i)},
		}
	}
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var got []outer
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d: got %+v want %+v", i, got[i], rows[i])
		}
	}
}

// A stream reuses encoder state across messages, so the base crosses message
// boundaries on both sides or on neither.
func TestStrDeltaAcrossStreamedMessages(t *testing.T) {
	mk := func(off int) []sdRow {
		out := make([]sdRow, 256)
		for i := range out {
			out[i] = sdRow{Seq: int64(off + i), URL: "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+off+i)}
		}
		return out
	}
	for _, off := range []int{0, 256, 512} {
		rows := mk(off)
		b, err := Marshal(rows, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		var got []sdRow
		if err := Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			if got[i] != rows[i] {
				t.Fatalf("offset %d row %d: got %+v want %+v", off, i, got[i], rows[i])
			}
		}
	}
}

// The batch decoder is the third independent reader of struct values. It must
// advance a delta base for a field its plan does not carry, exactly as
// decodeStruct and Skip do — a base left a row behind rebuilds the next value
// against the wrong prefix, and the types still line up so nothing errors.
func TestStrDeltaBatchSkippedFieldAdvancesTheBase(t *testing.T) {
	rows := make([]sdWide, 512)
	for i := range rows {
		rows[i] = sdWide{
			A: "/prefix/aaaaaaaaaaaaaaaaaaaa/" + strconv.Itoa(i),
			B: "/prefix/bbbbbbbbbbbbbbbbbbbb/" + strconv.Itoa(i),
		}
	}
	before := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if fired := strDeltaEmitted.Load() - before; fired == 0 {
		t.Fatal("delta never fired — this test would not exercise the batch skip path")
	}
	bat, err := UnmarshalBatch[sdBatchNarrow](b)
	if err != nil {
		t.Fatal(err)
	}

	defer bat.Release()
	got := bat.Rows
	if len(got) != len(rows) {
		t.Fatalf("decoded %d rows, want %d", len(got), len(rows))
	}
	for i := range rows {
		if v := bat.Str(got[i].B); v != rows[i].B {
			t.Fatalf("row %d: got %q want %q", i, v, rows[i].B)
		}
	}
}

// sdBatchNarrow mirrors sdNarrow for the batch decoder, which requires Str
// rather than string so decoded values can alias the batch slab.
type sdBatchNarrow struct {
	B Str `qdf:"b"`
}
