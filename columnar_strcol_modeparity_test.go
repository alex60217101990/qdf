package qdf_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// TestColumnarStringColumn_DecodeModeParity proves the columnar string-column
// decode (after the materializeStr noCopy/arena change) returns byte-identical
// values in all three decode modes — default (owned copy), arena (bump-allocated),
// and noCopy (aliases the input). It exercises the dict-column path (low-card
// repeated strings) which is the site whose table strings now alias d.buf under
// noCopy; a lifetime/aliasing bug there would surface as a value mismatch.
func TestColumnarStringColumn_DecodeModeParity(t *testing.T) {
	type Row struct {
		ID   int64  `qdf:"id"`
		Tag  string `qdf:"tag"`  // 3 distinct → dict column
		Host string `qdf:"host"` // few distinct → dict column
		Note string `qdf:"note"` // higher card → raw / per-value column
	}
	tags := []string{"alpha", "beta", "gamma"}
	hosts := []string{"h1", "h2"}
	in := make([]Row, 0, 64)
	for i := range 64 { // >= columnarMinElems(16) → columnar transpose
		in = append(in, Row{
			ID:   int64(i),
			Tag:  tags[i%len(tags)],
			Host: hosts[i%len(hosts)],
			Note: fmt.Sprintf("note-%d-%s", i, tags[i%len(tags)]),
		})
	}

	for _, opt := range []qdf.Options{qdf.OptCompression, qdf.OptBalanced | qdf.OptColumnIndex} {
		data, err := qdf.Marshal(&in, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal: %v", opt, err)
		}
		// Confirm the payload really took the columnar path (else the test would
		// pass trivially via row-major and not exercise materializeStr). 0xEF =
		// tagColStruct (pure columnar), 0xF7 = tagHybridColStruct.
		if bytes.IndexByte(data, 0xEF) < 0 && bytes.IndexByte(data, 0xF7) < 0 {
			t.Fatalf("opt=%v did not emit a columnar struct block (no 0xEF/0xF7)", opt)
		}

		check := func(mode string, opts ...qdf.QueryOption) {
			t.Helper()
			// Each decode gets its OWN buffer copy: noCopy aliases it, so it must
			// stay alive + unmodified for the lifetime of out — which it does here.
			buf := append([]byte(nil), data...)
			var out []Row
			if err := qdf.Unmarshal(buf, &out, opts...); err != nil {
				t.Fatalf("opt=%v %s decode: %v", opt, mode, err)
			}
			if !reflect.DeepEqual(out, in) {
				t.Fatalf("opt=%v %s: value mismatch\n got %v\nwant %v", opt, mode, out, in)
			}
		}
		check("default")
		check("nocopy", qdf.WithNoCopy())
		check("arena", qdf.WithArena(qdf.NewArena()))
	}
}
