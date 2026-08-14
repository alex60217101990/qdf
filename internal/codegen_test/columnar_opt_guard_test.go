package cgsample

import (
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// transposed reports whether a wire's root is one of the columnar containers.
// Which one depends on the SHAPE — a struct with a residual field takes the
// hybrid form (0xF7), a fully columnar-eligible one takes the pure form (0xEF) —
// so a test that pins a single tag fails on a legitimate shape change and
// blames the wrong thing.
func transposed(wire []byte) bool {
	return wire[5] == 0xF7 || wire[5] == 0xEF
}

// handRow has a codec of its own that is NOT structural: it writes a fixed
// payload that has nothing to do with its fields. Transposing it would produce a
// completely different value, so it must never be transposed — with the option
// set or not.
type handRow struct {
	ID   string `qdf:"id"`
	Seq  int64  `qdf:"seq"`
	Note string `qdf:"note"`
}

func (v *handRow) MarshalQDF(dst []byte) ([]byte, error) {
	return append(dst, 0xA1, 0x61, 0x78), nil // fixmap{1}: "x" -> ...
}

func (v *handRow) UnmarshalQDF(src []byte) (int, error) { return 3, nil }

func TestHandWrittenCodecIsNeverTransposed(t *testing.T) {
	rows := make([]handRow, 64)
	for i := range rows {
		rows[i] = handRow{ID: "id", Seq: int64(i), Note: "note"}
	}
	plain, err := qdf.Marshal(rows, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	opted, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnarGenerated)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != string(opted) {
		t.Fatalf("the option changed a hand-written codec's wire: %d bytes -> %d — "+
			"MarshalQDF was bypassed and the value is not what the type says it is",
			len(plain), len(opted))
	}
	if transposed(opted) {
		t.Fatalf("a hand-written codec's slice was transposed (root 0x%02x)", opted[5])
	}
}

// The mirror: a GENERATED type in the same module must be transposed, or the test
// above passes because the option does nothing at all.
func TestGeneratedTypeIsTransposedWithTheOption(t *testing.T) {
	rows := ransRows(64)
	off, err := qdf.Marshal(rows, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	on, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnarGenerated)
	if err != nil {
		t.Fatal(err)
	}
	if transposed(off) {
		t.Fatalf("the default already transposes (root 0x%02x)", off[5])
	}
	if !transposed(on) {
		t.Fatalf("the option did not transpose a generated type: root 0x%02x", on[5])
	}
	// Deliberately NO size assertion. Whether transposing SAVES depends on the
	// data and the shape, and this fixture is the case where it loses: ransRows
	// is pure-prefix text, which the row-major per-field string delta codes
	// almost perfectly, while the columnar form must pay a per-column header and
	// cannot see across rows the same way. Measured here: 2197 bytes transposed
	// against 1084 row-major.
	//
	// The saving is real on data whose values differ in the MIDDLE — a 512-element
	// service fixture goes 88,035 -> 54,141 — and the option's documentation says
	// so. What this test pins is that the option DOES something, which is what
	// makes the hand-written guard above meaningful.
	t.Logf("transposed %d bytes, row-major %d — direction is data-dependent by design",
		len(on), len(off))
}
