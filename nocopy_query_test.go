package qdf

import (
	"reflect"
	"testing"
	"unsafe"
)

type ncEntry struct {
	Service string `qdf:"service"`
	Msg     string `qdf:"msg"`
}
type ncBatch struct {
	Entries []ncEntry `qdf:"entries"`
}

func ncSample() *ncBatch {
	b := &ncBatch{Entries: make([]ncEntry, 50)}
	for i := range b.Entries {
		b.Entries[i] = ncEntry{
			Service: "api-gateway-prod",
			Msg:     "request handled id=" + string(rune('A'+i%26)) + "-detail-payload",
		}
	}
	return b
}

// Value-equality: WithNoCopy must decode to the same values as the default copy path.
func TestWithNoCopy_ValueEquality(t *testing.T) {
	for _, opt := range []Options{OptSpeed, OptBalanced} {
		data, err := Marshal(ncSample(), opt)
		if err != nil {
			t.Fatal(err)
		}
		var copyOut, ncOut ncBatch
		if err := Unmarshal(data, &copyOut); err != nil {
			t.Fatal(err)
		}
		if err := Unmarshal(data, &ncOut, WithNoCopy()); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(copyOut, ncOut) {
			t.Fatalf("opt %v: noCopy decode != copy decode", opt)
		}
	}
}

// Aliasing proof: a noCopy-decoded string's bytes live inside the input buffer;
// the default copy path's do not.
func TestWithNoCopy_Aliases(t *testing.T) {
	data, err := Marshal(ncSample(), OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	within := func(s string) bool {
		if len(s) == 0 || len(data) == 0 {
			return false
		}
		p := uintptr(unsafe.Pointer(unsafe.StringData(s)))
		base := uintptr(unsafe.Pointer(&data[0]))
		return p >= base && p < base+uintptr(len(data))
	}
	var ncOut ncBatch
	if err := Unmarshal(data, &ncOut, WithNoCopy()); err != nil {
		t.Fatal(err)
	}
	if !within(ncOut.Entries[0].Service) {
		t.Fatal("noCopy string does not alias input buffer")
	}
	var copyOut ncBatch
	if err := Unmarshal(data, &copyOut); err != nil {
		t.Fatal(err)
	}
	if within(copyOut.Entries[0].Service) {
		t.Fatal("copy-path string unexpectedly aliases input buffer")
	}
}
