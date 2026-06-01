package qdf

import "testing"

func TestLargeColumnar_Arr32(t *testing.T) {
	type row struct {
		ID  int64
		Opt *int64
		Tag string
	}
	const n = 70000
	rows := make([]row, n)
	for i := range rows {
		rows[i].ID = int64(i)
		rows[i].Tag = "t"
		if i%3 == 0 {
			v := int64(i)
			rows[i].Opt = &v
		}
	}
	for _, opts := range []Options{OptBalanced | OptColumnIndex, OptCompression | OptColumnIndex} {
		data, err := Marshal(rows, opts)
		if err != nil {
			t.Fatalf("opts=%d: %v", opts, err)
		}
		var out []row
		if err := Unmarshal(data, &out); err != nil {
			t.Fatalf("opts=%d: %v", opts, err)
		}
		if len(out) != n {
			t.Fatalf("opts=%d len %d != %d", opts, len(out), n)
		}
		if out[69999].ID != 69999 || out[0].Opt == nil || *out[0].Opt != 0 || out[1].Opt != nil {
			t.Fatalf("opts=%d large columnar content mismatch at boundaries", opts)
		}
	}
}
