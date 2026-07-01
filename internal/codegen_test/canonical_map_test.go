package cgsample

import (
	"bytes"
	"math"
	"os"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// TestGenerated_CanonicalMapDeterministic guards that generated EncodeQDF emits
// map entries in sorted key order under OptCanonical. Go map iteration is
// randomized, so before the emitEncodeMap canonical-sort fix two encodes of the
// same map produced different byte order under OptCanonical (breaking the
// canonical byte-identical guarantee). With the fix every encode is identical.
func TestGenerated_CanonicalMapDeterministic(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	in := Sample{
		Name: "x",
		Meta: map[string]string{
			"zeta": "1", "alpha": "2", "mike": "3", "bravo": "4",
			"yankee": "5", "charlie": "6", "november": "7", "delta": "8",
			"oscar": "9", "kilo": "10", "echo": "11", "papa": "12",
		},
	}
	var first []byte
	for i := range 40 {
		e := qdf.NewEncoderWith(qdf.OptCanonical)
		if err := (&in).EncodeQDF(e); err != nil {
			t.Fatalf("EncodeQDF: %v", err)
		}
		b := append([]byte(nil), e.Bytes()...)
		if i == 0 {
			first = b
			continue
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("canonical generated map output not deterministic at run %d", i)
		}
	}
}

// TestGenerated_CanonicalFloatMapNaNKey guards the float-keyed canonical branch:
// it must carry (key,value) pairs, not re-fetch values by map index. A NaN key is
// unfindable by map index (NaN != NaN), so `expr[NaN]` returns the zero value —
// the buggy form silently encoded "" for the NaN key. It also checks determinism
// (Float64bits sort) across runs.
func TestGenerated_CanonicalFloatMapNaNKey(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	nan := math.NaN()
	in := GenFloatMap{M: map[float64]string{
		1.5: "a", -2.0: "b", nan: "nanval", 0.0: "z", 3.25: "c",
	}}

	var first []byte
	for i := range 30 {
		e := qdf.NewEncoderWith(qdf.OptCanonical)
		if err := (&in).EncodeQDF(e); err != nil {
			t.Fatalf("EncodeQDF: %v", err)
		}
		b := append([]byte(nil), e.Bytes()...)
		if i == 0 {
			first = b
		} else if !bytes.Equal(b, first) {
			t.Fatalf("canonical float-map output not deterministic at run %d", i)
		}
	}

	// Round-trip: the NaN key's value must survive (regression: it was "" before).
	var out GenFloatMap
	if _, err := out.UnmarshalQDF(first); err != nil {
		t.Fatalf("UnmarshalQDF: %v", err)
	}
	if len(out.M) != len(in.M) {
		t.Fatalf("len(out.M)=%d want %d", len(out.M), len(in.M))
	}
	var nanVal string
	nanCount := 0
	for k, v := range out.M {
		if math.IsNaN(k) {
			nanCount++
			nanVal = v
		}
	}
	if nanCount != 1 {
		t.Fatalf("nan key count=%d want 1", nanCount)
	}
	if nanVal != "nanval" {
		t.Fatalf("nan key value=%q want %q (value lost — pair-gather regression)", nanVal, "nanval")
	}
	for k, want := range map[float64]string{1.5: "a", -2.0: "b", 0.0: "z", 3.25: "c"} {
		if out.M[k] != want {
			t.Fatalf("M[%v]=%q want %q", k, out.M[k], want)
		}
	}
}
