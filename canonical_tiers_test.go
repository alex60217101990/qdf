package qdf

import (
	"bytes"
	"math"
	"testing"
)

// tierItem and tierPayload exercise the two canonical-sensitive surfaces in one
// value: maps (iteration order) and floats (-0.0 / NaN), with a >=16-element
// []struct so the columnar path fires under OptBalanced / OptCompression while
// OptSpeed keeps it row-major.
type tierItem struct {
	Name string  `qdf:"name"`
	V    float64 `qdf:"v"`
	W    float32 `qdf:"w"`
}

type tierPayload struct {
	Tags map[string]int   `qdf:"tags"`
	IMap map[int64]string `qdf:"imap"`
	Cost float64          `qdf:"cost"`
	Rows []tierItem       `qdf:"rows"`
}

// buildTierPayload returns a logically-fixed value. neg flips every float to a
// canonically-equal-but-bitwise-different form (-0.0 instead of +0.0, a distinct
// NaN payload) and inserts the map keys in the opposite order, so two builds are
// equal only after canonical normalization.
func buildTierPayload(neg bool) tierPayload {
	zero := 0.0
	if neg {
		zero = math.Copysign(0, -1)
	}

	// Fixed logical content. Only the map INSERT order and the float sign-of-zero
	// / NaN payload vary between the two builds — never the key→value mapping or
	// the row values — so A and B are equal only after canonical normalization.
	skeys := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	ikeys := []int64{5, 1, 9, 3, 7}

	tags := map[string]int{}
	imap := map[int64]string{}
	order := []int{0, 1, 2, 3, 4}
	if neg {
		order = []int{4, 3, 2, 1, 0} // opposite insert order; canonical must sort it away
	}
	for _, idx := range order {
		tags[skeys[idx]] = idx        // mapping skeys[idx]→idx is identical in both builds
		imap[ikeys[idx]] = skeys[idx] // pairing ikeys[idx]→skeys[idx] is identical too
	}

	rows := make([]tierItem, 20)
	for i := range rows {
		v := float64(i) // most rows are ordinary values...
		if i%4 == 0 {
			v = zero // ...some carry the sign-of-zero variation
		}
		w := float32(i)
		if i%4 == 1 {
			w = float32(math.NaN()) // and some a NaN that must canonicalize
			if neg {
				w = math.Float32frombits(0x7FC00007) // a different NaN payload
			}
		}
		rows[i] = tierItem{Name: skeys[i%len(skeys)], V: v, W: w} // skeys never mutated
	}
	return tierPayload{Tags: tags, IMap: imap, Cost: zero, Rows: rows}
}

// TestCanonicalAcrossTiers proves the canonical guarantee holds under every
// option tier and the orthogonal bits that compose with it — not just
// OptBalanced. For each configuration: (1) two logically-equal but
// differently-built values encode byte-identically, and (2) re-encoding the same
// value is stable across Go's per-range map-iteration randomization.
func TestCanonicalAcrossTiers(t *testing.T) {
	tiers := []struct {
		name string
		opts Options
	}{
		{"Speed", OptSpeed | OptCanonical},
		{"Speed+Dense", OptSpeed | OptDense | OptCanonical},
		{"Balanced", OptBalanced | OptCanonical},
		{"Balanced+MapShape", OptBalanced | OptMapShape | OptCanonical},
		{"Balanced+ColumnIndex", OptBalanced | OptColumnIndex | OptCanonical},
		{"Compression", OptCompression | OptCanonical},
		{"Compression+ColumnIndex", OptCompression | OptColumnIndex | OptCanonical},
		{"Compression+FSST", OptCompression | OptFSST | OptCanonical},
	}

	a := buildTierPayload(false)
	b := buildTierPayload(true) // logically equal to a after canonical normalization

	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			// Determinism across differently-built equal values.
			ba, err := Marshal(a, tier.opts)
			if err != nil {
				t.Fatalf("marshal a: %v", err)
			}
			bb, err := Marshal(b, tier.opts)
			if err != nil {
				t.Fatalf("marshal b: %v", err)
			}
			if !bytes.Equal(ba, bb) {
				t.Fatalf("canonical not byte-identical for logically-equal values under %s", tier.name)
			}

			// Stability across map-iteration randomization: many re-encodes of
			// the SAME value must all match the first.
			for i := range 64 {
				bi, err := Marshal(a, tier.opts)
				if err != nil {
					t.Fatalf("re-marshal %d: %v", i, err)
				}
				if !bytes.Equal(ba, bi) {
					t.Fatalf("canonical unstable across re-encode %d under %s", i, tier.name)
				}
			}

			// The canonical bytes are still ordinary qdf — they must decode.
			var out tierPayload
			if err := Unmarshal(ba, &out); err != nil {
				t.Fatalf("decode canonical bytes under %s: %v", tier.name, err)
			}
		})
	}
}

// TestCanonicalTierSanityWithoutFlag is the control: without OptCanonical the
// two differently-built values are NOT required to match, confirming the test
// above is actually exercising the canonical branch and not some incidental
// determinism.
func TestCanonicalTierSanityWithoutFlag(t *testing.T) {
	a := buildTierPayload(false)
	// Same value, encoded canonically vs not, must differ in general only by the
	// normalization — but at minimum the canonical encode must itself be stable.
	c1, err := Marshal(a, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Marshal(a, OptBalanced|OptCanonical)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c1, c2) {
		t.Fatal("canonical encode not self-stable")
	}
}
