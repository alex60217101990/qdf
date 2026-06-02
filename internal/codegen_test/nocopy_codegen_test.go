package cgsample

import (
	"os"
	"reflect"
	"testing"
	"unsafe"

	qdf "github.com/alex60217101990/qdf"
)

func ncCodegenSample() Sample {
	return Sample{
		Name:   "service-name-long-enough-to-not-be-tiny",
		Age:    42,
		Active: true,
		Score:  3.5,
		Tags:   []string{"alpha-tag-value", "beta-tag-value"},
		Buf:    []byte("some-binary-payload-bytes-here"),
	}
}

// Generated UnmarshalQDFOpts(noCopy=true) must decode to the same values as the
// copying path, and the decoded string/[]byte must alias the source buffer.
func TestCodegen_NoCopy(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	data, err := qdf.Marshal(ncCodegenSample(), qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}

	// Value-equality: copy vs noCopy (both via the generated decoder).
	var copyOut, ncOut Sample
	if _, err := copyOut.UnmarshalQDF(data[5:]); err != nil { // body after 5-byte header
		t.Fatalf("copy decode: %v", err)
	}
	if _, err := ncOut.UnmarshalQDFOpts(data[5:], true); err != nil {
		t.Fatalf("noCopy decode: %v", err)
	}
	if !reflect.DeepEqual(copyOut, ncOut) {
		t.Fatal("codegen noCopy decode != copy decode")
	}

	// Aliasing: the noCopy string must point inside data.
	within := func(p uintptr) bool {
		base := uintptr(unsafe.Pointer(&data[0]))
		return p >= base && p < base+uintptr(len(data))
	}
	if !within(uintptr(unsafe.Pointer(unsafe.StringData(ncOut.Name)))) {
		t.Fatal("codegen noCopy string does not alias the source buffer")
	}
	if within(uintptr(unsafe.Pointer(unsafe.StringData(copyOut.Name)))) {
		t.Fatal("codegen copy-path string unexpectedly aliases the source buffer")
	}
}

// The reflect path (qdf.Unmarshal + WithNoCopy) into a codegen Unmarshaler type
// must thread noCopy through to the generated UnmarshalQDFOpts.
func TestCodegen_NoCopy_ViaReflectUnmarshal(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	data, err := qdf.Marshal(ncCodegenSample(), qdf.OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	var out Sample
	if err := qdf.Unmarshal(data, &out, qdf.WithNoCopy()); err != nil {
		t.Fatal(err)
	}
	base := uintptr(unsafe.Pointer(&data[0]))
	p := uintptr(unsafe.Pointer(unsafe.StringData(out.Name)))
	if !(p >= base && p < base+uintptr(len(data))) {
		t.Fatal("WithNoCopy did not thread through to codegen UnmarshalQDFOpts")
	}
}
