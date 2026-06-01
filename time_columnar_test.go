package qdf

import (
	"testing"
	"time"
)

// timeColRow is the struct used for columnar time.Time round-trip tests.
type timeColRow struct {
	ID   int64     `qdf:"id"`
	Ts   time.Time `qdf:"ts"`
	Name string    `qdf:"name"`
}

// TestColumnarTime_RoundTrip verifies that a []struct containing a time.Time
// field is encoded via the columnar path (colKindTime) and decodes back with
// identical instants.  It exercises OptBalanced and OptBalanced|OptColumnIndex.
func TestColumnarTime_RoundTrip(t *testing.T) {
	const n = 200
	base := time.Unix(1_700_000_000, 0).UTC()

	in := make([]timeColRow, n)
	for i := range in {
		ts := base.Add(time.Duration(i) * time.Second)
		// Every 5th row gets a sub-second nanosecond component.
		if i%5 == 0 {
			ts = ts.Add(time.Duration(i*123_456) * time.Nanosecond)
		}
		in[i] = timeColRow{
			ID:   int64(i),
			Ts:   ts,
			Name: []string{"alpha", "beta", "gamma"}[i%3],
		}
	}
	// Edge rows: year 1 (zero Time) and year 9999.
	in[0] = timeColRow{ID: 0, Ts: time.Time{}, Name: "zero"}
	in[1] = timeColRow{ID: 1, Ts: time.Date(9999, 12, 31, 23, 59, 59, 999_999_999, time.UTC), Name: "far-future"}

	opts := []struct {
		name string
		opt  Options
	}{
		{"OptBalanced", OptBalanced},
		{"OptBalanced|OptColumnIndex", OptBalanced | OptColumnIndex},
	}

	for _, tc := range opts {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := Marshal(in, tc.opt)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			// The payload must have been encoded columnar.
			if !containsByte(buf, tagColStruct) {
				t.Fatalf("expected tagColStruct in encoded payload (columnar path not taken)")
			}

			var out []timeColRow
			if err := Unmarshal(buf, &out); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if len(out) != len(in) {
				t.Fatalf("len mismatch: got %d want %d", len(out), len(in))
			}

			for i := range in {
				want := in[i]
				got := out[i]

				// ID and Name must match exactly.
				if got.ID != want.ID {
					t.Errorf("row %d: ID mismatch: got %d want %d", i, got.ID, want.ID)
				}
				if got.Name != want.Name {
					t.Errorf("row %d: Name mismatch: got %q want %q", i, got.Name, want.Name)
				}

				// Timestamps: compare instants (sec + nsec), not Equal()
				// directly, so that out-of-UnixNano-range times (year 1,
				// year 9999) are still validated correctly.
				wantSec := want.Ts.UTC().Unix()
				wantNsec := want.Ts.UTC().Nanosecond()
				if got.Ts.Unix() != wantSec || got.Ts.Nanosecond() != wantNsec {
					t.Errorf("row %d: Ts instant mismatch:\n  want unix=%d nsec=%d (%v)\n  got  unix=%d nsec=%d (%v)",
						i, wantSec, wantNsec, want.Ts,
						got.Ts.Unix(), got.Ts.Nanosecond(), got.Ts)
				}

				// For in-range instants also assert Equal().
				const minUnixNanoSec = -9_223_372_036
				const maxUnixNanoSec = 9_223_372_036
				if wantSec >= minUnixNanoSec && wantSec <= maxUnixNanoSec {
					if !got.Ts.Equal(want.Ts) {
						t.Errorf("row %d: Ts.Equal false: want=%v got=%v", i, want.Ts, got.Ts)
					}
					if got.Ts.UnixNano() != want.Ts.UnixNano() {
						t.Errorf("row %d: UnixNano mismatch: want=%d got=%d", i, want.Ts.UnixNano(), got.Ts.UnixNano())
					}
				}
			}
		})
	}
}
