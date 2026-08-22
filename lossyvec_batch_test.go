package qdf

import (
	"math"
	"testing"
)

type vecOnlyRow struct {
	Emb []float32
}

type vecMixedRow struct {
	ID   string
	Tags []string
	Emb  []float64
	N    int
}

func sinVec32(seed, dim int) []float32 {
	v := make([]float32, dim)
	for j := range v {
		v[j] = float32(math.Sin(float64(seed*dim+j) * 0.013))
	}
	return v
}

func sinVec64(seed, dim int) []float64 {
	v := make([]float64, dim)
	for j := range v {
		v[j] = math.Sin(float64(seed*dim+j) * 0.013)
	}
	return v
}

func cosine32(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	return dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30)
}

func cosine64(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	return dot / (math.Sqrt(na)*math.Sqrt(nb) + 1e-30)
}

func TestVecBatchRoundTripF32(t *testing.T) {
	const n, dim = 64, 256
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, dim)}
	}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MinCosine(0.999))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("encode: %v", err)
	}
	data := enc.Bytes()
	if data[5] != tagVecBatchStruct {
		t.Fatalf("expected tagVecBatchStruct (0xFE) at byte 5, got 0x%02x", data[5])
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != n {
		t.Fatalf("len %d want %d", len(out), n)
	}
	for i := range rows {
		if len(out[i].Emb) != dim {
			t.Fatalf("row %d dim %d", i, len(out[i].Emb))
		}
		if c := cosine32(rows[i].Emb, out[i].Emb); c < 0.999*0.999 {
			t.Fatalf("row %d cosine %.5f below target", i, c)
		}
	}
}

func TestVecBatchRoundTripMixed(t *testing.T) {
	const n, dim = 48, 128
	rows := make([]vecMixedRow, n)
	for i := range rows {
		rows[i] = vecMixedRow{
			ID:   "doc-" + string(rune('A'+i%5)),
			Tags: []string{"x", "y"},
			Emb:  sinVec64(i, dim),
			N:    i * 7,
		}
	}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.02))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatalf("encode: %v", err)
	}
	data := enc.Bytes()
	if data[5] != tagVecBatchStruct {
		t.Fatalf("expected 0xFE at byte 5, got 0x%02x", data[5])
	}
	var out []vecMixedRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != n {
		t.Fatalf("len %d", len(out))
	}
	for i := range rows {
		if out[i].ID != rows[i].ID || out[i].N != rows[i].N {
			t.Fatalf("row %d scalar mismatch: %q/%d vs %q/%d", i, out[i].ID, out[i].N, rows[i].ID, rows[i].N)
		}
		if len(out[i].Tags) != 2 || out[i].Tags[0] != "x" || out[i].Tags[1] != "y" {
			t.Fatalf("row %d tags %v", i, out[i].Tags)
		}
		if c := cosine64(rows[i].Emb, out[i].Emb); c < 0.999 {
			t.Fatalf("row %d cosine %.5f", i, c)
		}
	}
}

// TestVecBatchAnyFieldRoundTrip guards the schemaless (decodeAny) round trip.
// A []struct with a vector field held in an any-typed field used to encode as a
// batched block (tagVecBatchStruct) — which decodeAny has no case for, so the
// value failed to decode with ErrBadTag. The encoder now suppresses the batch in
// a dynamic-dispatch context (ifaceDepth>0) and falls back to the columnar path
// (tagColStruct), which decodeAny reads. Regression for the any-field gap.
func TestVecBatchAnyFieldRoundTrip(t *testing.T) {
	type wrap struct {
		Payload any
	}
	const n, dim = 32, 128
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, dim)}
	}
	data, err := Marshal(wrap{Payload: rows}, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Decode into the same shape: the any field routes through decodeAny.
	var out wrap
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal (any field with batchable []struct must round-trip, not ErrBadTag): %v", err)
	}
	if out.Payload == nil {
		t.Fatal("payload decoded nil")
	}
	// decodeAny materializes the []struct as a slice; assert the row count survived.
	rv, ok := out.Payload.([]any)
	if !ok {
		t.Fatalf("payload type %T, want []any", out.Payload)
	}
	if len(rv) != n {
		t.Fatalf("payload len %d want %d", len(rv), n)
	}
}

