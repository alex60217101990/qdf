package qdf

import (
	"fmt"
	"testing"
)

type strRawRec struct {
	Seq   int64  `qdf:"seq"`   // columnar-favorable: makes the struct go columnar
	TS    int64  `qdf:"ts"`    // so the string fields become columns
	GUID  string `qdf:"guid"`  // high-cardinality → bulk-blob (tagColStrRaw)
	Email string `qdf:"email"` // high-cardinality
	Dept  string `qdf:"dept"`  // low-cardinality → dict
}

func makeStrRawRecs(n, distinct int) []strRawRec {
	if distinct < 1 {
		distinct = 1
	}
	depts := []string{"Eng", "Sales", "HR", "Legal", "IT"}
	rs := make([]strRawRec, n)
	for i := range rs {
		d := i % distinct
		rs[i] = strRawRec{
			Seq:   int64(i),
			TS:    int64(1700000000 + i),
			GUID:  fmt.Sprintf("%032x-%d", uint64(d)*2654435761, d),
			Email: fmt.Sprintf("user.%06d@corp.contoso.example.com", d),
			Dept:  depts[i%len(depts)],
		}
	}
	return rs
}

// Round-trip across the full cardinality range and both copy modes. tagColStrRaw
// fires on the high-cardinality columns; the value must survive exactly.
func TestStrRawRoundTrip(t *testing.T) {
	for _, distinct := range []int{5000, 4000, 2500, 1000, 100, 10, 1} {
		recs := makeStrRawRecs(5000, distinct)
		for _, opts := range []Options{OptBalanced, OptCompression} {
			blob, err := Marshal(recs, opts)
			if err != nil {
				t.Fatalf("distinct=%d opts=%d marshal: %v", distinct, opts, err)
			}
			var out []strRawRec
			if err := Unmarshal(blob, &out); err != nil {
				t.Fatalf("distinct=%d opts=%d unmarshal: %v", distinct, opts, err)
			}
			if len(out) != len(recs) {
				t.Fatalf("distinct=%d opts=%d len %d != %d", distinct, opts, len(out), len(recs))
			}
			for i := range recs {
				if out[i] != recs[i] {
					t.Fatalf("distinct=%d opts=%d row %d: %+v != %+v", distinct, opts, i, out[i], recs[i])
				}
			}
		}
	}
}

// The bulk codec must never produce a larger wire than leaving the column to the
// per-value path: it is emitted only when its estimated size is no larger, so a
// build with the codec available is always ≤ a build that could not use it. We
// approximate "could not use it" by the low-cardinality regime (where the gate
// declines and per-value/dict is chosen), checking the wire never balloons.
func TestStrRawNeverLargerThanRaw(t *testing.T) {
	// A high-cardinality column: bulk must be no larger than the per-value form.
	// Compare against OptSpeed (no columnar transpose, plain per-value strings)
	// as an upper-bound reference for the same data.
	recs := makeStrRawRecs(5000, 5000)
	balanced, err := Marshal(recs, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	speed, err := Marshal(recs, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(balanced) > len(speed) {
		t.Fatalf("Balanced (with bulk codec) %d larger than OptSpeed %d", len(balanced), len(speed))
	}
}

// noCopy aliases the decoder window; bulk-blob rows are sub-slices of the input
// buffer. The values must still round-trip while the buffer is live.
func TestStrRawNoCopyRoundTrip(t *testing.T) {
	recs := makeStrRawRecs(2000, 2000)
	blob, err := Marshal(recs, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var out []strRawRec
	if err := Unmarshal(blob, &out, WithNoCopy()); err != nil {
		t.Fatal(err)
	}
	for i := range recs {
		if out[i] != recs[i] {
			t.Fatalf("row %d: %+v != %+v", i, out[i], recs[i])
		}
	}
}

// A corrupt tagColStrRaw block must error cleanly, never panic or read out of
// range: truncate the valid blob at every offset and decode each prefix.
func TestStrRawHostileTruncation(t *testing.T) {
	recs := makeStrRawRecs(64, 64)
	blob, err := Marshal(recs, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(blob); cut++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic decoding truncated prefix len=%d: %v", cut, r)
				}
			}()
			var out []strRawRec
			_ = Unmarshal(blob[:cut], &out) // error expected; must not panic
		}()
	}
}
