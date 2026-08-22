package bench

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"reflect"
	"testing"
)

// TestJSONTextCodecRoundTrips gates the hand-written jsontext codec against the
// two things that could make its benchmark numbers meaningless.
//
// A hand-rolled encoder that drops a field is faster than one that does not, and
// nothing else in this harness would notice: the benchmark only reads ns/op and
// the length of the output. So this checks the output against the reflection
// encoder byte for byte, and checks the hand-written decoder reproduces the
// value.
func TestJSONTextCodecRoundTrips(t *testing.T) {
	for _, n := range []struct {
		devices, samples int
	}{{1, 1}, {4, 16}, {32, 256}} {
		v := mkIoTBatch(n.devices, n.samples)

		enc := newJSONTextEncoder()
		got, err := enc.marshalIoTBatch(&v)
		if err != nil {
			t.Fatalf("%dx%d: marshal: %v", n.devices, n.samples, err)
		}

		// Byte-for-byte against encoding/json v1. The codec sorts map keys for
		// exactly this reason: without it the bytes differ by key order and the
		// size column would not line up with the v1 row it sits beside.
		want, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%dx%d: reference marshal: %v", n.devices, n.samples, err)
		}
		if string(got) != string(want) {
			t.Fatalf("%dx%d: jsontext output differs from encoding/json.\n got %d bytes\nwant %d bytes\nfirst divergence at %d",
				n.devices, n.samples, len(got), len(want), firstDiff(got, want))
		}

		// The hand-written decoder must reproduce the value, and so must v2's
		// reflection decoder reading the hand-written bytes — the second check
		// is what says the output is real JSON rather than something only this
		// file can read.
		var back IoTBatch
		dec := newJSONTextDecoder()
		if err := dec.unmarshalIoTBatch(got, &back); err != nil {
			t.Fatalf("%dx%d: jsontext unmarshal: %v", n.devices, n.samples, err)
		}
		if !reflect.DeepEqual(back, v) {
			t.Fatalf("%dx%d: jsontext decoder did not reproduce the value", n.devices, n.samples)
		}

		var viaV2 IoTBatch
		if err := jsonv2.Unmarshal(got, &viaV2); err != nil {
			t.Fatalf("%dx%d: json/v2 could not read the jsontext output: %v", n.devices, n.samples, err)
		}
		if !reflect.DeepEqual(viaV2, v) {
			t.Fatalf("%dx%d: json/v2 read the jsontext output into a different value", n.devices, n.samples)
		}
	}
}

func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
