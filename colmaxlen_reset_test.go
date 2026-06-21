package qdf

import "testing"

func TestColMaxLenResetOnPooledReuse(t *testing.T) {
	// Poison a pooled decoder with a stale per-column bound.
	d := decPool.Get().(*Decoder)
	d.colMaxLen = 3
	decPool.Put(d)
	// A normal qpack slice longer than the stale bound must still decode (the
	// entry reset clears colMaxLen). qpack int decoders consult colLenOK.
	in := make([]int64, 64)
	for i := range in {
		in[i] = int64(i)
	}
	b, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []int64
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("stale colMaxLen leaked into pooled decode: %v", err)
	}
	if len(out) != 64 {
		t.Fatalf("got len %d want 64", len(out))
	}
}
