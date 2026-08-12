package cgsample

import (
	"fmt"
	"os"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// scopeRows gives the outer and inner string fields DIFFERENT constant stems,
// and varies only the tail within each.
//
// Both halves matter and pull against each other. Consecutive values of one
// field must share a long prefix, or the delta has nothing to code and the test
// measures an idle codec — an earlier version alternated the stems row by row,
// which made every field's neighbour unlike it and left the codecs at 1.01x.
// The two FIELDS must differ from each other, or a value coded against the
// wrong field's base still decodes correctly by accident and a mis-binding is
// invisible.
//
// Constant-per-field, different-between-fields satisfies both.
func scopeRows(n int) []GenRow {
	const (
		outerStem = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		innerStem = "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"
	)
	out := make([]GenRow, n)
	for i := range out {
		out[i] = GenRow{
			ID:   int64(i),
			Name: outerStem + fmt.Sprintf("-name%04d", i),
			Inner: GenRowInner{
				X: i,
				Y: innerStem + fmt.Sprintf("-y%04d", i),
			},
		}
	}
	return out
}

// A nested generated struct must code its string fields against ITS OWN
// per-field state, not the enclosing type's.
//
// Generated code binds a field scope inside EncodeQDF for exactly this reason.
// When the scope was bound only by the slice loop, a nested struct reached
// through EncodeNested kept its parent's binding: no error, no short read, just
// a string rebuilt from a neighbouring field's prefix. Under OptStringAlphabet
// it turns into a hard failure instead — the shared slot declares a table under
// the child's shape and the parent's next reference reports an unknown state.
func TestNestedStructUsesItsOwnFieldScope(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	want := scopeRows(32)

	// The codecs must be ACTIVE, or everything below passes with them switched
	// off — which is exactly what happens if the scope is never bound at all,
	// and a round-trip alone cannot tell "correctly bound" from "absent". The
	// anchor is self-referential rather than a byte count: OptSpeed encodes the
	// same value with no codecs, and on this fixture they are worth several
	// times their absence.
	bare, err := qdf.Marshal(GenRowSet{Rows: want}, qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	coded, err := qdf.Marshal(GenRowSet{Rows: want}, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if worth := float64(len(bare)) / float64(len(coded)); worth < 1.5 {
		t.Fatalf("generated encoder wrote %d bytes with codecs against %d without, only %.2fx — "+
			"the field scope looks unbound, and the round-trips below would then pass "+
			"with the feature simply off", len(coded), len(bare), worth)
	}

	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"balanced", qdf.OptBalanced},
		{"balanced+alpha", qdf.OptBalanced | qdf.OptStringAlphabet},
		{"compression", qdf.OptCompression},
	} {
		b, err := qdf.Marshal(GenRowSet{Rows: want}, o.opts)
		if err != nil {
			t.Fatalf("%s: marshal: %v", o.name, err)
		}
		var got GenRowSet
		if err := qdf.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: unmarshal: %v", o.name, err)
		}
		if len(got.Rows) != len(want) {
			t.Fatalf("%s: %d rows, want %d", o.name, len(got.Rows), len(want))
		}
		for i := range want {
			if got.Rows[i].Name != want[i].Name {
				t.Fatalf("%s row %d Name:\n got %q\nwant %q — the outer field was coded "+
					"against the nested struct's base", o.name, i, got.Rows[i].Name, want[i].Name)
			}
			if got.Rows[i].Inner.Y != want[i].Inner.Y {
				t.Fatalf("%s row %d Inner.Y:\n got %q\nwant %q", o.name, i,
					got.Rows[i].Inner.Y, want[i].Inner.Y)
			}
		}
	}
}

// wideRow and narrowRow model a producer that writes a field the consumer does
// not declare. The consumer's generated decoder routes it to Skip().
type wideRow struct {
	A string `qdf:"a"`
	B string `qdf:"b"`
	C string `qdf:"c"`
}

// A dropped string field must be SKIPPED, not rejected — and skipping it has to
// advance that field's state.
//
// Skip() had no case for tagStrDelta or tagStrAlpha, so it fell through to
// ErrBadTag. The reflect decoder never showed it: for a dropped field it reads
// through readStringBytes rather than calling Skip. Generated decoders do call
// Skip, and this branch is what first makes generated ENCODERS emit those tags,
// so the gap became reachable from any producer.
//
// Stepping over the bytes would not be enough either: a skipped delta whose
// base did not advance leaves the next value of that field rebuilding against a
// stale prefix, and a skipped alphabet declaration leaves the table unrecorded.
func TestDroppedStringFieldIsSkippable(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	rows := make([]wideRow, 64)
	for i := range rows {
		rows[i] = wideRow{
			A: fmt.Sprintf("%032x", uint64(i)*11400714819323198485),
			B: "com.acme.platform.worker." + fmt.Sprintf("%06d", i),
			C: fmt.Sprintf("%032x", uint64(i)*2654435761),
		}
	}
	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"balanced", qdf.OptBalanced},
		{"balanced+alpha", qdf.OptBalanced | qdf.OptStringAlphabet},
	} {
		b, err := qdf.Marshal(rows, o.opts)
		if err != nil {
			t.Fatalf("%s: marshal: %v", o.name, err)
		}
		// The narrow shape drops B, so every row's middle field is skipped.
		var got []struct {
			A string `qdf:"a"`
			C string `qdf:"c"`
		}
		if err := qdf.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: decoding into the narrow shape: %v", o.name, err)
		}
		for i := range rows {
			if got[i].A != rows[i].A || got[i].C != rows[i].C {
				t.Fatalf("%s row %d: got A=%q C=%q, want A=%q C=%q — a skipped field "+
					"left the surviving ones coding against a stale base",
					o.name, i, got[i].A, got[i].C, rows[i].A, rows[i].C)
			}
		}
	}
}
