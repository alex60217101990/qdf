package qdf

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"
)

// Phase 1: buildColumnarPlan three-way classification.
//   - every field eligible  → pure columnar (residual == nil)
//   - some eligible, some not → hybrid (residual != nil, full ordered shape)
//   - no eligible field       → nil (full row-major)
func TestBuildColumnarPlanHybrid(t *testing.T) {
	type allEligible struct {
		A int64
		B string
		C bool
	}
	type mixed struct {
		ID    string            // eligible
		N     int64             // eligible
		Tags  []string          // residual (non-[]byte slice)
		Attrs map[string]string // residual (map)
	}
	type allResidual struct {
		M map[string]int
		S []string
	}

	plan := func(v any) *columnarPlan {
		td, err := descOf(reflect.TypeOf(v))
		if err != nil {
			t.Fatal(err)
		}
		return buildColumnarPlan(td)
	}

	// All eligible → pure columnar, no residual.
	if p := plan(allEligible{}); p == nil || len(p.cols) != 3 || p.residual != nil {
		t.Fatalf("allEligible: want pure-columnar 3 cols + nil residual, got %#v", p)
	}

	// Mixed → hybrid.
	p := plan(mixed{})
	if p == nil {
		t.Fatal("mixed: got nil plan, want hybrid")
	}
	if len(p.cols) != 2 {
		t.Fatalf("mixed: want 2 eligible cols (ID,N), got %d", len(p.cols))
	}
	if len(p.residual) != 2 {
		t.Fatalf("mixed: want 2 residual fields (Tags,Attrs), got %d", len(p.residual))
	}
	if len(p.hybridNames) != 4 || len(p.hybridKinds) != 4 {
		t.Fatalf("mixed: hybrid shape must list ALL 4 fields, got names=%d kinds=%d",
			len(p.hybridNames), len(p.hybridKinds))
	}
	// Field declaration order preserved; residual fields marked with residualKind.
	wantResidual := map[string]bool{"Tags": true, "Attrs": true}
	for i, name := range p.hybridNames {
		gotResidual := p.hybridKinds[i] == residualKind
		if gotResidual != wantResidual[name] {
			t.Fatalf("mixed: field %q residual=%v, want %v", name, gotResidual, wantResidual[name])
		}
	}
	// Residual descriptors carry the field's own codec (non-nil desc).
	for _, rf := range p.residual {
		if rf.desc == nil {
			t.Fatalf("mixed: residual field %q has nil desc", rf.name)
		}
	}

	// No eligible field → nil plan (nothing to transpose).
	if p := plan(allResidual{}); p != nil {
		t.Fatalf("allResidual: want nil plan, got %#v", p)
	}
}

// Phase 2: the hybrid shape table is a separate ID space from the pure-columnar
// table — a stream interleaving tagColStruct and tagHybridColStruct payloads
// must not alias shape IDs.
func TestHybridShapeIDIndependence(t *testing.T) {
	names := []string{"a", "b", "c"}
	colKinds := []colKind{colKindInt, colKindString, colKindBool}
	hybKinds := []colKind{colKindInt, residualKind, colKindBool}

	e := newEncState()
	// Declaring in each table starts independently at ID 1.
	if id := e.colShapeDeclare(names, colKinds); id != 1 {
		t.Fatalf("colShapeDeclare first id = %d, want 1", id)
	}
	if id := e.hybridShapeDeclare(names, hybKinds); id != 1 {
		t.Fatalf("hybridShapeDeclare first id = %d, want 1", id)
	}
	// Lookups stay within their own table.
	if got := e.hybridShapeFor(names, hybKinds); got != 1 {
		t.Fatalf("hybridShapeFor reuse = %d, want 1", got)
	}
	// A hybrid kinds set must NOT match a columnar shape with the same names
	// (different kinds — residualKind sentinel differs).
	if got := e.colShapeFor(names, hybKinds); got != 0 {
		t.Fatalf("colShapeFor must not match hybrid kinds, got %d", got)
	}
	if got := e.hybridShapeFor(names, colKinds); got != 0 {
		t.Fatalf("hybridShapeFor must not match columnar kinds, got %d", got)
	}

	d := newDecState()
	dc := d.colShapeDeclareDec(names, colKinds)
	dh := d.hybridShapeDeclareDec(names, hybKinds)
	if dc == nil || dh == nil {
		t.Fatal("decoder declare returned nil")
	}
	// Same wire ID (1) resolves to the correct, independent table entry.
	if got := d.colShapeLookup(1); got == nil || got.kinds[1] != colKindString {
		t.Fatalf("colShapeLookup(1) wrong: %+v", got)
	}
	if got := d.hybridShapeLookup(1); got == nil || got.kinds[1] != residualKind {
		t.Fatalf("hybridShapeLookup(1) wrong (want residualKind at [1]): %+v", got)
	}
}

