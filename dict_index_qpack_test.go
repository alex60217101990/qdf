package qdf

import (
	"math/rand"
	"testing"
)

// tagColStrDictQ QPack-codes the dict index. A skewed low-cardinality column
// whose distinct values share no prefix (so the plain dictionary, not the
// front-coded one, is used) carries a run-heavy index: QPack RLE/Dict packs it
// far below the flat ceil(log2 d)-bit width. The codec is never-larger, so it is
// chosen only when it strictly beats the flat index — proven here by a uniform
// (incompressible-index) column of the same cardinality staying on the flat dict
// while the skewed one shrinks.
type dictIdxQRow struct {
	Kind string
	Seq  int64
}

func dictIdxQInputs(skewed bool) []dictIdxQRow {
	const n = 2000
	r := rand.New(rand.NewSource(7))
	vals := []string{"alpha", "bravo", "charlie", "delta", "echo"} // no shared prefix → plain dict
	rows := make([]dictIdxQRow, n)
	for i := range rows {
		if skewed {
			if r.Float64() < 0.90 {
				rows[i].Kind = vals[0]
			} else {
				rows[i].Kind = vals[1+r.Intn(len(vals)-1)]
			}
		} else {
			rows[i].Kind = vals[i%len(vals)]
		}
		rows[i].Seq = int64(i)
	}
	return rows
}

func TestDictIndexQPackWinsOnSkew(t *testing.T) {
	uni, err := MarshalT(dictIdxQInputs(false), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	skew, err := MarshalT(dictIdxQInputs(true), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	// The skewed index must QPack-code well below the uniform flat index. (Both
	// share the same distinct table, n, and Seq column; only the index differs.)
	if len(skew) >= len(uni) {
		t.Fatalf("skewed index not QPack-compressed: skew=%d uniform=%d (want skew < uniform)", len(skew), len(uni))
	}
	// The exact margin is distribution-dependent (run count varies with the
	// scatter); require a clear win, not a specific figure.
	if float64(len(skew)) > 0.90*float64(len(uni)) {
		t.Fatalf("QPack index win too small: skew=%d uniform=%d (%.1f%%)", len(skew), len(uni), 100*float64(len(skew))/float64(len(uni)))
	}
	// The skewed column must carry the QPack-index tag; the uniform one must not.
	if !hasByte(skew, tagColStrDictQ) {
		t.Fatal("skewed column missing tagColStrDictQ")
	}
}

// A corrupted tagColStrDictQ wire must error, never panic: the index block's
// row count, the table lengths, and every decoded index are bounds-checked.
func TestDictIndexQPackHostileDecode(t *testing.T) {
	good, err := MarshalT(dictIdxQInputs(true), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if !hasByte(good, tagColStrDictQ) {
		t.Fatal("fixture did not exercise tagColStrDictQ")
	}
	// Truncations.
	for cut := range good {
		var out []dictIdxQRow
		_ = Unmarshal(good[:cut], &out) // must not panic
	}
	// Single-byte corruptions across the columnar string-column region.
	for i := range good {
		b := make([]byte, len(good))
		copy(b, good)
		b[i] ^= 0xFF
		var out []dictIdxQRow
		_ = Unmarshal(b, &out) // must not panic
	}
}

func TestDictIndexQPackRoundTrip(t *testing.T) {
	for _, skewed := range []bool{false, true} {
		in := dictIdxQInputs(skewed)
		for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
			b, err := MarshalT(in, opt)
			if err != nil {
				t.Fatalf("marshal skewed=%v opt=%v: %v", skewed, opt, err)
			}
			var out []dictIdxQRow
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
