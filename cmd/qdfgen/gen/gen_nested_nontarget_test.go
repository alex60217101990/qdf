package gen

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNestedNonTargetStructErrors pins the audit-4 diagnostic: a target struct
// referencing a nested struct that is neither a -type target nor a custom-codec
// type must fail generation with a clear, actionable error naming the missing
// type — not emit an EncodeNested/DecodeNested call that fails to compile later.
func TestNestedNonTargetStructErrors(t *testing.T) {
	for _, typeName := range []string{"Parent", "PtrParent"} {
		t.Run(typeName, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.go")
			err := Generate([]string{"./testdata/nestednontarget"}, Options{
				Types:   []string{typeName},
				OutFile: out,
			})
			if err == nil {
				t.Fatalf("%s: Generate succeeded; want a nested-struct diagnostic", typeName)
			}
			if !strings.Contains(err.Error(), "Inner") || !strings.Contains(err.Error(), "-type") {
				t.Fatalf("%s: error %q must name the missing type Inner and the -type fix", typeName, err)
			}
		})
	}
}
