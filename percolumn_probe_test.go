package qdf

import (
	"reflect"
	"testing"
	"unsafe"
)

// mixedCol is a struct whose columns disagree about the container: a monotonic
// timestamp that Delta/FOR crushes, and free text that no columnar codec helps
// today. One verdict for the whole struct cannot serve both.
type mixedCol struct {
	Ts   int64  `qdf:"ts"`
	Text string `qdf:"text"`
}

// The text is drawn from an alphabet wider than the alphabet packer's 64-byte
// limit and every value is distinct, so no columnar string codec on this branch
// helps it: per-value costs what row-major costs. The timestamp beside it is
// monotonic, which Delta/FOR crushes. One verdict cannot serve both.
func mkMixedCol(n int) []mixedCol {
	out := make([]mixedCol, n)
	var b [48]byte
	for i := range out {
		for j := range b {
			// 96 distinct byte values, spread so no short prefix repeats.
			b[j] = byte(32 + (i*31+j*17)%96)
		}
		out[i] = mixedCol{Ts: int64(1700000000 + i), Text: string(b[:])}
	}
	return out
}

func TestProbeReportsPerColumn(t *testing.T) {
	rows := mkMixedCol(2048)
	td, err := descOf(reflect.TypeFor[mixedCol]())
	if err != nil {
		t.Fatal(err)
	}
	plan := buildColumnarPlan(td)
	if plan == nil {
		t.Fatal("flat scalar+string struct must be columnar-eligible")
	}

	keep := make([]bool, len(plan.cols))
	base := unsafe.Pointer(&rows[0])
	colBytes, rowBytes := columnarProbeColumns(plan, base, len(rows), false, nil, true, keep)
	if colBytes <= 0 || rowBytes <= 0 {
		t.Fatalf("totals not populated: col=%d row=%d", colBytes, rowBytes)
	}

	tsIdx, textIdx := -1, -1
	for i := range plan.cols {
		switch plan.cols[i].name {
		case "ts":
			tsIdx = i
		case "text":
			textIdx = i
		}
	}
	if tsIdx < 0 || textIdx < 0 {
		t.Fatalf("columns not found: ts=%d text=%d", tsIdx, textIdx)
	}

	// The two verdicts must differ. If they agree the probe is not
	// discriminating per column, which is the whole point of the split.
	if !keep[tsIdx] {
		t.Error("monotonic int64 column reported as not worth transposing")
	}
	if keep[textIdx] {
		t.Error("free-text column reported as worth transposing, before the front-delta term exists")
	}
}

// The split must not change what the old entry point decides: it applies the
// same gate to the same totals.
func TestProbeWrapperAgreesWithColumns(t *testing.T) {
	rows := mkMixedCol(2048)
	td, err := descOf(reflect.TypeFor[mixedCol]())
	if err != nil {
		t.Fatal(err)
	}
	plan := buildColumnarPlan(td)
	base := unsafe.Pointer(&rows[0])

	for _, internAware := range []bool{false, true} {
		keep := make([]bool, len(plan.cols))
		colBytes, rowBytes := columnarProbeColumns(plan, base, len(rows), false, nil, internAware, keep)
		gain := columnarMinGainPct
		if internAware {
			gain = columnarMinGainPctInternAware
		}
		want := rowBytes != 0 && colBytes*100 <= rowBytes*(100-gain)
		if got := columnarProbe(plan, base, len(rows), false, nil, internAware); got != want {
			t.Errorf("internAware=%v: wrapper says %v, totals say %v", internAware, got, want)
		}
	}
}
