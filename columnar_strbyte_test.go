package qdf

import (
	"bytes"
	"fmt"
	"testing"
)

type colStrField struct {
	A    int64  `qdf:"a"`
	Name string `qdf:"name"`
}

type colByteField struct {
	A    int64  `qdf:"a"`
	Name []byte `qdf:"name"`
}

// TestColumnarStringIntoByteField asserts a columnar string column decodes
// correctly into a []byte target field — the same string<->[]byte schema
// interchange that works row-major. Regression for the block-codec
// (Raw/Dict/FSST) gate that skipped the block reader when the target was
// []byte and fell into a per-value loop that misread the codec tag.
func TestColumnarStringIntoByteField(t *testing.T) {
	cases := []struct {
		name string
		opt  Options
		gen  func(i int) string
	}{
		{"compression-highcard", OptCompression, func(i int) string { return fmt.Sprintf("trace-%016x-%d", i*2654435761, i) }},
		{"compression-lowcard", OptCompression, func(i int) string { return []string{"alpha", "bravo", "charlie", "delta"}[i&3] }},
		{"balanced-highcard", OptBalanced, func(i int) string { return fmt.Sprintf("dev-%08x-%d", i*40503, i) }},
		{"speed-highcard", OptSpeed, func(i int) string { return fmt.Sprintf("host-%d-%x", i, i*7) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const n = 1000
			in := make([]colStrField, n)
			for i := range in {
				in[i].A = int64(i)
				in[i].Name = tc.gen(i)
			}
			buf, err := Marshal(&in, tc.opt)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			// Baseline: string target must work.
			var outS []colStrField
			if err := Unmarshal(buf, &outS); err != nil {
				t.Fatalf("string target decode: %v", err)
			}

			// The fix: []byte target must work and match.
			var outB []colByteField
			if err := Unmarshal(buf, &outB); err != nil {
				t.Fatalf("[]byte target decode: %v", err)
			}
			if len(outB) != n {
				t.Fatalf("len = %d, want %d", len(outB), n)
			}
			for i := range outB {
				if !bytes.Equal(outB[i].Name, []byte(in[i].Name)) {
					t.Fatalf("row %d: got %q want %q", i, outB[i].Name, in[i].Name)
				}
			}
		})
	}
}
