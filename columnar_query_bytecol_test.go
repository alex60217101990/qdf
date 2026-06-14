package qdf

import (
	"bytes"
	"testing"
)

type qByteRec struct {
	Name []byte `qdf:"name"`
	Code int32  `qdf:"code"`
}

// TestQueryByteColumnPredicate guards the case where a []byte column is BOTH
// projected (decoded into cv.bs, leaving cv.s nil) AND referenced by a string
// predicate. The predicate eval indexed cv.s[i] unconditionally and panicked on
// the nil slice; it must read the value from cv.bs instead.
func TestQueryByteColumnPredicate(t *testing.T) {
	const n = 32
	in := make([]qByteRec, n)
	for i := range in {
		if i%2 == 0 {
			in[i].Name = []byte("keep")
		} else {
			in[i].Name = []byte("drop")
		}
		in[i].Code = int32(i)
	}
	buf, err := Marshal(&in, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out []qByteRec
	if err := Unmarshal(buf, &out, Where("name", func(s string) bool { return s == "keep" })); err != nil {
		t.Fatalf("query decode: %v", err)
	}
	if len(out) != n/2 {
		t.Fatalf("got %d rows, want %d", len(out), n/2)
	}
	for _, r := range out {
		if !bytes.Equal(r.Name, []byte("keep")) {
			t.Fatalf("predicate let through %q", r.Name)
		}
	}
}
