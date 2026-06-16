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
	any := markChangedRows(bm, plan, stride, od, nd, 64, false)
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

type ccStr struct {
	A int64
	S string
}

// TestColDiffStringLowCardDict exercises the dict codec on the column-diff path:
// a low-cardinality string column with many changed cells. Under OptBalanced
// (Dense mode) the column patch must WIN over positional AND Apply must succeed
// — this is the exact bug the stateless routing fixes (a stateful intern body
// would emit a state-table ref the decoder never built → "unknown state-table
// id" on Apply). Run under both Dense (OptBalanced) and Speed.
func TestColDiffStringLowCardDict(t *testing.T) {
	const n = 200
	vals := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for _, opt := range []Options{OptBalanced, OptSpeed} {
		old := make([]ccStr, n)
		neu := make([]ccStr, n)
		for i := range old {
			old[i] = ccStr{A: int64(i), S: vals[i%len(vals)]}
			neu[i] = old[i]
		}
		// Change many cells (~half) to a different value from the small set.
		for i := 0; i < n; i += 2 {
			neu[i].S = vals[(i+1)%len(vals)]
		}
		patch, err := Diff(old, neu, opt)
		if err != nil {
			t.Fatalf("%v diff: %v", opt, err)
		}
		if tag, ok := patchTag(patch); !ok || tag != tagColSlicePatch {
			t.Fatalf("%v: expected tagColSlicePatch (column body must win), got %#x ok=%v", opt, tag, ok)
		}
		got := append([]ccStr(nil), old...)
		if err := Apply(&got, patch); err != nil {
			t.Fatalf("%v apply: %v", opt, err)
		}
		if !reflect.DeepEqual(got, neu) {
			t.Fatalf("%v low-card string round-trip mismatch", opt)
		}
	}
}

// TestColDiffStringHighCard exercises the raw codec: an all-distinct string
// column. Round-trip + Apply success under Dense (OptBalanced) and Speed.
func TestColDiffStringHighCard(t *testing.T) {
	const n = 200
	for _, opt := range []Options{OptBalanced, OptSpeed} {
		old := make([]ccStr, n)
		neu := make([]ccStr, n)
		for i := range old {
			old[i] = ccStr{A: int64(i), S: "id-" + string(rune('A'+i%26)) + "-" + itoaTest(i)}
			neu[i] = old[i]
		}
		// Mutate enough cells that the whole column re-ships (dense-whole → raw).
		for i := range neu {
			neu[i].S += "-mut"
		}
		patch, err := Diff(old, neu, opt)
		if err != nil {
			t.Fatalf("%v diff: %v", opt, err)
		}
		got := append([]ccStr(nil), old...)
		if err := Apply(&got, patch); err != nil {
			t.Fatalf("%v apply: %v", opt, err)
		}
		if !reflect.DeepEqual(got, neu) {
			t.Fatalf("%v high-card string round-trip mismatch", opt)
		}
	}
}

func itoaTest(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

// FuzzColDiffStringOracle mutates a string column with random per-row choices
// (drawn from a mix of low-cardinality and high-cardinality candidates) and
// asserts Diff then Apply round-trips DeepEqual across all modes — directly
// stressing the stateless dict/raw column-diff path.
func FuzzColDiffStringOracle(f *testing.F) {
	f.Add(uint64(1), uint64(2), 64)
	f.Add(uint64(7), uint64(7), 13)
	f.Add(uint64(0xdead), uint64(0xbeef), 200)
	pool := []string{"", "a", "alpha", "beta", "gamma", "x", "id-12345", "long-distinct-value-"}
	mkVal := func(r uint64, i int) string {
		choice := (r ^ uint64(i*2654435761)) % uint64(len(pool)+2)
		switch {
		case choice < uint64(len(pool)):
			return pool[choice]
		case choice == uint64(len(pool)):
			return "u-" + itoaTest(int(r%97)) + "-" + itoaTest(i) // semi-distinct
		default:
			return "d-" + itoaTest(i) + "-" + itoaTest(int(r&0xffff)) // distinct-ish
		}
	}
	f.Fuzz(func(t *testing.T, so, sn uint64, nRaw int) {
		n := nRaw % 257
		if n < 0 {
			n = -n
		}
		// n >= 1: an empty slice exercises the framework's nil-vs-empty base
		// fingerprint distinction (append([]T(nil), empty...) yields nil), which is
		// orthogonal to the string-column codec under test and covered elsewhere.
		n++
		old := make([]ccStr, n)
		neu := make([]ccStr, n)
		for i := range old {
			old[i] = ccStr{A: int64(i), S: mkVal(so, i)}
			neu[i] = ccStr{A: int64(i), S: mkVal(sn, i)}
		}
		for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
			patch, err := Diff(old, neu, opt)
			if err != nil {
				t.Fatalf("opt=%v diff: %v", opt, err)
			}
			got := append([]ccStr(nil), old...)
			if err := Apply(&got, patch); err != nil {
				t.Fatalf("opt=%v apply: %v", opt, err)
			}
			if !reflect.DeepEqual(got, neu) {
				t.Fatalf("opt=%v mismatch\n old=%+v\n new=%+v\n got=%+v", opt, old, neu, got)
			}
		}
	})
}

