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
	for k := 0; k < 3; k++ {      // multiple decodes into the SAME backing
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
