package qdf

import "testing"

type reuseNum struct {
	A int64   `qdf:"a"`
	B uint64  `qdf:"b"`
	C float64 `qdf:"c"`
	D bool    `qdf:"d"`
}

// row-major (small n) + columnar (n>=16) both exercised by varying size.
func reuseRoundTrip(t *testing.T, n int) {
	in := make([]reuseNum, n)
	for i := range in {
		in[i] = reuseNum{A: int64(i), B: uint64(i * 2), C: float64(i) * 1.5, D: i%2 == 0}
	}
	buf, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]reuseNum, 0, n) // pre-sized → reuse path
	for k := range 3 {            // multiple decodes into the SAME backing
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("n=%d iter=%d: %v", n, k, err)
		}
		if len(out) != n {
			t.Fatalf("n=%d len=%d", n, len(out))
		}
		for i := range in {
			if out[i] != in[i] {
				t.Fatalf("n=%d iter=%d row %d: %+v != %+v", n, k, i, out[i], in[i])
			}
		}
	}
}

func TestReuse_RowMajorAndColumnar(t *testing.T) {
	reuseRoundTrip(t, 8)   // row-major (< columnarMinElems)
	reuseRoundTrip(t, 100) // columnar
}

// The clear MUST zero fields the wire shape does not set, else a reused
// backing leaks stale data (schema evolution: decode []small into []big).
type small struct {
	A int64 `qdf:"a"`
}
type big struct {
	A int64 `qdf:"a"`
	B int64 `qdf:"b"` // absent in the wire
}

func TestReuse_ClearsStaleFields(t *testing.T) {
	in := make([]small, 50)
	for i := range in {
		in[i] = small{A: int64(i)}
	}
	buf, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]big, 50, 64) // pre-sized reuse
	for i := range out {
		out[i] = big{A: -999, B: 0xDEADBEEF} // sentinel stale
	}
	out = out[:0]
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	for i := range out {
		if out[i].A != int64(i) {
			t.Fatalf("row %d A=%d", i, out[i].A)
		}
		if out[i].B != 0 {
			t.Fatalf("row %d B=%d, stale not cleared", i, out[i].B)
		}
	}
}

// Pointer-containing element types must still decode correctly (fresh path).
func TestReuse_PointerTypeUnaffected(t *testing.T) {
	type withStr struct {
		A int64  `qdf:"a"`
		S string `qdf:"s"`
	}
	in := make([]withStr, 50)
	for i := range in {
		in[i] = withStr{A: int64(i), S: "svc"}
	}
	buf, _ := Marshal(in, OptBalanced)
	out := make([]withStr, 0, 64)
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if out[i] != in[i] {
			t.Fatalf("row %d: %+v", i, out[i])
		}
	}
}

// v2: pointer-containing element reuse must also clear stale fields the wire
// shape omits (schema evolution) — via reflect.Value.Clear, not byte-clear.
type smallS struct {
	A int64 `qdf:"a"`
}
type bigS struct {
	A int64  `qdf:"a"`
	S string `qdf:"s"` // absent in the wire
}

func TestReuse_PointerStaleCleared(t *testing.T) {
	in := make([]smallS, 50)
	for i := range in {
		in[i] = smallS{A: int64(i)}
	}
	buf, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]bigS, 50, 64)
	for i := range out {
		out[i] = bigS{A: -1, S: "STALE"} // sentinel pointer field
	}
	out = out[:0]
	if err := Unmarshal(buf, &out); err != nil {
		t.Fatal(err)
	}
	for i := range out {
		if out[i].A != int64(i) || out[i].S != "" {
			t.Fatalf("row %d = %+v, stale string not cleared", i, out[i])
		}
	}
}

// TestDecodePtrReuse verifies that decodePtr reuses the existing heap object
// on repeated decode instead of allocating a fresh one each time.
func TestDecodePtrReuse(t *testing.T) {
	type Inner struct {
		X int64 `qdf:"x"`
	}
	type Outer struct {
		P *Inner `qdf:"p"`
	}

	b1, err := Marshal(Outer{P: &Inner{X: 42}}, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var dst Outer
	if err := Unmarshal(b1, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.P == nil || dst.P.X != 42 {
		t.Fatalf("first decode: %+v", dst)
	}
	first := dst.P // save address for reuse check

	b2, err := Marshal(Outer{P: &Inner{X: 99}}, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(b2, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.P == nil || dst.P.X != 99 {
		t.Fatalf("second decode: %+v", dst)
	}
	// After the fix, decodePtr reuses the existing allocation.
	if dst.P != first {
		t.Errorf("pointer not reused: first=%p second=%p", first, dst.P)
	}

	// Also verify nil-tag path: encoding a nil pointer must clear the field.
	b3, err := Marshal(Outer{P: nil}, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(b3, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.P != nil {
		t.Fatalf("nil not written: %+v", dst)
	}
}

// BenchmarkDecodePtrReuse measures the allocation savings from reusing the
// existing heap object in decodePtr on repeated decodes into the same struct.
func BenchmarkDecodePtrReuse(b *testing.B) {
	type Inner struct {
		X int64   `qdf:"x"`
		Y float64 `qdf:"y"`
	}
	type Outer struct {
		P *Inner `qdf:"p"`
	}
	src := Outer{P: &Inner{X: 42, Y: 3.14}}
	buf, err := Marshal(src, OptSpeed)
	if err != nil {
		b.Fatal(err)
	}
	var dst Outer
	// Prime dst with a live allocation so the reuse path is exercised.
	if err := Unmarshal(buf, &dst); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := Unmarshal(buf, &dst); err != nil {
			b.Fatal(err)
		}
	}
}
