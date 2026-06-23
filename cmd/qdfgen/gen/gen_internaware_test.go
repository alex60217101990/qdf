package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestColumnarGate_InternAwareForHybrid pins that the generated columnar gate for
// a string-only element matches the reflect probe: a HYBRID string-only element
// (residual fields, no numeric) uses StringColumnsBeneficialHybrid (intern-aware
// + alpha), while a PURE string-only element keeps the plain StringColumnsBeneficial.
func TestColumnarGate_InternAwareForHybrid(t *testing.T) {
	gen := func(t *testing.T, types ...string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.go")
		if err := Generate([]string{"./testdata/internaware"}, Options{Types: types, OutFile: out}); err != nil {
			t.Fatalf("Generate(%v): %v", types, err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		return string(b)
	}

	t.Run("hybrid_string_only_uses_intern_aware_gate", func(t *testing.T) {
		src := gen(t, "HybridStrHolder", "HybridStrElem")
		if !strings.Contains(src, "StringColumnsBeneficialHybrid(") {
			t.Fatalf("hybrid string-only element must gate columnar via StringColumnsBeneficialHybrid:\n%s", src)
		}
	})

	t.Run("pure_string_only_uses_plain_gate", func(t *testing.T) {
		src := gen(t, "PureStrHolder", "PureStrElem")
		if strings.Contains(src, "StringColumnsBeneficialHybrid(") {
			t.Fatalf("pure string-only element must NOT use the intern-aware gate:\n%s", src)
		}
		if !strings.Contains(src, "StringColumnsBeneficial(") {
			t.Fatalf("pure string-only element must gate columnar via StringColumnsBeneficial:\n%s", src)
		}
	})
}
