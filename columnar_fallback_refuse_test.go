package qdf

import (
	"testing"
)

// badRow has a codec of its own AND a field no structural walk can describe, so
// the fallback must refuse it. The producer is a different type entirely — a
// plain struct that encodes columnar — which is the only way this path is
// reachable: nothing could ever WRITE a columnar frame from badRow itself.
type badRow struct {
	ID string   `qdf:"id"`
	Ch chan int `qdf:"ch"`
}

func (v *badRow) MarshalQDF(dst []byte) ([]byte, error)      { return dst, nil }
func (v *badRow) UnmarshalQDF(src []byte) (n int, err error) { return 0, nil }

type goodRow struct {
	ID   string `qdf:"id"`
	Seq  int64  `qdf:"seq"`
	Note string `qdf:"note"`
}

// An element the fallback cannot describe must be refused ONCE, not re-walked on
// every decode.
//
// The refusal itself is the cheap half to get right. The expensive half is that
// without remembering it, each decode rebuilds the element's whole field list —
// allocating descriptors — and throws it away, forever, on the one path that
// gains nothing. Asserted by allocation count rather than by timing: a walk that
// no longer happens cannot allocate.
func TestColumnarFallbackRefusesOnlyOnce(t *testing.T) {
	rows := make([]goodRow, 64)
	for i := range rows {
		rows[i] = goodRow{
			ID:   "svc-" + string(rune('a'+i%26)),
			Seq:  int64(i),
			Note: "a note that is long enough to be worth columnarising",
		}
	}
	wire, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if wire[5] != tagHybridColStruct && wire[5] != tagColStruct {
		t.Fatalf("the fixture wire is 0x%02x, not columnar — this test would never reach "+
			"the fallback and would pass without exercising anything", wire[5])
	}

	decode := func() {
		var into []badRow
		if err := Unmarshal(wire, &into); err == nil {
			t.Fatal("an element with a channel field decoded a columnar frame")
		}
	}

	decode() // first refusal: pays for the walk that fails
	steady := testing.AllocsPerRun(50, decode)
	if steady > 8 {
		t.Errorf("a refused element still allocates %.0f objects per decode — the refusal "+
			"is not being remembered and the field walk runs every time", steady)
	}
}
