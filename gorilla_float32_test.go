package qdf

import (
	"math"
	"testing"
)

// (f32bitsEqual lives in float_edge_codec_test.go — bit-exact []float32 compare.)

// rawF32Wire is the size writePackedFloat32Slice would emit (tag+kind+uvarint+4n),
// excluding the shared stream header — the never-larger reference.
func rawF32Wire(n int) int { return 2 + uvarintLen(uint64(n)) + n*4 }

func smoothF32(n int) []float32 {
	s := make([]float32, n)
	v := float32(1000.0)
	for i := range s {
		v += 0.0009765625 // 1/1024: small, exponent-stable step
		s[i] = v
	}
	return s
}

// TestGorillaF32_RoundTrip checks bit-exact round-trip through the Gorilla
// float32 path (OptCompression enables it) across a spread of shapes, including
// the NaN / Inf / -0.0 / repeat edge cases the XOR codec must preserve.
func TestGorillaF32_RoundTrip(t *testing.T) {
	cases := map[string][]float32{
		"empty":        {},
		"single":       {3.14},
		"tiny":         {1, 2, 3, 4, 5},
		"smooth16":     smoothF32(16),
		"smooth1000":   smoothF32(1000),
		"all_equal":    {7, 7, 7, 7, 7, 7, 7, 7, 7, 7},
		"with_zero":    {0, 0, 1, 0, 2, 0, 0, 0, 3, 0},
		"neg_zero":     {math.Float32frombits(1 << 31), 0, math.Float32frombits(1 << 31)},
		"nan_inf":      {float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 1, 2, 3, 4, 5},
		"random_ish":   {1.1, -9999.5, 0.0001, 42, -0.5, 1e9, -1e-9, 3, 8, 13},
		"large_smooth": smoothF32(5000),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := Marshal(in, OptCompression)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out []float32
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			// Nil-vs-empty: an empty input round-trips to a zero-length slice.
			if len(in) == 0 {
				if len(out) != 0 {
					t.Fatalf("empty: got len %d", len(out))
				}
				return
			}
			if !f32bitsEqual(in, out) {
				t.Fatalf("round-trip mismatch:\n in =%v\n out=%v", in, out)
			}
		})
	}
}

// TestGorillaF32_ShrinksSmooth is the measure-first win check, isolating
// Gorilla's contribution: OptBalanced|OptGorillaFloat enables the float32
// Gorilla path WITHOUT the OptRANS/OptFSST final pass that OptCompression layers
// on top (which would otherwise frame the value region and hide the tag). On
// smooth data Gorilla must fire (tag present) and shrink the wire vs both raw
// and the no-Gorilla baseline.
func TestGorillaF32_ShrinksSmooth(t *testing.T) {
	const gorOnly = OptBalanced | OptGorillaFloat
	in := smoothF32(2000)

	b, err := Marshal(in, gorOnly)
	if err != nil {
		t.Fatal(err)
	}
	base, err := Marshal(in, OptBalanced) // no Gorilla → raw float32
	if err != nil {
		t.Fatal(err)
	}
	raw := rawF32Wire(len(in))
	got := len(b) - 5 // drop the 5-byte stream header to compare against raw body
	if got >= raw {
		t.Fatalf("Gorilla did not shrink smooth float32: got %d >= raw %d", got, raw)
	}
	if len(b) >= len(base) {
		t.Fatalf("Gorilla not smaller than no-Gorilla baseline: %d >= %d", len(b), len(base))
	}
	t.Logf("smooth float32 n=%d: gorilla=%d raw=%d baseline=%d (%.1f%% of raw)",
		len(in), got, raw, len(base), 100*float64(got)/float64(raw))
	// Confirm the Gorilla tag fired (value region starts right after the 5-byte
	// header when no entropy pass is layered on) — not a silent raw fallback.
	if len(b) < 7 || b[5] != tagPackGorilla || b[6] != qpackKindFloat32 {
		t.Fatalf("expected tagPackGorilla/float32 at value start, got %#x %#x", b[5], b[6])
	}
}

// TestGorillaF32_NeverLarger checks the never-larger gate: high-entropy float32
// must NOT inflate beyond the raw encoding (Gorilla rolls back to raw).
func TestGorillaF32_NeverLarger(t *testing.T) {
	// Pseudo-random-ish high-entropy float32 (no fixed seed needed; the values
	// are deterministic and adversarial to XOR delta).
	in := make([]float32, 1000)
	x := uint32(0x9e3779b9)
	for i := range in {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		in[i] = math.Float32frombits(x)
	}
	// Isolate the Gorilla gate (no rANS/FSST pass on top, which could mask it).
	b, err := Marshal(in, OptBalanced|OptGorillaFloat)
	if err != nil {
		t.Fatal(err)
	}
	got := len(b) - 5
	raw := rawF32Wire(len(in))
	if got > raw {
		t.Fatalf("never-larger violated: high-entropy gorilla=%d > raw=%d", got, raw)
	}
	// And it still round-trips.
	var out []float32
	if err := Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !f32bitsEqual(in, out) {
		t.Fatal("high-entropy round-trip mismatch")
	}
}
