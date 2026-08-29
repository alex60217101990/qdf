package qdf

import (
	"bytes"
	"math/rand"
	"testing"
)

// hybridHexRec is a string-only HYBRID element whose string columns are ALL
// high-cardinality restricted-alphabet (hex) IDs + a residual map, NO numeric
// and NO low-card enum column. This is exactly the case the OLD codegen gate
// (StringColumnsBeneficial) declines (dict/per-value ≈ raw) but the intern-aware
// reflect path accepts (alpha-packs the hex) — the divergence the hybrid gate
// closes.
type hybridHexRec struct {
	Span   string            `qdf:"span"`
	Parent string            `qdf:"parent"`
	Tags   map[string]string `qdf:"tags"`
}

func mkHybridHexRecs(n int) []hybridHexRec {
	r := rand.New(rand.NewSource(7))
	hex := func(ln int) string {
		const h = "0123456789abcdef"
		b := make([]byte, ln)
		for j := range b {
			b[j] = h[r.Intn(16)]
		}
		return string(b)
	}
	out := make([]hybridHexRec, n)
	for i := range out {
		out[i] = hybridHexRec{Span: hex(16), Parent: hex(16), Tags: map[string]string{"k": "v"}}
	}
	return out
}

func hasByte(b []byte, x byte) bool { return bytes.IndexByte(b, x) >= 0 }

// TestCodegenGate_MatchesReflectInternAware: for a hybrid hex-only element the
// reflect path flips it into a columnar (hybrid) frame and alpha-packs the IDs;
// the NEW codegen gate (StringColumnsBeneficialHybrid) agrees, while the OLD gate
// (StringColumnsBeneficial) would have kept codegen row-major — the divergence.
func TestCodegenGate_MatchesReflectInternAware(t *testing.T) {
	recs := mkHybridHexRecs(2000)
	b, err := Marshal(recs, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !hasByte(b, tagHybridColStruct) {
		t.Fatalf("reflect did not emit a hybrid columnar frame for the hybrid hex element (wire=%d)", len(b))
	}
	spans := make([]string, len(recs))
	parents := make([]string, len(recs))
	for i, r := range recs {
		spans[i] = r.Span
		parents[i] = r.Parent
	}
	if !StringColumnsBeneficialHybrid(spans, parents) {
		t.Fatal("codegen hybrid gate declined columnar where reflect accepted it (divergence)")
	}
	if StringColumnsBeneficial(spans, parents) {
		t.Fatal("expected the plain gate to decline (the gap the hybrid gate closes); it accepted")
	}
}

// TestCodegenGate_ConservativeStaysRowMajor: a hybrid element whose string column
// does NOT compress (full-alphabet high-card) must be declined by the hybrid gate
// too, so codegen and reflect both keep it row-major.
func TestCodegenGate_ConservativeStaysRowMajor(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	cols := make([]string, 2000)
	for i := range cols {
		b := make([]byte, 24)
		for j := range b {
			b[j] = byte(32 + r.Intn(95))
		}
		cols[i] = string(b)
	}
	if StringColumnsBeneficialHybrid(cols) {
		t.Fatal("hybrid gate accepted an incompressible full-alphabet column (would bloat)")
	}
}
