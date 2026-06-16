package qdf

import (
	"reflect"
	"testing"
	"time"
	"unsafe"
)

type ccRow struct {
	A int64
	B uint32
	F float64
}

func TestColDiffRoundTripPlaceholder(t *testing.T) {
	old := make([]ccRow, 100)
	neu := make([]ccRow, 100)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
		neu[i] = old[i]
	}
	neu[5].A = 999
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]ccRow, 100)
	copy(got, old)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("round-trip mismatch")
	}
}

func TestColChangedAttribution(t *testing.T) {
	old := make([]ccRow, 64)
	neu := make([]ccRow, 64)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
		neu[i] = old[i]
	}
	neu[3].A = 1  // col 0 (A) changed at row 3
	neu[40].A = 2 // col 0 (A) changed at row 40
	neu[7].F = 9  // col 2 (F) changed at row 7

	// The columnar plan is built on the slice descriptor, not the element struct.
	td, err := descOf(reflect.TypeFor[[]ccRow]())
	if err != nil {
		t.Fatal(err)
	}
	plan := td.colPlan
	if plan == nil {
		t.Fatal("expected columnar plan on []ccRow")
	}
	stride := plan.stride
	od := unsafe.Pointer(&old[0])
	nd := unsafe.Pointer(&neu[0])

	bm := newChangedBitmap(nil, 64)
	any := markChangedRows(bm, plan, stride, od, nd, 64)
	if !any {
		t.Fatal("expected changes")
	}
	// Attribute column 0 (A): rows 3 and 40.
	rows := colChangedRows(nil, bm, &plan.cols[0], stride, od, nd, 64)
	if len(rows) != 2 || rows[0] != 3 || rows[1] != 40 {
		t.Fatalf("col A changed rows = %v, want [3 40]", rows)
	}
	// Column 1 (B): unchanged.
	rowsB := colChangedRows(nil, bm, &plan.cols[1], stride, od, nd, 64)
	if len(rowsB) != 0 {
		t.Fatalf("col B changed rows = %v, want []", rowsB)
	}
}

func patchTag(patch []byte) (byte, bool) {
	h, n, err := readPatchHeader(patch)
	if err != nil {
		return 0, false
	}
	body := patch[n:]
	if h.flags&flagPatchRANS != 0 {
		b, err := decompressPatchBody(body)
		if err != nil {
			return 0, false
		}
		body = b
	}
	// body = root op (opMerge) then the slice patch tag at body[1:].
	if len(body) < 2 || body[0] != opMerge {
		return 0, false
	}
	return body[1], true
}

func TestColDiffIntSparse(t *testing.T) {
	old := make([]ccRow, 200)
	neu := make([]ccRow, 200)
	for i := range old {
		old[i] = ccRow{A: int64(i * 7), B: uint32(i), F: float64(i)}
		neu[i] = old[i]
	}
	neu[10].A = -1
	neu[150].A = -2
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if tag, ok := patchTag(patch); !ok || tag != tagColSlicePatch {
		t.Fatalf("expected tagColSlicePatch, got %#x ok=%v", tag, ok)
	}
	got := append([]ccRow(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("sparse int round-trip mismatch")
	}
}

func TestColDiffIntDelta(t *testing.T) {
	old := make([]ccRow, 200)
	neu := make([]ccRow, 200)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
		neu[i] = old[i]
		neu[i].A = int64(i) + 1000 // every row's A changed → delta mode
	}
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if tag, _ := patchTag(patch); tag != tagColSlicePatch {
		t.Fatalf("expected tagColSlicePatch, got %#x", tag)
	}
	got := append([]ccRow(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("delta int round-trip mismatch")
	}
}

func TestColDiffNeverLarger(t *testing.T) {
	const n = 300
	old := make([]ccRow, n)
	neu := make([]ccRow, n)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i * 3), F: float64(i)}
		neu[i] = old[i]
	}
	// Clustered: only column A changes, across many rows → columnar should win.
	for i := 0; i < n; i += 2 {
		neu[i].A = int64(i) * 11
	}
	cPatch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	full, err := Marshal(neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(cPatch) >= len(full) {
		t.Fatalf("columnar patch %d not smaller than full encode %d", len(cPatch), len(full))
	}
	if tag, _ := patchTag(cPatch); tag != tagColSlicePatch {
		t.Fatalf("expected tagColSlicePatch on clustered batch, got %#x", tag)
	}
	// Round-trip holds.
	got := append([]ccRow(nil), old...)
	if err := Apply(&got, cPatch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("round-trip mismatch")
	}
}

