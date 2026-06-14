package cgsample

import (
	"fmt"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// distinctSample returns a Sample whose string fields are unique per i so the
// arena copy path is genuinely exercised (no accidental dedup) and a stale read
// across a Reset would surface as a value mismatch.
func distinctSample(i int) Sample {
	return Sample{
		Name:   fmt.Sprintf("user-%06d", i),
		Age:    i,
		Active: i%2 == 0,
		Score:  float64(i) + 0.5,
		Tags:   []string{fmt.Sprintf("tag-a-%d", i), fmt.Sprintf("tag-b-%d", i)},
		Meta: map[string]string{
			"host":   fmt.Sprintf("node-%04d", i),
			"region": fmt.Sprintf("r-%d", i%7),
		},
		Inner:  Inner{X: i, Y: float64(i)},
		Buf:    []byte{byte(i), byte(i >> 8)},
		OptPtr: &Inner{X: -i, Y: -float64(i)},
		Counts: [3]int32{int32(i), int32(i + 1), int32(i + 2)},
	}
}

// TestCodegenArena_EqualsPlain guards that the GENERATED UnmarshalQDFArena
// decode (the codegen path, reached via qdf.Unmarshal because Sample implements
// UnmarshalerArena) is equal to a plain heap decode whether or not an arena is
// attached. Codegen emits Fast-mode wire (no intern table) so the strings flow
// through the inline ReadString arena path, not the Dense getString path — this
// pins that codegen + arena round-trips correctly.
func TestCodegenArena_EqualsPlain(t *testing.T) {
	for i := range 50 {
		v := distinctSample(i)
		src, err := v.MarshalQDF(nil)
		if err != nil {
			t.Fatal(err)
		}

		var plain Sample
		if _, err := plain.UnmarshalQDF(src); err != nil {
			t.Fatal(err)
		}

		// (a) generated arena entry point directly.
		a := qdf.NewArena()
		var viaArena Sample
		if _, err := viaArena.UnmarshalQDFArena(src, false, a); err != nil {
			t.Fatal(err)
		}
		if !equalSample(plain, viaArena) {
			t.Fatalf("i=%d: UnmarshalQDFArena decode diverges from plain decode", i)
		}

		// (b) via qdf.Unmarshal + WithArena (dispatches to the generated arena
		// method through the UnmarshalerArena interface).
		a2 := qdf.NewArena()
		var viaUnmarshal Sample
		if err := qdf.Unmarshal(src, &viaUnmarshal, qdf.WithArena(a2)); err != nil {
			t.Fatal(err)
		}
		if !equalSample(plain, viaUnmarshal) {
			t.Fatalf("i=%d: qdf.Unmarshal+WithArena diverges from plain decode", i)
		}
	}
}

// TestCodegenArena_ResetReuse reuses one arena across many distinct generated
// decodes with Reset between them, comparing each before the next Reset. Guards
// that the codegen arena path stays correct across Reset cycles.
func TestCodegenArena_ResetReuse(t *testing.T) {
	a := qdf.NewArena()
	for i := range 100 {
		v := distinctSample(i)
		src, err := v.MarshalQDF(nil)
		if err != nil {
			t.Fatal(err)
		}
		a.Reset()
		var out Sample
		if _, err := out.UnmarshalQDFArena(src, false, a); err != nil {
			t.Fatal(err)
		}
		if !equalSample(v, out) {
			t.Fatalf("i=%d: reset-reused codegen arena decode diverges from source", i)
		}
	}
}