// TestVecBatchSmallerThanPerRow asserts the batched path is materially smaller
// than the per-vector (count=1) encoding the same data would otherwise produce.
func TestVecBatchSmallerThanPerRow(t *testing.T) {
	const n, dim = 64, 256
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, dim)}
	}
	batched, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	// Force the per-row path by encoding each row's single-element slice.
	perRow := 0
	for i := range rows {
		one := []vecOnlyRow{rows[i]} // n=1 < columnarMinElems => row-major per-vector
		b, err := Marshal(one, OptBalanced|OptLossyVec)
		if err != nil {
			t.Fatal(err)
		}
		perRow += len(b)
	}
	bpvBatched := float64(len(batched)) / n
	bpvPerRow := float64(perRow) / n
	t.Logf("batched=%.1f B/vec  per-row=%.1f B/vec  (%.0f%% smaller)",
		bpvBatched, bpvPerRow, 100*(1-bpvBatched/bpvPerRow))
	if bpvBatched >= bpvPerRow {
		t.Fatalf("batched %.1f not smaller than per-row %.1f B/vec", bpvBatched, bpvPerRow)
	}
}

// TestVecBatchVaryingDimFallsBack: rows with different vector lengths cannot be
// batched; must still round-trip (via row-major fallback).
func TestVecBatchVaryingDimFallsBack(t *testing.T) {
	const n = 32
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, 64+i)} // varying dim
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 5 && data[5] == tagVecBatchStruct {
		t.Fatalf("varying-dim should NOT batch, but emitted 0xFE")
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for i := range rows {
		if len(out[i].Emb) != 64+i {
			t.Fatalf("row %d dim %d want %d", i, len(out[i].Emb), 64+i)
		}
	}
}

// TestVecBatchOffByDefaultExact: without OptLossyVec the vectors round-trip
// bit-exact (no batching, no loss).
func TestVecBatchOffByDefaultExact(t *testing.T) {
	const n, dim = 32, 64
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, dim)}
	}
	data, err := Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 5 && data[5] == tagVecBatchStruct {
		t.Fatalf("no OptLossyVec but emitted 0xFE")
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		for j := range rows[i].Emb {
			if math.Float32bits(out[i].Emb[j]) != math.Float32bits(rows[i].Emb[j]) {
				t.Fatalf("row %d coord %d not bit-exact without lossy", i, j)
			}
		}
	}
}

type twoVecRow struct {
	A []float32
	B []float64
	K int
}

func TestVecBatchTwoVecFields(t *testing.T) {
	const n, dim = 40, 96
	rows := make([]twoVecRow, n)
	for i := range rows {
		rows[i] = twoVecRow{A: sinVec32(i, dim), B: sinVec64(i+1, dim), K: i}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	if data[5] != tagVecBatchStruct {
		t.Fatalf("want 0xFE, got 0x%02x", data[5])
	}
	var out []twoVecRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if out[i].K != i {
			t.Fatalf("row %d K=%d", i, out[i].K)
		}
		if cosine32(rows[i].A, out[i].A) < 0.99 || cosine64(rows[i].B, out[i].B) < 0.99 {
			t.Fatalf("row %d cosine low", i)
		}
	}
}

func TestVecBatchNaNInfPreserved(t *testing.T) {
	const n, dim = 32, 64
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		v := sinVec32(i, dim)
		v[0] = float32(math.NaN())
		v[1] = float32(math.Inf(1))
		v[2] = float32(math.Inf(-1))
		rows[i] = vecOnlyRow{Emb: v}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if !math.IsNaN(float64(out[i].Emb[0])) {
			t.Fatalf("row %d NaN not preserved: %v", i, out[i].Emb[0])
		}
		if !math.IsInf(float64(out[i].Emb[1]), 1) || !math.IsInf(float64(out[i].Emb[2]), -1) {
			t.Fatalf("row %d Inf not preserved", i)
		}
	}
}

func TestVecBatchBelowMinElemsNoBatch(t *testing.T) {
	// n >= 16 but vector dim < lossyVecMinElems(32) => not batchable.
	const n, dim = 32, 16
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, dim)}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 5 && data[5] == tagVecBatchStruct {
		t.Fatalf("dim<min should not batch")
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != n {
		t.Fatalf("len %d", len(out))
	}
}

func TestVecBatchEmptyAndNilRows(t *testing.T) {
	// All-empty vectors: dim 0 < min => no batch, must still round-trip incl
	// the nil-vs-empty distinction.
	rows := make([]vecOnlyRow, 20)
	for i := range rows {
		if i%2 == 0 {
			rows[i] = vecOnlyRow{Emb: nil}
		} else {
			rows[i] = vecOnlyRow{Emb: []float32{}}
		}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if (rows[i].Emb == nil) != (out[i].Emb == nil) {
			t.Fatalf("row %d nil-vs-empty mismatch: %v vs %v", i, rows[i].Emb, out[i].Emb)
		}
	}
}

func FuzzVecBatchDecode(f *testing.F) {
	rows := make([]vecOnlyRow, 32)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, 64)}
	}
	seed, _ := Marshal(rows, OptBalanced|OptLossyVec)
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		var out []vecOnlyRow
		_ = Unmarshal(data, &out) // must never panic / OOM
	})
}

