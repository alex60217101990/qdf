package cgsample

import (
	"fmt"
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// ransRows is prefix-rich so rANS has real redundancy to remove — otherwise the
// assertions below would be measuring an entropy coder that had nothing to do
// and would pass whether or not it ran.
func ransRows(n int) []GenRow {
	out := make([]GenRow, n)
	for i := range out {
		id := fmt.Sprintf("%06d", i)
		out[i] = GenRow{
			ID:   int64(i),
			Name: "com.acme.platform.worker.service." + id,
			Inner: GenRowInner{
				X: i,
				Y: "/opt/acme/platform/bin/worker --shard=" + id,
			},
		}
	}
	return out
}

// A top-level value that is a GENERATED STRUCT must be rANS-framed like any
// other, and today it is not.
//
// reflect_encode.go marks any top-level Marshaler `customFramed`, and
// maybeApplyRANS then declines: "a top-level Marshaler forced Fast framing and
// its bytes are opts-invariant by contract". True of a hand-written Marshaler,
// which emits its own Fast body and ignores Options. NOT true of a generated
// EncoderMarshaler, which writes into the shared encoder and honours its mode —
// it calls StructShape and WriteStringField, both of which respect Dense.
//
// The slice arm is the anchor rather than decoration. The same rows as a
// []GenRow ARE framed, so it proves this data compresses; without it, a test
// that only checks the struct would pass the day rANS stops helping for an
// unrelated reason.
func TestTopLevelGeneratedStructIsRANSFramed(t *testing.T) {
	rows := ransRows(64)
	set := GenRowSet{Rows: rows}

	sliceBal, err := qdf.Marshal(rows, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	sliceRANS, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	if len(sliceRANS) >= len(sliceBal) {
		t.Fatalf("anchor: a slice of the same rows went %d -> %d with rANS — this data "+
			"does not compress, so the struct assertion below would prove nothing",
			len(sliceBal), len(sliceRANS))
	}

	structBal, err := qdf.Marshal(set, qdf.OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	structRANS, err := qdf.Marshal(set, qdf.OptBalanced|qdf.OptRANS)
	if err != nil {
		t.Fatal(err)
	}
	// The framing bit is the property this test is named for, and unlike a size
	// comparison it cannot be satisfied by degrading the other side: a
	// regression that INFLATED the OptBalanced wire would make a
	// smaller-than-balanced assertion easier to pass, not harder.
	if structRANS[4]&qdf.FlagRANS == 0 {
		t.Errorf("a top-level generated struct came back with header flag %08b, "+
			"FlagRANS clear — the body was not framed", structRANS[4])
	}
	if len(structRANS) >= len(structBal) {
		t.Errorf("a top-level generated struct went %d -> %d with rANS (no change), "+
			"while the same rows as a slice went %d -> %d (%.1f%% off)",
			len(structBal), len(structRANS), len(sliceBal), len(sliceRANS),
			float64(len(sliceBal)-len(sliceRANS))/float64(len(sliceBal))*100)
	}

	// Whatever framing it gets, the value must come back.
	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"balanced", qdf.OptBalanced},
		{"balanced+rans", qdf.OptBalanced | qdf.OptRANS},
		{"compression", qdf.OptCompression},
	} {
		b, err := qdf.Marshal(set, o.opts)
		if err != nil {
			t.Fatalf("%s: %v", o.name, err)
		}
		var got GenRowSet
		if err := qdf.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: decode: %v", o.name, err)
		}
		if len(got.Rows) != len(rows) {
			t.Fatalf("%s: %d rows, want %d", o.name, len(got.Rows), len(rows))
		}
		for i := range rows {
			if !reflect.DeepEqual(got.Rows[i], rows[i]) {
				t.Fatalf("%s row %d:\n got %+v\nwant %+v", o.name, i, got.Rows[i], rows[i])
			}
		}
	}
}

// The hand-written side of this distinction is NOT tested here on purpose.
// marshaler_framing_test.go in the root package already covers it and covers it
// more strictly — it asserts the flag byte is zero, where a version of this test
// only compared two wires for equality and would have missed a Balanced header
// stamped onto a hand-written body while rANS declined.
//
// That test was also verified to fail if this change is made too broadly (the
// kind distinction dropped, customFramed never set), so the guard is real and
// duplicating it here would only be a second thing to maintain.
