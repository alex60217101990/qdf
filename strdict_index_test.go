package qdf

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
)

// wantStrDictWireDigest pins the encoding of string columns across the shapes
// the dictionary builder discriminates between. The builder indexes distinct
// values through an open-addressed table rather than a map, and ids are handed
// out in first-appearance order — so any drift in probe order, in the
// distinct-cap bail, or in the high-cardinality bail would reorder the table
// and change these bytes. Verified equal against the map implementation.
const wantStrDictWireDigest = "07bd4590ab412d406d2f0fce455555169c76c6aa614a76ea5ad1af2b428d5e92"

func TestStrDictWireIdentity(t *testing.T) {
	type row struct {
		Svc  string `qdf:"svc"`
		Path string `qdf:"path"`
		Lvl  string `qdf:"lvl"`
	}
	mk := func(shape string, n int) []row {
		out := make([]row, n)
		for i := range out {
			switch shape {
			case "lowcard": // dict territory
				out[i] = row{Svc: "svc" + strconv.Itoa(i%6), Path: "/api/v1/x" + strconv.Itoa(i%4), Lvl: "info"}
			case "prefix": // front-coded table territory
				out[i] = row{Svc: "service/aaa" + strconv.Itoa(i%12), Path: "/very/long/shared/prefix/" + strconv.Itoa(i%9), Lvl: "warn"}
			case "atcap": // exactly at the distinct cap boundary
				out[i] = row{Svc: "s" + strconv.Itoa(i%256), Path: "p" + strconv.Itoa(i%300), Lvl: "e" + strconv.Itoa(i%2)}
			default: // highcard — must bail
				out[i] = row{Svc: "u" + strconv.Itoa(i), Path: "/p/" + strconv.Itoa(i*7), Lvl: strconv.Itoa(i)}
			}
		}
		return out
	}
	h := sha256.New()
	for _, shape := range []string{"lowcard", "prefix", "atcap", "highcard"} {
		for _, n := range []int{16, 64, 500, 5000} {
			for _, o := range []Options{OptSpeed, OptBalanced, OptQPack, OptCompression} {
				blob, err := Marshal(mk(shape, n), o)
				if err != nil {
					t.Fatal(err)
				}
				h.Write(blob)
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != wantStrDictWireDigest {
		t.Errorf("string-dict wire digest changed:\n got %s\nwant %s", got, wantStrDictWireDigest)
	}
}
