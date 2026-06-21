package qdf

import "testing"

// A nil slice/[]byte value inside a generated map fast path must round-trip as
// nil (tagNil), not collapse to an empty slice — matching the reflect encoder so
// the nil-vs-empty distinction is preserved and the two paths emit identical
// wire. Regression for the mapsgen kindStringSlice / kindBytes writeBlock.
func TestMapSliceValueNilPreserved(t *testing.T) {
	opts := []Options{OptSpeed, OptBalanced, OptCompression, OptCanonical | OptDense}
	for _, opt := range opts {
		// map[string][]string
		m1 := map[string][]string{"nilv": nil, "empty": {}, "full": {"x", "y"}}
		b1, err := Marshal(m1, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal []string: %v", opt, err)
		}
		var o1 map[string][]string
		if err := Unmarshal(b1, &o1); err != nil {
			t.Fatalf("opt=%v unmarshal []string: %v", opt, err)
		}
		if o1["nilv"] != nil {
			t.Errorf("opt=%v []string nil value -> %#v, want nil", opt, o1["nilv"])
		}
		if o1["empty"] == nil || len(o1["empty"]) != 0 {
			t.Errorf("opt=%v []string empty value -> %#v, want empty non-nil", opt, o1["empty"])
		}
		if len(o1["full"]) != 2 || o1["full"][0] != "x" || o1["full"][1] != "y" {
			t.Errorf("opt=%v []string full value lost -> %#v", opt, o1["full"])
		}

		// map[string][]byte
		m2 := map[string][]byte{"nilv": nil, "empty": {}, "full": {1, 2, 3}}
		b2, err := Marshal(m2, opt)
		if err != nil {
			t.Fatalf("opt=%v marshal []byte: %v", opt, err)
		}
		var o2 map[string][]byte
		if err := Unmarshal(b2, &o2); err != nil {
			t.Fatalf("opt=%v unmarshal []byte: %v", opt, err)
		}
		if o2["nilv"] != nil {
			t.Errorf("opt=%v []byte nil value -> %#v, want nil", opt, o2["nilv"])
		}
		if o2["empty"] == nil || len(o2["empty"]) != 0 {
			t.Errorf("opt=%v []byte empty value -> %#v, want empty non-nil", opt, o2["empty"])
		}
		if len(o2["full"]) != 3 {
			t.Errorf("opt=%v []byte full value lost -> %#v", opt, o2["full"])
		}
	}
}
