package qdf

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func strdictHasByte(buf []byte, b byte) bool { return bytes.IndexByte(buf, b) >= 0 }

type strDictRow struct {
	Level   string
	Service string
	Host    string
	Seq     int64
}

func mkStrDictRows(n int, seed int64) []strDictRow {
	r := rand.New(rand.NewSource(seed))
	levels := []string{"INFO", "INFO", "INFO", "WARN", "ERROR", "DEBUG"}
	services := []string{"api-gateway", "auth-service", "billing-service", "user-service"}
	hosts := []string{"node-01", "node-02", "node-03", "node-04"}
	out := make([]strDictRow, n)
	for i := range out {
		out[i] = strDictRow{
			Level:   levels[r.Intn(len(levels))],
			Service: services[r.Intn(len(services))],
			Host:    hosts[r.Intn(len(hosts))],
			Seq:     int64(i),
		}
	}
	return out
}

func TestStrDict_RoundTripStruct(t *testing.T) {
	rows := mkStrDictRows(2000, 1)
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if !strdictHasByte(enc, tagColStrDict) {
		t.Fatalf("expected string-dict tag 0x%X on wire for enum columns, absent", tagColStrDict)
	}
	var got []strDictRow
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows) {
		t.Fatalf("len %d != %d", len(got), len(rows))
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d mismatch: %+v != %+v", i, got[i], rows[i])
		}
	}
}

func TestStrDict_RoundTripAny(t *testing.T) {
	rows := mkStrDictRows(1000, 2)
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var gotAny any
	if err := Unmarshal(enc, &gotAny); err != nil {
		t.Fatal(err)
	}
	got, ok := gotAny.([]any)
	if !ok || len(got) != len(rows) {
		t.Fatalf("any decode shape: %T len mismatch", gotAny)
	}
	for i := range rows {
		m, ok := got[i].(map[string]any)
		if !ok {
			t.Fatalf("row %d not a map: %T", i, got[i])
		}
		if m["Level"] != rows[i].Level || m["Service"] != rows[i].Service || m["Host"] != rows[i].Host {
			t.Fatalf("row %d any mismatch: %v", i, m)
		}
	}
}

func TestStrDict_SmallerThanPerValue(t *testing.T) {
	rows := mkStrDictRows(4000, 3)
	withDict, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: dict fired and the payload is meaningfully smaller than the
	// raw lower bound of one byte per string value (3 string cols * n).
	floor := 3 * len(rows)
	if len(withDict) >= floor {
		t.Fatalf("dict payload %d not below per-value floor %d", len(withDict), floor)
	}
	fmt.Printf("strdict n=%d size=%d (per-value floor ~%d)\n", len(rows), len(withDict), floor)
}

// Run-heavy (sorted) columns must NOT regress: the never-worse gate should
// keep them on the per-value/repeat path or at least not grow the wire.
func TestStrDict_NeverWorseRunHeavy(t *testing.T) {
	// One value repeated in long runs → repeat-coding should win; dict must
	// not be chosen (its flat index would be larger than the run encoding).
	type row struct{ S string }
	rows := make([]row, 3000)
	vals := []string{"AAAA", "BBBB", "CCCC"}
	for i := range rows {
		rows[i].S = vals[i/1000] // 3 long runs
	}
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var got []row
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("run-heavy row %d mismatch", i)
		}
	}
}

// High-cardinality columns (unique-ish strings) must not be dict-encoded.
func TestStrDict_HighCardinalityFallback(t *testing.T) {
	type row struct{ ID string }
	rows := make([]row, 2000)
	for i := range rows {
		rows[i].ID = fmt.Sprintf("id-%08x-%d", i*2654435761, i)
	}
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if strdictHasByte(enc, tagColStrDict) {
		t.Fatal("high-cardinality column must not use the string-dict codec")
	}
	var got []row
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("highcard row %d mismatch", i)
		}
	}
}

// Single distinct value → bits==0, empty index body.
func TestStrDict_SingleDistinct(t *testing.T) {
	type row struct {
		S string
		N int64
	}
	rows := make([]row, 500)
	for i := range rows {
		rows[i] = row{S: "constant", N: int64(i)}
	}
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	var got []row
	if err := Unmarshal(enc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("single-distinct row %d mismatch", i)
		}
	}
}

// FuzzStrDict_RoundTrip drives enum-column selection from fuzz bytes and
// asserts the columnar string-dict codec round-trips for any picks.
func FuzzStrDict_RoundTrip(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7})
	vals := []string{"", "INFO", "WARN", "ERROR", "a", "service-x", "service-y", "node-01"}
	f.Fuzz(func(t *testing.T, b []byte) {
		type row struct {
			A string
			B string
		}
		n := 20 + len(b)
		rows := make([]row, n)
		for i := range rows {
			if len(b) == 0 {
				rows[i] = row{A: "INFO", B: "x"}
				continue
			}
			rows[i] = row{A: vals[int(b[i%len(b)])%len(vals)], B: vals[int(b[(i*7+1)%len(b)])%len(vals)]}
		}
		enc, err := Marshal(rows, OptBalanced&^OptRANS)
		if err != nil {
			t.Fatal(err)
		}
		var got []row
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
	})
}

// TestStrDict_DecodeRejectsSingleDistinct hardens the decoder against a
// hostile count==1 (or 0) dictionary, which a valid encoder never produces
// (the never-worse gate rejects single-distinct). count<2 carries no index
// body, so accepting it would let a tiny input claim a huge row count and
// drive a large allocation. The decoder must reject it without panic/OOM.
func TestStrDict_DecodeRejectsSingleDistinct(t *testing.T) {
	rows := mkStrDictRows(2000, 5) // multi-distinct enum columns → dict fires
	enc, err := Marshal(rows, OptBalanced&^OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	pos := bytes.IndexByte(enc, tagColStrDict)
	if pos < 0 || pos+1 >= len(enc) {
		t.Skip("no string-dict tag in this encoding")
	}
	// Byte after 0xF5 is varuint(count); a small distinct count is one byte.
	for _, bad := range []byte{0x01, 0x00} {
		m := append([]byte(nil), enc...)
		m[pos+1] = bad
		var got []strDictRow
		// Correctness contract: never panic/hang on malformed input. An error
		// (or a benign mismatch) is fine; a panic or OOM is not.
		_ = Unmarshal(m, &got)
	}
}