type hybridRec struct {
	Level  string            // eligible, low-card → dict
	Seq    int64             // eligible, monotonic → Delta+FOR
	Active bool              // eligible → bitpack
	Region string            // eligible, low-card → dict
	Tags   []string          // residual (slice)
	Attrs  map[string]string // residual (map)
}

func mkHybridRecs(n int) []hybridRec {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	regions := []string{"us-east", "eu-west", "ap-south"}
	out := make([]hybridRec, n)
	for i := range out {
		out[i] = hybridRec{
			Level:  levels[i%len(levels)],
			Seq:    int64(i),
			Active: i%2 == 0,
			Region: regions[i%len(regions)],
			Tags:   []string{"a", fmt.Sprintf("t%d", i%7)},
			Attrs:  map[string]string{"k1": fmt.Sprintf("v%d", i%5), "k2": "const"},
		}
	}
	return out
}

// Phase 3+4: hybrid encode+decode round-trips bit-exactly across every tier,
// and the hybrid tag fires on the Balanced tier for a compressible mixed struct.
func TestHybridRoundTrip(t *testing.T) {
	in := mkHybridRecs(200)
	opts := []Options{
		OptSpeed,
		OptBalanced,
		OptBalanced | OptMapShape,
		OptBalanced | OptColumnIndex,
		OptBalanced | OptFSST,
		OptCompression,
	}
	for _, opt := range opts {
		b, err := Marshal(in, opt)
		if err != nil {
			t.Fatalf("opt=%d encode: %v", opt, err)
		}
		var out []hybridRec
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("opt=%d decode: %v", opt, err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("opt=%d round-trip mismatch", opt)
		}
	}

	// Hybrid auto-fires when FSST is enabled. OptBalanced|OptFSST has no rANS to
	// wrap/hide the tag, so the tag is directly observable — proving the eligible
	// columns are transposed rather than the whole struct falling to row-major.
	b, _ := Marshal(in, OptBalanced|OptFSST)
	if !containsByte(b, tagHybridColStruct) {
		t.Fatalf("OptBalanced|OptFSST: mixed struct must use hybrid, got %x...", b[:min(48, len(b))])
	}
	// Without FSST, hybrid must NOT fire (no Balanced regression guarantee).
	bb, _ := Marshal(in, OptBalanced)
	if containsByte(bb, tagHybridColStruct) {
		t.Fatal("OptBalanced (no FSST): hybrid must not auto-fire in v1")
	}
}

// Randomized round-trip stress: varied cardinality and edge eligible values
// (negative/zero/large ints, empty/unicode strings, both bools) through the
// hybrid (FSST) and row-major (Balanced) paths. Residual collections stay
// non-empty to avoid the orthogonal nil-vs-empty slice semantics.
func TestHybridRandomizedRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(0x68796272, 0x69647a7a)) // "hybr","idzz"
	strs := []string{"", "x", "INFO", "héllo-Ω", "a-fairly-long-token-value", "INFO"}
	for iter := range 40 {
		n := 16 + r.IntN(300)
		card := 1 + r.IntN(8) // eligible string cardinality this batch
		in := make([]hybridRec, n)
		for i := range in {
			in[i] = hybridRec{
				Level:  strs[r.IntN(min(card, len(strs)))],
				Seq:    int64(r.Uint64()) - (1 << 62), // negative, zero, large
				Active: r.IntN(2) == 0,
				Region: strs[r.IntN(min(card, len(strs)))],
				Tags:   []string{strs[r.IntN(len(strs))]},
				Attrs:  map[string]string{"k": strs[r.IntN(len(strs))]},
			}
		}
		for _, opt := range []Options{OptBalanced, OptBalanced | OptFSST, OptCompression} {
			b, err := Marshal(in, opt)
			if err != nil {
				t.Fatalf("iter %d opt=%d encode: %v", iter, opt, err)
			}
			var out []hybridRec
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("iter %d opt=%d decode: %v", iter, opt, err)
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("iter %d opt=%d round-trip mismatch (n=%d card=%d)", iter, opt, n, card)
			}
		}
	}
}

