package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestColumnarSkipsCustomCodecElem pins that columnarElemPlan does not transpose
// a []struct field whose element has a hand-written MarshalQDF/UnmarshalQDF: the
// transpose would replay the struct layout and bypass the custom codec,
// diverging from the reflect path (mirrors reflect commit 9c6f524). A plain
// all-scalar element must still be columnar-transposed (guard must not over-fire).
func TestColumnarSkipsCustomCodecElem(t *testing.T) {
	gen := func(t *testing.T, typeName string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.go")
		err := Generate([]string{"./testdata/customcodec"}, Options{
			Types:   []string{typeName},
			OutFile: out,
		})
		if err != nil {
			t.Fatalf("Generate(%s): %v", typeName, err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		return string(b)
	}

	t.Run("custom_codec_elem_stays_row_major", func(t *testing.T) {
		src := gen(t, "CustomHolder")
		if strings.Contains(src, "WriteColStructHeader") {
			t.Fatalf("CustomHolder columnar-transposed a custom-codec element (bypasses MarshalQDF):\n%s", src)
		}
		if !strings.Contains(src, "EncodeNested") || !strings.Contains(src, "DecodeNested") {
			t.Fatalf("CustomHolder must route through the custom codec via EncodeNested/DecodeNested:\n%s", src)
		}
	})

	t.Run("plain_elem_is_columnar", func(t *testing.T) {
		src := gen(t, "PlainHolder")
		if !strings.Contains(src, "WriteColStructHeader") {
			t.Fatalf("PlainHolder should columnar-transpose its plain all-scalar element:\n%s", src)
		}
	})
}