// vecBatchPolarMask parses a tagVecBatchStruct stream and returns its polarMask
// byte (0 if not a batched stream). Layout after the 5-byte header:
//
//	0xFE, uvarint(n), nv, mask, polarMask, ...
func vecBatchPolarMask(t *testing.T, data []byte) byte {
	t.Helper()
	if len(data) < 6 || data[5] != tagVecBatchStruct {
		return 0
	}
	i := 6
	for i < len(data) && data[i] >= 0x80 { // skip uvarint(n) continuation bytes
		i++
	}
	i++ // last uvarint byte
	if i+3 > len(data) {
		return 0
	}
	return data[i+2] // nv, mask, polarMask
}

func TestVecBatchPolarVaryingNorm(t *testing.T) {
	const n, dim = 64, 256
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		v := sinVec32(i, dim)
		scale := float32(math.Exp(math.Sin(float64(i)) * 2)) // ~e^-2..e^2 spread
		for j := range v {
			v[j] *= scale
		}
		rows[i] = vecOnlyRow{Emb: v}
	}
	enc := NewEncoderWith(OptBalanced | OptLossyVec)
	enc.SetVectorBudget(MaxRelError(0.05))
	if err := enc.EncodeValue(rows); err != nil {
		t.Fatal(err)
	}
	data := enc.Bytes()
	if pm := vecBatchPolarMask(t, data); pm == 0 {
		t.Fatalf("polar should engage on varying-norm data, polarMask=0")
	}
	var out []vecOnlyRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		var se, ne float64
		for j := range rows[i].Emb {
			d := float64(rows[i].Emb[j] - out[i].Emb[j])
			se += d * d
			ne += float64(rows[i].Emb[j]) * float64(rows[i].Emb[j])
		}
		if rel := math.Sqrt(se / (ne + 1e-30)); rel > 0.08 {
			t.Fatalf("row %d relErr %.4f exceeds budget+slack", i, rel)
		}
	}
}

func TestVecBatchPolarSkippedUnitNorm(t *testing.T) {
	// Unit-norm vectors: norm CV ~0, polar must be skipped (polarMask=0).
	const n, dim = 64, 256
	rows := make([]vecOnlyRow, n)
	for i := range rows {
		v := sinVec32(i, dim)
		var s float64
		for _, x := range v {
			s += float64(x) * float64(x)
		}
		inv := float32(1 / math.Sqrt(s))
		for j := range v {
			v[j] *= inv
		}
		rows[i] = vecOnlyRow{Emb: v}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	if pm := vecBatchPolarMask(t, data); pm != 0 {
		t.Fatalf("polar should be skipped on unit-norm data, polarMask=0x%02x", pm)
	}
}

// NamedVec is a named []float32 type; it must NOT be batched (a named slice type
// could carry a custom codec), but must still round-trip via the row-major path.
type (
	NamedVec    []float32
	namedVecRow struct {
		V NamedVec
	}
)

func TestVecBatchExcludesNamedType(t *testing.T) {
	const n, dim = 32, 64
	rows := make([]namedVecRow, n)
	for i := range rows {
		rows[i] = namedVecRow{V: NamedVec(sinVec32(i, dim))}
	}
	data, err := Marshal(rows, OptBalanced|OptLossyVec)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 5 && data[5] == tagVecBatchStruct {
		t.Fatalf("named slice type must not batch (0xFE emitted)")
	}
	var out []namedVecRow
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if len(out[i].V) != dim {
			t.Fatalf("row %d len %d", i, len(out[i].V))
		}
	}
}

// TestVecBatchHostileCountNoOOM crafts a 0xFE block claiming a huge row count and
// asserts decode rejects it cleanly (no panic, no giant allocation).
func TestVecBatchHostileCountNoOOM(t *testing.T) {
	// Valid stream to get the 5-byte header, then a crafted 0xFE body.
	rows := make([]vecOnlyRow, 32)
	for i := range rows {
		rows[i] = vecOnlyRow{Emb: sinVec32(i, 64)}
	}
	good, _ := Marshal(rows, OptBalanced|OptLossyVec)
	hdr := good[:5] // QDF header
	// 0xFE, uvarint(huge), nv, mask, polarMask
	body := make([]byte, 0, 1+10+3)
	body = append(body, tagVecBatchStruct)
	huge := make([]byte, 0, 10)
	x := uint64(1) << 40
	for x >= 0x80 {
		huge = append(huge, byte(x)|0x80)
		x >>= 7
	}
	huge = append(huge, byte(x))
	body = append(body, huge...)
	body = append(body, 1, 1, 0) // nv=1, mask=1, polarMask=0
	hostile := append(append([]byte{}, hdr...), body...)
	var out []vecOnlyRow
	if err := Unmarshal(hostile, &out); err == nil {
		t.Fatal("expected error on hostile huge count, got nil")
	}
}