// Schema-evolution skip + dynamic/any decode of a hybrid payload. These hit
// Skip(), decodeAny() and the elemDynamic ([]map[string]any) paths, which must
// all recognize tagHybridColStruct (parallel to tagColStruct).
func TestHybridSkipAndAny(t *testing.T) {
	in := mkHybridRecs(50)

	// (1) Skip: a parent struct carrying a hybrid-slice field, decoded into a
	// struct that lacks it → the unknown field must be skipped cleanly.
	type withField struct {
		Tag     string
		Recs    []hybridRec
		Trailer int64
	}
	type without struct {
		Tag     string
		Trailer int64
	}
	b, err := Marshal(withField{Tag: "x", Recs: in, Trailer: 99}, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var w without
	if err := Unmarshal(b, &w); err != nil {
		t.Fatalf("skip hybrid field: %v", err)
	}
	if w.Tag != "x" || w.Trailer != 99 {
		t.Fatalf("skip desynced state: %+v", w)
	}

	// (2) any field holding a hybrid slice.
	type anyHolder struct{ V any }
	ab, err := Marshal(anyHolder{V: in}, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var ah anyHolder
	if err := Unmarshal(ab, &ah); err != nil {
		t.Fatalf("decodeAny hybrid: %v", err)
	}
	rows, ok := ah.V.([]any)
	if !ok || len(rows) != len(in) {
		t.Fatalf("any hybrid: got %T len=%d", ah.V, len(rows))
	}

	// (3) elemDynamic: decode straight into []map[string]any.
	db, err := Marshal(in, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var dyn []map[string]any
	if err := Unmarshal(db, &dyn); err != nil {
		t.Fatalf("dynamic hybrid: %v", err)
	}
	if len(dyn) != len(in) {
		t.Fatalf("dynamic hybrid len=%d want %d", len(dyn), len(in))
	}
	// Spot-check an eligible column + a residual field survived.
	if dyn[3]["Level"] != in[3].Level {
		t.Fatalf("dynamic eligible mismatch: %v vs %v", dyn[3]["Level"], in[3].Level)
	}
}

// A columnar bool column must bound its claimed element count by the row count
// (colMaxLen), like every other columnar codec — not just by the body bytes
// (which admit up to 8× the buffer). n=128 with colMaxLen=16 passes the
// body-byte bound (128 <= 16*8) but exceeds the rows and must be rejected
// before allocating.
func TestColumnarBoolColLenGuard(t *testing.T) {
	buf := []byte{tagPackBool}
	buf = appendUvarint(buf, 128)
	buf = append(buf, make([]byte, 16)...) // 128 bits of body
	d := NewDecoderOnBuf(buf)
	d.colMaxLen = 16
	var out []bool
	if err := decodeSliceBoolInto(d, &out); err == nil {
		t.Fatalf("bool column n=128 with colMaxLen=16 must be rejected, got nil (len=%d)", len(out))
	}
}

// A small slice (< columnarMinElems) must fall back to row-major.
func TestHybridSmallSliceFallback(t *testing.T) {
	in := mkHybridRecs(8)
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if containsByte(b, tagHybridColStruct) {
		t.Fatal("below columnarMinElems must not use hybrid")
	}
	var out []hybridRec
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("small-slice fallback round-trip mismatch")
	}
}

// Query/Select over a hybrid payload returns ErrUnsupported in v1.
func TestHybridQueryUnsupported(t *testing.T) {
	in := mkHybridRecs(200)
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []hybridRec
	err = Unmarshal(b, &out, Where("Seq", func(v int64) bool { return v > 100 }))
	if err == nil {
		t.Fatal("query over hybrid payload must error in v1")
	}
}
