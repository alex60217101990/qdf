package cgsample

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
	qdfgen "github.com/alex60217101990/qdf/cmd/qdfgen/gen"
)

const generatedFile = "sample_qdf.go"

func TestGenerate(t *testing.T) {
	dir, _ := filepath.Abs(".")
	err := qdfgen.Generate([]string{"./..."}, qdfgen.Options{
		Types:   []string{"Sample", "Inner", "Edge", "GenMetric", "GenMetricBatch", "GenEvent", "GenEventLog", "GenRowInner", "GenRow", "GenRowSet", "GenName", "GenNameList", "GenBlobRow", "GenBlobSet", "GenOpt", "GenOptSet", "GenTrailed", "GenNamedCodec", "GenFloatMap"},
		OutFile: filepath.Join(dir, generatedFile),
		Verbose: testing.Verbose(),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := os.Stat(generatedFile); err != nil {
		t.Fatalf("expected %s to be written: %v", generatedFile, err)
	}
}

func TestRoundTrip_Generated(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	in := Sample{
		Name:   "alice",
		Age:    33,
		Active: true,
		Score:  98.6,
		Tags:   []string{"a", "b", "c"},
		Meta:   map[string]string{"k1": "v1", "k2": "v2"},
		Inner:  Inner{X: 7, Y: 1.5},
		When:   time.Unix(1700000000, 0),
		Buf:    []byte{1, 2, 3, 4, 5},
		OptPtr: &Inner{X: 99, Y: -2.0},
		Counts: [3]int32{10, 20, 30},
	}
	// Generated Marshaler implements qdf.Marshaler — pick it up via reflect
	// to avoid an import cycle on the build of this test file when the
	// generated file is absent.
	marshaler, ok := any(&in).(interface {
		MarshalQDF(dst []byte) ([]byte, error)
	})
	if !ok {
		t.Fatalf("Sample does not implement MarshalQDF — generator produced no method")
	}
	b, err := marshaler.MarshalQDF(nil)
	if err != nil {
		t.Fatalf("MarshalQDF: %v", err)
	}
	var out Sample
	unmarshaler, ok := any(&out).(interface {
		UnmarshalQDF(src []byte) (int, error)
	})
	if !ok {
		t.Fatalf("Sample does not implement UnmarshalQDF")
	}
	if _, err := unmarshaler.UnmarshalQDF(b); err != nil {
		t.Fatalf("UnmarshalQDF: %v", err)
	}
	if !equalSample(in, out) {
		t.Fatalf("mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// TestRoundTrip_GeneratedViaPublicAPI ensures the generated MarshalQDF data
// can also be decoded through the public qdf.Unmarshal (reflect path).
func TestRoundTrip_GeneratedViaPublicAPI(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	in := Sample{Name: "bob", Tags: []string{"x"}}
	marshaler, ok := any(&in).(interface {
		MarshalQDF(dst []byte) ([]byte, error)
	})
	if !ok {
		t.Skip("generated MarshalQDF not present")
	}
	b, err := marshaler.MarshalQDF(nil)
	if err != nil {
		t.Fatal(err)
	}
	var out Sample
	if err := qdf.Unmarshal(b, &out); err != nil {
		t.Fatalf("qdf.Unmarshal: %v", err)
	}
	if in.Name != out.Name {
		t.Fatalf("name mismatch: %q vs %q", in.Name, out.Name)
	}
}

func equalSample(a, b Sample) bool {
	if a.Name != b.Name || a.Age != b.Age || a.Active != b.Active || a.Score != b.Score {
		return false
	}
	if !reflect.DeepEqual(a.Tags, b.Tags) {
		return false
	}
	if !reflect.DeepEqual(a.Meta, b.Meta) {
		return false
	}
	if a.Inner != b.Inner {
		return false
	}
	if !a.When.Equal(b.When) {
		return false
	}
	if !bytes.Equal(a.Buf, b.Buf) {
		return false
	}
	if (a.OptPtr == nil) != (b.OptPtr == nil) {
		return false
	}
	if a.OptPtr != nil && *a.OptPtr != *b.OptPtr {
		return false
	}
	if a.Counts != b.Counts {
		return false
	}
	return true
}