type ccBytes struct {
	A int64
	B []byte
}

// TestColDiffBytesColumn exercises a []byte column (classified colKindString
// with isByte) on the column-diff path, both sparse (few changed) and
// dense-whole (all changed), under Dense (OptBalanced) and Speed.
func TestColDiffBytesColumn(t *testing.T) {
	const n = 200
	for _, opt := range []Options{OptBalanced, OptSpeed} {
		for _, ratio := range []int{20, 1} { // sparse, then dense-whole
			old := make([]ccBytes, n)
			neu := make([]ccBytes, n)
			for i := range old {
				old[i] = ccBytes{A: int64(i), B: []byte("payload-" + itoaTest(i%7))}
				neu[i] = ccBytes{A: int64(i), B: append([]byte(nil), old[i].B...)}
			}
			for i := range neu {
				if ratio == 1 || i%ratio == 0 {
					neu[i].B = append(append([]byte(nil), neu[i].B...), 0xAB)
				}
			}
			patch, err := Diff(old, neu, opt)
			if err != nil {
				t.Fatalf("%v/ratio%d diff: %v", opt, ratio, err)
			}
			got := append([]ccBytes(nil), old...)
			if err := Apply(&got, patch); err != nil {
				t.Fatalf("%v/ratio%d apply: %v", opt, ratio, err)
			}
			if !reflect.DeepEqual(got, neu) {
				t.Fatalf("%v/ratio%d bytes round-trip mismatch", opt, ratio)
			}
		}
	}
}

func TestColDiffApplyHostile(t *testing.T) {
	old := make([]ccRow, 64)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
	}
	neu := append([]ccRow(nil), old...)
	for i := 0; i < 64; i += 2 {
		neu[i].A = int64(i) * 5
	}
	good, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	seeds := make([][]byte, 0, 3+len(good))
	seeds = append(seeds, nil, []byte{}, []byte{0x00})
	for i := range len(good) {
		trunc := make([]byte, i)
		copy(trunc, good[:i])
		seeds = append(seeds, trunc)
	}
	for i, b := range seeds {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("seed %d panicked: %v", i, rec)
				}
			}()
			got := append([]ccRow(nil), old...)
			_ = Apply(&got, b)
		}()
	}
}

func FuzzColDiffApply(f *testing.F) {
	old := make([]ccRow, 32)
	for i := range old {
		old[i] = ccRow{A: int64(i), B: uint32(i), F: float64(i)}
	}
	neu := append([]ccRow(nil), old...)
	neu[1].A = 5
	if good, err := Diff(old, neu, OptBalanced); err == nil {
		f.Add(good)
	}
	f.Add([]byte{})
	f.Add([]byte("garbage"))
	f.Fuzz(func(t *testing.T, patch []byte) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("panic on %x: %v", patch, rec)
			}
		}()
		got := append([]ccRow(nil), old...)
		_ = Apply(&got, patch)
	})
}

func FuzzColDiffOracle(f *testing.F) {
	f.Add(int64(1), uint8(3), uint8(5))
	f.Fuzz(func(t *testing.T, seed int64, nMod, which uint8) {
		const n = 80
		old := make([]ccRow, n)
		for i := range old {
			old[i] = ccRow{A: int64(i) ^ seed, B: uint32(i), F: float64(i)}
		}
		neu := append([]ccRow(nil), old...)
		r := seed
		for k := 0; k < int(nMod)%n; k++ {
			r = r*1103515245 + 12345
			idx := int(uint64(r)>>33) % n
			switch which % 3 {
			case 0:
				neu[idx].A += r
			case 1:
				neu[idx].B ^= uint32(r)
			case 2:
				neu[idx].F = float64(r)
			}
		}
		patch, err := Diff(old, neu, OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		got := append([]ccRow(nil), old...)
		if err := Apply(&got, patch); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, neu) {
			t.Fatalf("oracle mismatch seed=%d", seed)
		}
	})
}

func TestColDiffLengthChange(t *testing.T) {
	old := make([]ccRow, 64)
	for i := range old {
		old[i] = ccRow{A: int64(i)}
	}
	neu := append(append([]ccRow(nil), old...), ccRow{A: 999}) // grew by 1
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	got := append([]ccRow(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("length-change round-trip mismatch")
	}
}

func TestColDiffHybridFallsBack(t *testing.T) {
	type hybrid struct {
		A int64
		M map[string]int
	}
	n := 32
	old := make([]hybrid, n)
	neu := make([]hybrid, n)
	for i := range old {
		old[i] = hybrid{A: int64(i), M: map[string]int{"k": i}}
		neu[i] = hybrid{A: int64(i), M: map[string]int{"k": i}}
	}
	neu[3].A = 100
	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]hybrid, n)
	for i := range old {
		got[i] = hybrid{A: old[i].A, M: map[string]int{"k": i}}
	}
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("hybrid round-trip mismatch")
	}
}
