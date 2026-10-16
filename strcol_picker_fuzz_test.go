package qdf

import (
	"bytes"
	"reflect"
	"testing"
)

// FuzzStringColumnPicker round-trips string columns through every codec the
// picker can choose. The picker measures FSST, alphabet packing and front-delta
// and emits only the smallest, from state each measure staged, so a measure that
// leaves its staging stale or an emit that reads another codec's staging would
// write a block that decodes to the wrong rows. Three columns per row so the
// per-column FSST cache is crossed too.
func FuzzStringColumnPicker(f *testing.F) {
	f.Add([]byte("deadbeef01\ncafebabe02\n0123456789\nabcdef0000"), uint8(200))
	f.Add([]byte("https://a.example/x/1\nhttps://a.example/x/2\nhttps://a.example/y/3"), uint8(64))
	f.Add([]byte("level=info component=api msg=\"request done\" status=200 path=/v1/users\nlevel=warn component=db msg=\"slow query\" took=1.2s table=orders"), uint8(250))
	type row struct {
		A string `qdf:"a"`
		B string `qdf:"b"`
		C string `qdf:"c"`
	}
	const hex = "0123456789abcdef"
	f.Fuzz(func(t *testing.T, seed []byte, nRows uint8) {
		parts := bytes.Split(seed, []byte{'\n'})
		if len(parts) == 0 || nRows < 16 {
			return
		}
		rows := make([]row, int(nRows))
		for i := range rows {
			p := parts[i%len(parts)]
			// Every column is high-cardinality (the row index is in each value),
			// so none is settled by the dictionary before the picker runs:
			// A is the seed folded onto hex (alphabet packing), B a shared prefix
			// with a row tail (front-delta), C the raw seed text with the index
			// spliced in (FSST, when enabled).
			ab := make([]byte, 0, len(p)+4)
			for _, c := range p {
				ab = append(ab, hex[c&15])
			}
			ab = append(ab, hex[i>>4&15], hex[i&15])
			rows[i] = row{
				A: string(ab),
				B: string(p) + "/" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)),
				C: string(p) + " id=" + string(rune('A'+i%26)) + string(p[:len(p)/2]) + string(rune('0'+i%10)),
			}
		}
		for _, o := range []Options{OptBalanced, OptCompression, OptBalanced | OptFSST} {
			w, err := Marshal(rows, o)
			if err != nil {
				t.Fatal(err)
			}
			var back []row
			if err := Unmarshal(w, &back); err != nil {
				t.Fatalf("opts %#x: decode: %v", o, err)
			}
			if !reflect.DeepEqual(back, rows) {
				t.Fatalf("opts %#x: round-trip mismatch", o)
			}
		}
	})
}
