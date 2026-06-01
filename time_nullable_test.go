package qdf

import (
	"testing"
	"time"
)

// timeMatrixRow is the struct used by TestTime_MatrixBundles for non-columnar
// (B1..B5, B8, B9) bundle testing of a struct WITH a time.Time field.
type timeMatrixRow struct {
	ID   int64      `qdf:"id"`
	Ts   time.Time  `qdf:"ts"`
	Opt  *time.Time `qdf:"opt"`
	Name string     `qdf:"name"`
}

// assertTimeMatrixRow compares two timeMatrixRow values with per-field
// assertions: ID/Name exact, Ts/Opt by Unix()+Nanosecond() plus .Equal()
// for in-range instants. Using explicit field comparison avoids reflect.DeepEqual
// issues with time.Time (monotonic clock readings, location pointers).
func assertTimeMatrixRow(t *testing.T, bundle string, i int, got, want timeMatrixRow) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("[%s] row %d ID: got %d want %d", bundle, i, got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("[%s] row %d Name: got %q want %q", bundle, i, got.Name, want.Name)
	}

	const minUnixNanoSec = -9_223_372_036
	const maxUnixNanoSec = 9_223_372_036

	checkTime := func(field string, g, w time.Time) {
		wSec, wNsec := w.UTC().Unix(), w.UTC().Nanosecond()
		if g.Unix() != wSec || g.Nanosecond() != wNsec {
			t.Errorf("[%s] row %d %s instant: got unix=%d nsec=%d, want unix=%d nsec=%d",
				bundle, i, field, g.Unix(), g.Nanosecond(), wSec, wNsec)
		}
		if wSec >= minUnixNanoSec && wSec <= maxUnixNanoSec {
			if !g.Equal(w) {
				t.Errorf("[%s] row %d %s: Equal false: got=%v want=%v", bundle, i, field, g, w)
			}
		}
	}

	checkTime("Ts", got.Ts, want.Ts)

	if want.Opt == nil {
		if got.Opt != nil {
			t.Errorf("[%s] row %d Opt: got non-nil, want nil", bundle, i)
		}
	} else {
		if got.Opt == nil {
			t.Errorf("[%s] row %d Opt: got nil, want non-nil (%v)", bundle, i, *want.Opt)
		} else {
			checkTime("Opt", *got.Opt, *want.Opt)
		}
	}
}

// makeTimeMatrixRows builds ~150 rows: monotonic Ts increments, ~1/3 nil Opt,
// a zero-Time row at index 0, and a year-9999 row at index 1. All times are
// constructed in UTC so that decoded .UTC() values are reflect.DeepEqual-safe
// (no monotonic reading, same Location pointer). This lets the matrix helpers
// use explicit per-field comparison which is safe for all time values.
func makeTimeMatrixRows(n int) []timeMatrixRow {
	base := time.Unix(1_700_000_000, 0).UTC()
	rows := make([]timeMatrixRow, n)
	for i := range rows {
		ts := base.Add(time.Duration(i)*time.Second + time.Duration(i*777)*time.Millisecond)
		var opt *time.Time
		if i%3 != 0 {
			v := base.Add(time.Duration(i) * time.Minute).UTC()
			opt = &v
		}
		rows[i] = timeMatrixRow{
			ID:   int64(i),
			Ts:   ts.UTC(),
			Opt:  opt,
			Name: []string{"alpha", "beta", "gamma", "delta"}[i%4],
		}
	}
	// Edge rows.
	rows[0] = timeMatrixRow{ID: 0, Ts: time.Time{}, Opt: nil, Name: "zero"}
	rows[1] = timeMatrixRow{
		ID:   1,
		Ts:   time.Date(9999, 12, 31, 23, 59, 59, 999_999_999, time.UTC),
		Opt:  nil,
		Name: "far-future",
	}
	return rows
}

// TestTime_MatrixBundles exercises time.Time and *time.Time under every codec
// bundle.  Because time.Time is not safely comparable with reflect.DeepEqual
// (monotonic reading, location pointer), we use explicit per-field assertions
// via assertTimeMatrixRow rather than the roundtripBundles/roundtripColumnar
// helpers (which rely on reflect.DeepEqual).
func TestTime_MatrixBundles(t *testing.T) {
	rows := makeTimeMatrixRows(150)

	for _, b := range matrixBundles() {
		t.Run(b.name, func(t *testing.T) {
			data, err := Marshal(rows, b.opts)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out []timeMatrixRow
			if err := Unmarshal(data, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(out) != len(rows) {
				t.Fatalf("len: got %d want %d", len(out), len(rows))
			}
			for i := range rows {
				assertTimeMatrixRow(t, b.name, i, out[i], rows[i])
			}
		})
	}
}

// TestNullableTime_Columnar verifies that a []struct{ Opt *time.Time } round-trips
// correctly under various codec options and null percentages.
func TestNullableTime_Columnar(t *testing.T) {
	base := time.Unix(1_600_000_000, 0).UTC()

	makeRows := func(n, nullPct int) []struct{ Opt *time.Time } {
		rows := make([]struct{ Opt *time.Time }, n)
		for i := range rows {
			if i%100 < nullPct {
				rows[i].Opt = nil
			} else {
				v := base.Add(time.Duration(i) * time.Second).UTC()
				rows[i].Opt = &v
			}
		}
		return rows
	}

	type tc struct {
		name string
		opts Options
	}
	cases := []tc{
		{"OptBalanced", OptBalanced},
		{"OptBalanced|OptColumnIndex", OptBalanced | OptColumnIndex},
		{"OptCompression", OptCompression},
	}

	for _, nullPct := range []int{0, 30, 95, 100} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				rows := makeRows(200, nullPct)
				data, err := Marshal(rows, c.opts)
				if err != nil {
					t.Fatalf("null%%=%d %s marshal: %v", nullPct, c.name, err)
				}
				var out []struct{ Opt *time.Time }
				if err := Unmarshal(data, &out); err != nil {
					t.Fatalf("null%%=%d %s unmarshal: %v", nullPct, c.name, err)
				}
				if len(out) != len(rows) {
					t.Fatalf("null%%=%d %s len: got %d want %d", nullPct, c.name, len(out), len(rows))
				}
				for i := range rows {
					want := rows[i].Opt
					got := out[i].Opt
					if want == nil {
						if got != nil {
							t.Errorf("null%%=%d %s row %d: got non-nil, want nil", nullPct, c.name, i)
						}
					} else {
						if got == nil {
							t.Errorf("null%%=%d %s row %d: got nil, want %v", nullPct, c.name, i, *want)
						} else if !got.Equal(*want) {
							t.Errorf("null%%=%d %s row %d: Equal false: got=%v want=%v", nullPct, c.name, i, *got, *want)
						}
					}
				}
			})
		}
	}
}
