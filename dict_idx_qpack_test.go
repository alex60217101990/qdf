package qdf

import (
	"math/rand"
	"testing"
)

// A skewed (Zipf-like) low-cardinality string column must dictionary-compress
// just as well as a uniform one: the dict index spends ceil(log2 d) bits/row
// regardless of distribution, so wire size is set by cardinality, not skew.
//
// Before the never-worse gate was corrected, a skewed column whose dominant
// value clusters into few runs was REJECTED from the dictionary form (the gate
// modeled the per-value fallback as run-length-collapsing, which the gathered-
// column loop never does) and fell back to per-value strings — bloating it
// ~2-3x. The fallback costs >= n bytes (one >=1-byte ref per row), so for any
// d < 256 the bitpacked index is strictly smaller and dict must be chosen.
type dictQRow struct {
	Status string
	Seq    int64
}

func dictQInputs(n int, skewed bool) []dictQRow {
	r := rand.New(rand.NewSource(11))
	// Longer values make the per-value-vs-dict gap unmistakable.
	vals := []string{
		"status_active_connection_established_ok",
		"status_pending_retry_backoff_queued",
		"status_closed_graceful_shutdown_done",
		"status_failed_timeout_deadline_exceeded",
		"status_draining_inflight_requests_wait",
	}
	rows := make([]dictQRow, n)
	for i := range rows {
		if skewed {
			if r.Float64() < 0.90 {
				rows[i].Status = vals[0]
			} else {
				rows[i].Status = vals[1+r.Intn(len(vals)-1)]
			}
		} else {
			rows[i].Status = vals[i%len(vals)] // even spread
		}
		rows[i].Seq = int64(i)
	}
	return rows
}

func TestDictIndexSkewNotBloated(t *testing.T) {
	const n = 2000
	uni, err := MarshalT(dictQInputs(n, false), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	skew, err := MarshalT(dictQInputs(n, true), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// Same distinct table, same n: both must take the dict form, so the skewed
	// column may not be materially larger than the uniform one. (Pre-fix it was
	// ~2.7x larger.) Allow a small slack for index-codec/framing differences.
	if float64(len(skew)) > 1.15*float64(len(uni)) {
		t.Fatalf("skewed dict column bloated: skew=%d uniform=%d (%.2fx); gate likely rejecting dict for clustered low-card data",
			len(skew), len(uni), float64(len(skew))/float64(len(uni)))
	}
}

func TestDictIndexSkewRoundTrip(t *testing.T) {
	const n = 2000
	for _, skewed := range []bool{false, true} {
		in := dictQInputs(n, skewed)
		for _, opt := range []Options{OptBalanced, OptCompression} {
			b, err := MarshalT(in, opt)
			if err != nil {
				t.Fatalf("marshal skewed=%v opt=%v: %v", skewed, opt, err)
			}
			var out []dictQRow
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("unmarshal skewed=%v opt=%v: %v", skewed, opt, err)
			}
			if len(out) != len(in) {
				t.Fatalf("len skewed=%v opt=%v: got %d want %d", skewed, opt, len(out), len(in))
			}
			for i := range in {
				if out[i] != in[i] {
					t.Fatalf("row %d skewed=%v opt=%v: got %+v want %+v", i, skewed, opt, out[i], in[i])
				}
			}
		}
	}
}
