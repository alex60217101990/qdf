package qdf

import (
	"testing"
	"unsafe"
)

// TestNoCopy_DoesNotLeakIntoUnmarshalT pins that a WithNoCopy() decode never
// leaves the pooled decoder in aliasing mode for a later acquirer. Before the
// fix, unmarshal() set dec.noCopy=true and never reset it, so a subsequent
// UnmarshalT (which did not set noCopy) could inherit it and silently return
// strings aliasing the caller's input buffer — a use-after-free the race
// detector cannot catch.
func TestNoCopy_DoesNotLeakIntoUnmarshalT(t *testing.T) {
	type S struct {
		Name string `qdf:"name"`
	}
	data, err := Marshal(&S{Name: "a-service-name-long-enough-to-not-be-tiny"}, OptSpeed)
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
	for i := range 200 {
		// Prime the pool: a WithNoCopy decode sets the pooled decoder noCopy=true.
		var prime S
		if err := Unmarshal(data, &prime, WithNoCopy()); err != nil {
			t.Fatal(err)
		}
		// UnmarshalT must NOT inherit noCopy — it must return an owned copy.
		var out S
		if err := UnmarshalT(data, &out); err != nil {
			t.Fatal(err)
		}
		if within(out.Name) {
			t.Fatalf("iter %d: UnmarshalT returned a string aliasing the input — noCopy leaked from the pool", i)
		}
	}
}
