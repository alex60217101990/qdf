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
	gen := func(t *testing.T, types ...string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.go")
		err := Generate([]string{"./testdata/customcodec"}, Options{
			Types:   types,
			OutFile: out,
		})
		if err != nil {
			t.Fatalf("Generate(%v): %v", types, err)
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
		// PlainElem must also be a target: the columnar field's row-major fallback
		// (len < columnarMinElems) encodes each element via EncodeNested.
		src := gen(t, "PlainHolder", "PlainElem")
		if !strings.Contains(src, "WriteColStructHeader") {
			t.Fatalf("PlainHolder should columnar-transpose its plain all-scalar element:\n%s", src)
		}
	})

	// A []struct element with a defined-byte-element slice field (`type B byte;
	// []B`) must NOT classify that field as a columnar Bytes column: the Bytes
	// emit assumes a literal []byte and would generate `unsafe.SliceData([]B)`
	// (*B, not *byte) and `[]B([]byte(...))` (an illegal conversion) — neither
	// compiles. It must fall through to the generic per-element encoder.
	t.Run("defined_byte_elem_slice_not_bytes_column", func(t *testing.T) {
		// NamedByteElem is a nested struct, so it must itself be a target (else
		// qdfgen now rejects the row-major EncodeNested against a non-generated
		// type — see TestNestedNonTargetStructErrors).
		src := gen(t, "NamedByteHolder", "NamedByteElem")
		// The broken Bytes-column emit dereferences the field via unsafe.SliceData;
		// the generic row-major path uses WriteArrayHeader instead.
		if strings.Contains(src, "unsafe.SliceData(") && strings.Contains(src, ".Data") {
			t.Fatalf("[]NamedByte field was emitted as a Bytes column (non-compiling):\n%s", src)
		}
		if !strings.Contains(src, "WriteArrayHeader") {
			t.Fatalf("[]NamedByte field should encode via the generic per-element path (WriteArrayHeader):\n%s", src)
		}
	})
}
