package qdf

import (
	"testing"
	"time"
)

// TestScalarTime_RoundTrip tests that time.Time values round-trip correctly
// through the scalar wire codec, including full-range instants outside the
// UnixNano-representable window (year 1, year 9999, pre-1970 negatives).
func TestScalarTime_RoundTrip(t *testing.T) {
	type wrapper struct {
		T time.Time `qdf:"t"`
	}

	cases := []struct {
		name string
		in   time.Time
	}{
		{
			name: "zero_year1",
			in:   time.Time{}, // year 1, January 1 — outside UnixNano range
		},
		{
			name: "year9999",
			in:   time.Date(9999, 12, 31, 23, 59, 59, 999_999_999, time.UTC),
		},
		{
			name: "pre1970_1900",
			in:   time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "epoch",
			in:   time.Unix(0, 0).UTC(),
		},
		{
			name: "recent_with_nsec",
			in:   time.Unix(1_700_000_000, 123_456_789).UTC(),
		},
		{
			name: "non_utc_input_instant_preserved",
			// Zone is NOT preserved — only the instant is. Decoded result will be UTC.
			in: time.Date(2020, 6, 1, 12, 0, 0, 0, time.FixedZone("x", 3600)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			src := wrapper{T: in}

			buf, err := Marshal(src, OptSpeed)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var dst wrapper
			if err := Unmarshal(buf, &dst); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			got := dst.T

			// The codec normalises to UTC; compare instants, not wall-clock forms.
			wantSec := in.UTC().Unix()
			wantNsec := in.UTC().Nanosecond()

			if got.Unix() != wantSec || got.Nanosecond() != wantNsec {
				t.Fatalf("instant mismatch:\n  in   = %v (unix=%d nsec=%d)\n  got  = %v (unix=%d nsec=%d)",
					in, wantSec, wantNsec,
					got, got.Unix(), got.Nanosecond())
			}

			// For instants that fit in UnixNano range (approx year 1678..2262)
			// also verify the full nanosecond equality.
			const minUnixNanoSec = -9_223_372_036 // ~year 1678
			const maxUnixNanoSec = 9_223_372_036  // ~year 2262
			if wantSec >= minUnixNanoSec && wantSec <= maxUnixNanoSec {
				if !got.Equal(in) {
					t.Fatalf("Equal mismatch for in-range instant: in=%v got=%v", in, got)
				}
			}

			// Non-UTC input: the decoded instant must equal the input instant
			// (zone not preserved, instant is).
			if !got.Equal(in) {
				// Only a hard failure for cases where Equal MUST hold.
				// For full-range times outside UnixNano, Equal may not be
				// meaningful — we already checked Unix+Nanosecond above.
				// So only fail if the instant is in-range.
				if wantSec >= minUnixNanoSec && wantSec <= maxUnixNanoSec {
					t.Fatalf("got.Equal(in) false: in=%v got=%v", in, got)
				}
			}
		})
	}
}

// FuzzRoundTrip_Time verifies the scalar time codec over random (sec, nsec) pairs.
func FuzzRoundTrip_Time(f *testing.F) {
	type wrapper struct {
		T time.Time `qdf:"t"`
	}

	// Seed corpus: cover epoch, negative (pre-1970), and a recent timestamp.
	f.Add(int64(0), uint32(0))
	f.Add(int64(1_700_000_000), uint32(123_456_789))
	f.Add(int64(-2_208_988_800), uint32(0))            // 1900-01-01
	f.Add(int64(-62_135_596_800), uint32(999_999_999)) // year 1 approx

	f.Fuzz(func(t *testing.T, sec int64, nsec uint32) {
		if nsec >= 1_000_000_000 {
			nsec = nsec % 1_000_000_000
		}
		in := time.Unix(sec, int64(nsec)).UTC()

		buf, err := Marshal(wrapper{T: in}, OptSpeed)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var out wrapper
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if out.T.Unix() != in.Unix() || out.T.Nanosecond() != in.Nanosecond() {
			t.Fatalf("instant mismatch: in=%v got=%v", in, out.T)
		}
		if !out.T.Equal(in) {
			t.Fatalf("Equal mismatch: in=%v got=%v", in, out.T)
		}
	})
}
