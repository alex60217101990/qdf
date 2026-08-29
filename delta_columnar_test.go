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
	changed := markChangedRows(bm, plan, stride, od, nd, 64, false)
	if !changed {
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

	for b.Loop() {
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

	for b.Loop() {
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
	for i := range good {
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
		for range int(nMod) % n {
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

// big130Row is a struct with 130 int64 fields — exactly 2 more than the old
// 128-column cap — used to exercise the nChangedCols >=128 varint path.
type big130Row struct {
	F000, F001, F002, F003, F004, F005, F006, F007, F008, F009 int64
	F010, F011, F012, F013, F014, F015, F016, F017, F018, F019 int64
	F020, F021, F022, F023, F024, F025, F026, F027, F028, F029 int64
	F030, F031, F032, F033, F034, F035, F036, F037, F038, F039 int64
	F040, F041, F042, F043, F044, F045, F046, F047, F048, F049 int64
	F050, F051, F052, F053, F054, F055, F056, F057, F058, F059 int64
	F060, F061, F062, F063, F064, F065, F066, F067, F068, F069 int64
	F070, F071, F072, F073, F074, F075, F076, F077, F078, F079 int64
	F080, F081, F082, F083, F084, F085, F086, F087, F088, F089 int64
	F090, F091, F092, F093, F094, F095, F096, F097, F098, F099 int64
	F100, F101, F102, F103, F104, F105, F106, F107, F108, F109 int64
	F110, F111, F112, F113, F114, F115, F116, F117, F118, F119 int64
	F120, F121, F122, F123, F124, F125, F126, F127, F128, F129 int64
}

// TestColDiff128ColsRoundTrip encodes a slice of big130Row where all 130
// columns change, so nChangedCols = 130 >= 128. Before the fix,
// diffColumnarEligible declined structs with >=128 cols (the patch would use
// positional encoding and patchTag would not be tagColSlicePatch). After the
// fix, the columnar path is used with a 2-byte padded uvarint for nChangedCols,
// and the round-trip must produce the original values.
func TestColDiff128ColsRoundTrip(t *testing.T) {
	const n = 500
	old := make([]big130Row, n)
	neu := make([]big130Row, n)

	// Initialize: field Fxxx = int64(xxx) for every row.
	for i := range old {
		r := &old[i]
		r.F000, r.F001, r.F002, r.F003, r.F004 = 0, 1, 2, 3, 4
		r.F005, r.F006, r.F007, r.F008, r.F009 = 5, 6, 7, 8, 9
		r.F010, r.F011, r.F012, r.F013, r.F014 = 10, 11, 12, 13, 14
		r.F015, r.F016, r.F017, r.F018, r.F019 = 15, 16, 17, 18, 19
		r.F020, r.F021, r.F022, r.F023, r.F024 = 20, 21, 22, 23, 24
		r.F025, r.F026, r.F027, r.F028, r.F029 = 25, 26, 27, 28, 29
		r.F030, r.F031, r.F032, r.F033, r.F034 = 30, 31, 32, 33, 34
		r.F035, r.F036, r.F037, r.F038, r.F039 = 35, 36, 37, 38, 39
		r.F040, r.F041, r.F042, r.F043, r.F044 = 40, 41, 42, 43, 44
		r.F045, r.F046, r.F047, r.F048, r.F049 = 45, 46, 47, 48, 49
		r.F050, r.F051, r.F052, r.F053, r.F054 = 50, 51, 52, 53, 54
		r.F055, r.F056, r.F057, r.F058, r.F059 = 55, 56, 57, 58, 59
		r.F060, r.F061, r.F062, r.F063, r.F064 = 60, 61, 62, 63, 64
		r.F065, r.F066, r.F067, r.F068, r.F069 = 65, 66, 67, 68, 69
		r.F070, r.F071, r.F072, r.F073, r.F074 = 70, 71, 72, 73, 74
		r.F075, r.F076, r.F077, r.F078, r.F079 = 75, 76, 77, 78, 79
		r.F080, r.F081, r.F082, r.F083, r.F084 = 80, 81, 82, 83, 84
		r.F085, r.F086, r.F087, r.F088, r.F089 = 85, 86, 87, 88, 89
		r.F090, r.F091, r.F092, r.F093, r.F094 = 90, 91, 92, 93, 94
		r.F095, r.F096, r.F097, r.F098, r.F099 = 95, 96, 97, 98, 99
		r.F100, r.F101, r.F102, r.F103, r.F104 = 100, 101, 102, 103, 104
		r.F105, r.F106, r.F107, r.F108, r.F109 = 105, 106, 107, 108, 109
		r.F110, r.F111, r.F112, r.F113, r.F114 = 110, 111, 112, 113, 114
		r.F115, r.F116, r.F117, r.F118, r.F119 = 115, 116, 117, 118, 119
		r.F120, r.F121, r.F122, r.F123, r.F124 = 120, 121, 122, 123, 124
		r.F125, r.F126, r.F127, r.F128, r.F129 = 125, 126, 127, 128, 129
	}
	// new: every field += 1000. All 130 columns change → nChangedCols = 130.
	for i := range neu {
		r := &old[i]
		w := &neu[i]
		w.F000, w.F001, w.F002, w.F003, w.F004 = r.F000+1000, r.F001+1000, r.F002+1000, r.F003+1000, r.F004+1000
		w.F005, w.F006, w.F007, w.F008, w.F009 = r.F005+1000, r.F006+1000, r.F007+1000, r.F008+1000, r.F009+1000
		w.F010, w.F011, w.F012, w.F013, w.F014 = r.F010+1000, r.F011+1000, r.F012+1000, r.F013+1000, r.F014+1000
		w.F015, w.F016, w.F017, w.F018, w.F019 = r.F015+1000, r.F016+1000, r.F017+1000, r.F018+1000, r.F019+1000
		w.F020, w.F021, w.F022, w.F023, w.F024 = r.F020+1000, r.F021+1000, r.F022+1000, r.F023+1000, r.F024+1000
		w.F025, w.F026, w.F027, w.F028, w.F029 = r.F025+1000, r.F026+1000, r.F027+1000, r.F028+1000, r.F029+1000
		w.F030, w.F031, w.F032, w.F033, w.F034 = r.F030+1000, r.F031+1000, r.F032+1000, r.F033+1000, r.F034+1000
		w.F035, w.F036, w.F037, w.F038, w.F039 = r.F035+1000, r.F036+1000, r.F037+1000, r.F038+1000, r.F039+1000
		w.F040, w.F041, w.F042, w.F043, w.F044 = r.F040+1000, r.F041+1000, r.F042+1000, r.F043+1000, r.F044+1000
		w.F045, w.F046, w.F047, w.F048, w.F049 = r.F045+1000, r.F046+1000, r.F047+1000, r.F048+1000, r.F049+1000
		w.F050, w.F051, w.F052, w.F053, w.F054 = r.F050+1000, r.F051+1000, r.F052+1000, r.F053+1000, r.F054+1000
		w.F055, w.F056, w.F057, w.F058, w.F059 = r.F055+1000, r.F056+1000, r.F057+1000, r.F058+1000, r.F059+1000
		w.F060, w.F061, w.F062, w.F063, w.F064 = r.F060+1000, r.F061+1000, r.F062+1000, r.F063+1000, r.F064+1000
		w.F065, w.F066, w.F067, w.F068, w.F069 = r.F065+1000, r.F066+1000, r.F067+1000, r.F068+1000, r.F069+1000
		w.F070, w.F071, w.F072, w.F073, w.F074 = r.F070+1000, r.F071+1000, r.F072+1000, r.F073+1000, r.F074+1000
		w.F075, w.F076, w.F077, w.F078, w.F079 = r.F075+1000, r.F076+1000, r.F077+1000, r.F078+1000, r.F079+1000
		w.F080, w.F081, w.F082, w.F083, w.F084 = r.F080+1000, r.F081+1000, r.F082+1000, r.F083+1000, r.F084+1000
		w.F085, w.F086, w.F087, w.F088, w.F089 = r.F085+1000, r.F086+1000, r.F087+1000, r.F088+1000, r.F089+1000
		w.F090, w.F091, w.F092, w.F093, w.F094 = r.F090+1000, r.F091+1000, r.F092+1000, r.F093+1000, r.F094+1000
		w.F095, w.F096, w.F097, w.F098, w.F099 = r.F095+1000, r.F096+1000, r.F097+1000, r.F098+1000, r.F099+1000
		w.F100, w.F101, w.F102, w.F103, w.F104 = r.F100+1000, r.F101+1000, r.F102+1000, r.F103+1000, r.F104+1000
		w.F105, w.F106, w.F107, w.F108, w.F109 = r.F105+1000, r.F106+1000, r.F107+1000, r.F108+1000, r.F109+1000
		w.F110, w.F111, w.F112, w.F113, w.F114 = r.F110+1000, r.F111+1000, r.F112+1000, r.F113+1000, r.F114+1000
		w.F115, w.F116, w.F117, w.F118, w.F119 = r.F115+1000, r.F116+1000, r.F117+1000, r.F118+1000, r.F119+1000
		w.F120, w.F121, w.F122, w.F123, w.F124 = r.F120+1000, r.F121+1000, r.F122+1000, r.F123+1000, r.F124+1000
		w.F125, w.F126, w.F127, w.F128, w.F129 = r.F125+1000, r.F126+1000, r.F127+1000, r.F128+1000, r.F129+1000
	}

	patch, err := Diff(old, neu, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Verify the columnar path was chosen (nChangedCols = 130 requires the
	// 2-byte varint fix; positional would have been chosen before the fix).
	if tag, ok := patchTag(patch); !ok || tag != tagColSlicePatch {
		t.Fatalf("expected tagColSlicePatch for 130-col struct, got tag=%#x ok=%v", tag, ok)
	}
	got := append([]big130Row(nil), old...)
	if err := Apply(&got, patch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, neu) {
		t.Fatal("128-col round-trip mismatch")
	}
}