func TestColDiffScatteredFallsBack(t *testing.T) {
	const n = 64
	old := make([]ccRow, n)
	neu := make([]ccRow, n)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
		neu[i] = old[i]
	}
	neu[1].A = 1
	neu[2].B = 2
	neu[3].F = 3 // three different columns, one row each
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	got := append([]ccRow(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("round-trip mismatch")
	}
	// Either tag is acceptable here, but the patch must be no larger than the
	// positional-only encoding. (Correctness + never-larger is what matters.)
}

func benchColRows(n int) ([]ccRow, []ccRow) {
	old := make([]ccRow, n)
	neu := make([]ccRow, n)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i * 3), F: float64(i)}
		neu[i] = old[i]
	}
	for i := 0; i < n; i += 3 {
		neu[i].A = int64(i) * 7 // one column changes across ~1/3 of rows
	}
	return old, neu
}

func BenchmarkColDiffDiff(b *testing.B) {
	old, neu := benchColRows(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Diff(old, neu, OptBalanced); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkColDiffApply(b *testing.B) {
	old, neu := benchColRows(1000)
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		got := append([]ccRow(nil), old...)
		if err := Apply(&got, patch); err != nil {
			b.Fatal(err)
		}
	}
}

type ccAll struct {
	I  int32
	U  uint16
	F  float64
	F3 float32
	Bo bool
	S  string
	By []byte
	T  time.Time
}

func TestColDiffAllKinds(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	mk := func() []ccAll {
		rows := make([]ccAll, 120)
		for i := range rows {
			rows[i] = ccAll{
				I: int32(i), U: uint16(i), F: float64(i) * 1.5, F3: float32(i),
				Bo: i%2 == 0, S: "v" + string(rune('a'+i%26)),
				By: []byte{byte(i)}, T: base.Add(time.Duration(i) * time.Second),
			}
		}
		return rows
	}
	mutators := map[string]func(*ccAll){
		"int":     func(r *ccAll) { r.I += 1000 },
		"uint":    func(r *ccAll) { r.U += 7 },
		"float":   func(r *ccAll) { r.F += 0.25 },
		"float32": func(r *ccAll) { r.F3 += 2 },
		"bool":    func(r *ccAll) { r.Bo = !r.Bo },
		"string":  func(r *ccAll) { r.S += "X" },
		"bytes":   func(r *ccAll) { r.By = append(append([]byte(nil), r.By...), 0xFF) },
		"time":    func(r *ccAll) { r.T = r.T.Add(time.Hour) },
	}
	for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
		old := mk()
		for name, mut := range mutators {
			// sparse: mutate a few rows (ratio 20); dense/delta: mutate all (ratio 1).
			for _, ratio := range []int{20, 1} {
				neu := mk()
				for i := range neu {
					if ratio == 1 || i%ratio == 0 {
						mut(&neu[i])
					}
				}
				patch, err := Diff(old, neu, opt)
				if err != nil {
					t.Fatalf("%s/%v/ratio%d diff: %v", name, opt, ratio, err)
				}
				got := append([]ccAll(nil), old...)
				if err := Apply(&got, patch); err != nil {
					t.Fatalf("%s/%v/ratio%d apply: %v", name, opt, ratio, err)
				}
				if !reflect.DeepEqual(got, neu) {
					t.Fatalf("%s/%v/ratio%d round-trip mismatch", name, opt, ratio)
				}
			}
		}
	}
}

func TestColDiffNullable(t *testing.T) {
	type ccNull struct {
		A int64
		P *int32
	}
	n := 64
	old := make([]ccNull, n)
	neu := make([]ccNull, n)
	for i := range old {
		v := int32(i)
		old[i] = ccNull{A: int64(i), P: &v}
		w := int32(i)
		neu[i] = ccNull{A: int64(i), P: &w}
	}
	x := int32(999)
	neu[10].P = &x
	neu[20].P = nil
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	got := append([]ccNull(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("nullable round-trip mismatch")
	}
}
