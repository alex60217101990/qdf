package qdf

import (
	"reflect"
	"testing"
)

// bundle is one meaningfully-distinct encoder configuration.
type bundle struct {
	name         string
	opts         Options
	columnarOnly bool // distinguishing behaviour only shows on []struct payloads
}

// matrixBundles is the single source of truth for the config matrix.
func matrixBundles() []bundle {
	return []bundle{
		{"B1_Speed", OptSpeed, false},
		{"B2_Dense", OptDense, false},
		{"B3_QPack", OptQPack, false},
		{"B4_Balanced", OptBalanced, false},
		{"B5_Compression", OptCompression, false},
		{"B6_Balanced_ColIndex", OptBalanced | OptColumnIndex, true},
		{"B7_Compression_ColIndex", OptCompression | OptColumnIndex, true},
		{"B8_QPack_Gorilla", OptQPack | OptGorillaFloat, false},
		{"B9_Dense_RANS", OptDense | OptRANS, false},
	}
}

// roundtripBundles encodes value under every non-columnar-only bundle,
// decodes into a fresh *T, and asserts deep equality. Returns the wire
// per bundle keyed by name.
func roundtripBundles[T any](t *testing.T, value T) map[string][]byte {
	t.Helper()
	wires := map[string][]byte{}
	for _, b := range matrixBundles() {
		if b.columnarOnly {
			continue
		}
		t.Run(b.name, func(t *testing.T) {
			data, err := Marshal(value, b.opts)
			if err != nil {
				t.Fatalf("marshal %s: %v", b.name, err)
			}
			var out T
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("unmarshal %s: %v", b.name, err)
			}
			if !reflect.DeepEqual(value, out) {
				t.Fatalf("%s roundtrip mismatch:\n in=%#v\nout=%#v", b.name, value, out)
			}
			wires[b.name] = data
		})
	}
	return wires
}

// roundtripColumnar runs ALL bundles (incl. B6/B7) over a []struct payload.
func roundtripColumnar[T any](t *testing.T, rows []T) {
	t.Helper()
	for _, b := range matrixBundles() {
		t.Run(b.name, func(t *testing.T) {
			data, err := Marshal(rows, b.opts)
			if err != nil {
				t.Fatalf("marshal %s: %v", b.name, err)
			}
			var out []T
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("unmarshal %s: %v", b.name, err)
			}
			if !reflect.DeepEqual(rows, out) {
				t.Fatalf("%s columnar mismatch", b.name)
			}
		})
	}
}

func TestMatrix_Smoke(t *testing.T) {
	type rec struct {
		A int64
		B string
		C []float64
		D *int
	}
	n := 7
	roundtripBundles(t, rec{A: -5, B: "hello", C: []float64{1, 2, 3}, D: &n})
}

func TestMatrix_ColumnarSmoke(t *testing.T) {
	type row struct {
		ID  int64
		Tag string
	}
	rows := []row{{1, "a"}, {2, "b"}, {2, "a"}, {3, "c"}}
	roundtripColumnar(t, rows)
}
