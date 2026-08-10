package qdf

import (
	"strconv"
	"testing"
)

type sdRow struct {
	Seq int64  `qdf:"seq"`
	URL string `qdf:"url"`
}

// Count firings, not bytes. A size assertion here would pass on interning
// alone — which is exactly how this branch's columnar codec looked healthy
// while never running once.
func TestStrDeltaFiresOnPrefixSharedField(t *testing.T) {
	rows := make([]sdRow, 2048)
	for i := range rows {
		rows[i] = sdRow{Seq: int64(i), URL: "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+i)}
	}
	before := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if fired := strDeltaEmitted.Load() - before; fired == 0 {
		t.Fatal("delta never fired on a field of prefix-shared URLs")
	}
	var got []sdRow
	if err := Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows) {
		t.Fatalf("decoded %d rows, want %d", len(got), len(rows))
	}
	for i := range rows {
		if got[i] != rows[i] {
			t.Fatalf("row %d: got %+v want %+v", i, got[i], rows[i])
		}
	}
}

// A repeated value costs one byte today (tagStateRepeat). The delta form needs
// at least three, so it must not be chosen — this is the degradation the design
// most has to avoid, and pfx == len(s) is exactly where the delta looks cheap.
func TestStrDeltaDoesNotDisplaceRepeats(t *testing.T) {
	rows := make([]sdRow, 2048)
	for i := range rows {
		rows[i] = sdRow{Seq: int64(i), URL: "/healthz/ready/probe/endpoint"}
	}
	before := strDeltaEmitted.Load()
	b, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if fired := strDeltaEmitted.Load() - before; fired != 0 {
		t.Fatalf("delta fired %d times on a constant field where a repeat costs one byte", fired)
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

// The wire must never grow. Every emission is chosen by an exact byte
// comparison against the form the encoder would otherwise write, so a payload
// that fires must come out no larger than one that cannot.
func TestStrDeltaNeverGrowsTheWire(t *testing.T) {
	mk := func(n int, f func(i int) string) []sdRow {
		out := make([]sdRow, n)
		for i := range out {
			out[i] = sdRow{Seq: int64(i), URL: f(i)}
		}
		return out
	}
	shapes := map[string]func(i int) string{
		"prefix-shared": func(i int) string { return "/api/v1/tenants/9f3a/users/" + strconv.Itoa(100000+i) },
		"all-distinct":  func(i int) string { return strconv.Itoa(i*2654435761) + "-" + strconv.Itoa(i*40503) },
		"constant":      func(i int) string { return "/healthz/ready/probe/endpoint" },
		"short":         func(i int) string { return strconv.Itoa(i % 7) },
	}
	for name, f := range shapes {
		rows := mk(1024, f)
		b, err := Marshal(rows, OptBalanced)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var got []sdRow
		if err := Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i := range rows {
			if got[i] != rows[i] {
				t.Fatalf("%s row %d: got %+v want %+v", name, i, got[i], rows[i])
			}
		}
		t.Logf("%-14s wire=%d", name, len(b))
	}
}
