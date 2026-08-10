package bench

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/alex60217101990/qdf"
)

// A generated decoder walks a shape's names and reads values without knowing
// which field it is on, so a tagStrDelta value has no base to rebuild against
// unless the generated loop binds one. qdfgen now emits EnterField/LeaveField
// around every field for exactly that reason.
//
// This test crosses the two implementations: the reflect encoder writes the
// wire, the generated decoder reads it. That is the pairing the delta form
// broke before the generator learned about it, and the failure was a loud
// ErrBadTag rather than silent corruption — which is why nothing else caught it.
func TestStrDeltaGeneratedDecoderReadsReflectWire(t *testing.T) {
	mk := func(n int) []LogEntry {
		out := make([]LogEntry, n)
		base := time.Unix(1700000000, 0).UTC()
		for i := range out {
			out[i] = LogEntry{
				Time:    base.Add(time.Duration(i) * time.Second),
				Level:   []string{"info", "warn", "error"}[i%3],
				Service: "checkout-api",
				Host:    "host-eu-central-1-node-" + strconv.Itoa(i%64),
				Region:  "eu-central-1",
				// Long shared prefixes: the delta's territory.
				TraceID:  "4bf92f3577b34da6a3ce929d0e0e" + strconv.Itoa(100000+i),
				SpanID:   "00f067aa0ba902b" + strconv.Itoa(100000+i),
				Msg:      "request completed for tenant 9f3a operation " + strconv.Itoa(i),
				Duration: float64(i%997) / 7.0,
			}
		}
		return out
	}
	rows := mk(512)

	for _, opts := range []qdf.Options{
		qdf.OptBalanced,
		qdf.OptBalanced | qdf.OptCanonical,
		qdf.OptCompression,
	} {
		// Reflect encoder writes it.
		b, err := qdf.Marshal(LogBatch{Entries: rows}, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}

		// Generated decoder reads it, through the generated UnmarshalQDF —
		// which walks each element's shape and is the path that lacked a field
		// context before qdfgen learned to bind one.
		var gen LogBatch
		if _, err := gen.UnmarshalQDF(b); err != nil {
			t.Fatalf("opts=%d generated decode: %v", opts, err)
		}
		if !reflect.DeepEqual(gen.Entries, rows) {
			for i := range rows {
				if gen.Entries[i] != rows[i] {
					t.Fatalf("opts=%d row %d:\n got %+v\nwant %+v", opts, i, gen.Entries[i], rows[i])
				}
			}
			t.Fatalf("opts=%d: generated decode differs but no row does", opts)
		}
	}
}
