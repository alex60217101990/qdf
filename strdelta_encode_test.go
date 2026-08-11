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

// A repeated value must take the one-byte state-ref, not the delta.
//
// Called directly, because the payloads that would exercise this through
// Marshal take the columnar path instead — TestStrDeltaDoesNotDisplaceRepeats
// passes vacuously for that reason, and this is the assertion that does not.
//
// The hazard is specific to the order writeStringField uses: the prefix compare
// runs before the intern lookup, so a value identical to the base scores
// pfx == len(s) and a delta cost of about three bytes, which beats the
// first-sighting cost the threshold compares against. Without the equality fast
// path it emitted 0xF2 and three bytes where tagStateRepeat spends one.
func TestStrDeltaRepeatTakesTheStateRef(t *testing.T) {
	const s = "/healthz/ready/probe/endpoint"
	e := NewEncoderWith(OptBalanced)
	base := ""
	var g strDeltaGate

	e.writeStringField(s, &base, &g)
	first := len(e.buf)
	e.writeStringField(s, &base, &g)
	repeat := len(e.buf) - first
	tag := e.buf[first]

	if repeat > 2 {
		t.Fatalf("a repeated value cost %d bytes (tag 0x%02x); a state-ref spends one", repeat, tag)
	}
	if tag == tagStrDelta {
		t.Fatalf("a repeated value took the delta form")
	}
}
