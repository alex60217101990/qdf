package qdf

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

// Options matrix tests: every interesting combination of the codec
// bits must round-trip identically — Marshal then Unmarshal yields a
// deep-equal value to the input — across a representative set of
// payload shapes. The decoder is wire-self-describing, so the same
// Unmarshal entry point must handle every emitted variant.

type optsAddr struct {
	Street string `qdf:"street"`
	City   string `qdf:"city"`
	Zip    string `qdf:"zip"`
}

type optsEvent struct {
	Service string   `qdf:"service"`
	Region  string   `qdf:"region"`
	Level   string   `qdf:"level"`
	Host    string   `qdf:"host"`
	Status  int      `qdf:"status"`
	Tags    []string `qdf:"tags"`
}

type optsCombo struct {
	Header  optsEvent         `qdf:"header"`
	Address optsAddr          `qdf:"address"`
	Repeats []string          `qdf:"repeats"`
	Counts  []int64           `qdf:"counts"`
	Floats  []float64         `qdf:"floats"`
	Flags   []bool            `qdf:"flags"`
	Bytes   []byte            `qdf:"bytes"`
	Map     map[string]string `qdf:"map"`
}

// optionsMatrix is a hand-picked set of Options bit combinations
// covering both the simple presets and every individual bit toggled
// on top of OptDense. Combinations that disable OptDense automatically
// disable the codecs that depend on it; the gating logic must handle
// that without misencoding the stream.
func optionsMatrix() []Options {
	dense := OptDense
	return []Options{
		OptSpeed,
		OptQPack,
		dense,
		dense | OptQPack,
		dense | OptShapeIntern,
		dense | OptPairPred,
		dense | OptMTF,
		dense | OptQPack | OptShapeIntern,
		dense | OptQPack | OptPairPred,
		dense | OptQPack | OptMTF,
		dense | OptShapeIntern | OptPairPred,
		dense | OptShapeIntern | OptMTF,
		dense | OptPairPred | OptMTF,
		dense | OptQPack | OptShapeIntern | OptPairPred,
		dense | OptQPack | OptShapeIntern | OptMTF,
		dense | OptQPack | OptPairPred | OptMTF,
		dense | OptShapeIntern | OptPairPred | OptMTF,
		OptBalanced,
		OptCompression,
	}
}

type optsDeep struct {
	V int       `qdf:"v"`
	N *optsDeep `qdf:"n,omitempty"`
}

type optsTimed struct {
	When time.Time `qdf:"when"`
	Note string    `qdf:"note"`
}

type optsNumbers struct {
	Monotonic []int64   `qdf:"monotonic"`
	Wild      []int64   `qdf:"wild"`
	Floats    []float64 `qdf:"floats"`
	F32       []float32 `qdf:"f32"`
	Bools     []bool    `qdf:"bools"`
	Uints     []uint64  `qdf:"uints"`
	Bytes     []byte    `qdf:"bytes"`
}

func optsFixtures() []struct {
	name string
	gen  func() any
} {
	return []struct {
		name string
		gen  func() any
	}{
		{
			name: "addr_single",
			gen:  func() any { return optsAddr{Street: "Vilnius St 1", City: "Klaipeda", Zip: "LT-91300"} },
		},
		{
			name: "addr_array_5",
			gen: func() any {
				out := make([]optsAddr, 5)
				for i := range out {
					out[i] = optsAddr{Street: "S", City: "C", Zip: "Z"}
				}
				return out
			},
		},
		{
			name: "event_repeating_strings",
			gen: func() any {
				out := make([]optsEvent, 8)
				for i := range out {
					out[i] = optsEvent{
						Service: "billing",
						Region:  "eu-west-1",
						Level:   "info",
						Host:    fmt.Sprintf("ip-10-0-%d", i),
						Status:  200 + i%3,
						Tags:    []string{"prod", "v3"},
					}
				}
				return out
			},
		},
		{
			name: "combo_mixed",
			gen: func() any {
				return optsCombo{
					Header: optsEvent{
						Service: "ingest", Region: "eu-west-1", Level: "warn",
						Host: "ip-10-0-42", Status: 503, Tags: []string{"prod"},
					},
					Address: optsAddr{Street: "S", City: "C", Zip: "Z"},
					Repeats: []string{"alpha-token", "alpha-token", "alpha-token", "beta-token"},
					Counts:  []int64{1, 2, 3, 4, 5, 6, 7, 8},
					Floats:  []float64{1.5, 2.5, 3.5, 4.5, 5.5},
					Flags:   []bool{true, false, true, false, true, true, false},
					Bytes:   []byte("hello, world"),
					Map:     map[string]string{"a": "1", "b": "2", "c": "3"},
				}
			},
		},
		{
			name: "scalars",
			gen: func() any {
				return struct {
					I int     `qdf:"i"`
					U uint64  `qdf:"u"`
					F float64 `qdf:"f"`
					B bool    `qdf:"b"`
					S string  `qdf:"s"`
				}{42, 0xdeadbeef, 3.14, true, "hello"}
			},
		},
		{
			name: "deep_nested_8",
			gen: func() any {
				root := &optsDeep{V: 0}
				cur := root
				for i := 1; i < 8; i++ {
					cur.N = &optsDeep{V: i}
					cur = cur.N
				}
				return *root
			},
		},
		{
			name: "empty_struct",
			gen:  func() any { return optsAddr{} },
		},
		{
			name: "empty_slices",
			gen: func() any {
				// Non-nil empty slices: encoder emits arr0/bin0 which
				// the decoder restores as empty (non-nil) slices. A nil
				// slice would round-trip as empty, which is DeepEqual-
				// false, but documents the intentional Go-stdlib-like
				// behavior and is not exercised here.
				return optsNumbers{
					Monotonic: []int64{},
					Wild:      []int64{},
					Floats:    []float64{},
					F32:       []float32{},
					Bools:     []bool{},
					Uints:     []uint64{},
					Bytes:     []byte{},
				}
			},
		},
		{
			name: "numeric_edge",
			gen: func() any {
				return optsNumbers{
					Monotonic: []int64{1_700_000_000, 1_700_000_001, 1_700_000_002, 1_700_000_003, 1_700_000_004},
					Wild:      []int64{math.MinInt64, -1, 0, 1, math.MaxInt64},
					Floats:    []float64{0, -0, math.Pi, math.MaxFloat64, math.SmallestNonzeroFloat64},
					F32:       []float32{0, float32(math.Pi), float32(math.MaxFloat32)},
					Bools:     []bool{true, false, true, true, false, false, true, false, true},
					Uints:     []uint64{0, 1, 127, 128, 16383, 16384, math.MaxUint32, math.MaxUint64},
					Bytes:     []byte("the quick brown fox"),
				}
			},
		},
		{
			name: "long_string_repeats",
			gen: func() any {
				// 200x identical 64-byte string — triggers state-ref
				// repeat + MTF + Pair on every entry.
				const s = "the-quick-brown-fox-jumps-over-the-lazy-dog-1234567890-ABCDEFGH"
				out := make([]string, 200)
				for i := range out {
					out[i] = s
				}
				return out
			},
		},
	}
}

// roundTripWith encodes v with Marshal(opts), decodes into a fresh
// value of the same type, and returns the decoded value plus the wire
// bytes for diagnostics. The fresh value is obtained via reflect.New
// so the test stays type-agnostic.
func roundTripWith(t *testing.T, v any, opts Options) (any, []byte) {
	t.Helper()
	b, err := Marshal(v, opts)
	if err != nil {
		t.Fatalf("Marshal(opts=%b): %v", opts, err)
	}
	outPtr := reflect.New(reflect.TypeOf(v))
	if err := Unmarshal(b, outPtr.Interface()); err != nil {
		t.Fatalf("Unmarshal(opts=%b): %v wire=%x", opts, err, b)
	}
	return outPtr.Elem().Interface(), b
}

func TestOptions_MatrixRoundTrip(t *testing.T) {
	for _, fx := range optsFixtures() {
		for _, opts := range optionsMatrix() {
			name := fmt.Sprintf("%s/opts=%05b", fx.name, opts)
			t.Run(name, func(t *testing.T) {
				in := fx.gen()
				out, wire := roundTripWith(t, in, opts)
				if !reflect.DeepEqual(in, out) {
					t.Fatalf("round-trip mismatch\n want=%+v\n  got=%+v\n wire=%x", in, out, wire)
				}
			})
		}
	}
}

// TestOptions_WireSizeRanking — soft sanity that turning on more
// codecs does not grow the wire for a payload that is friendly to all
// of them. Picks the repeated-event corpus where every Dense codec
// has at least one hit.
func TestOptions_WireSizeRanking(t *testing.T) {
	fixture := optsFixtures()[2].gen() // event_repeating_strings

	encode := func(opts Options) int {
		b, err := Marshal(fixture, opts)
		if err != nil {
			t.Fatal(err)
		}
		return len(b)
	}

	speed := encode(OptSpeed)
	dense := encode(OptDense)
	balanced := encode(OptBalanced)
	compression := encode(OptCompression)

	t.Logf("OptSpeed=%d  OptDense=%d  OptBalanced=%d  OptCompression=%d",
		speed, dense, balanced, compression)

	// OptDense intern table alone must shrink wire vs OptSpeed.
	if dense >= speed {
		t.Fatalf("OptDense (%d) must be < OptSpeed (%d) on repetitive corpus", dense, speed)
	}
	// OptBalanced (all codecs) must be ≤ OptDense (intern only).
	if balanced > dense {
		t.Fatalf("OptBalanced (%d) must be ≤ OptDense (%d)", balanced, dense)
	}
	// OptCompression adds OptGorillaFloat on top of OptBalanced. There
	// are no float slices in this fixture so Gorilla never fires and the
	// wires match byte for byte; the check guards against the heavy
	// bundle ever growing the wire over the balanced default.
	if compression > balanced {
		t.Fatalf("OptCompression (%d) must be ≤ OptBalanced (%d) on a repetitive corpus",
			compression, balanced)
	}
}

// TestOptions_BundleAliases verifies that the convenience bundles
// (OptSpeed / OptBalanced / OptCompression) produce the wire that
// their composition implies. Catches drift if a bundle constant is
// accidentally rewired without an explicit bit change.
func TestOptions_BundleAliases(t *testing.T) {
	for _, fx := range optsFixtures() {
		if fx.name == "combo_mixed" {
			continue // map iteration is randomized
		}
		in := fx.gen()
		t.Run(fx.name+"/Speed=zero", func(t *testing.T) {
			a, err := Marshal(in, OptSpeed)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Marshal(in, 0)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("OptSpeed differs from Options(0)")
			}
		})
		t.Run(fx.name+"/Compression≤Balanced", func(t *testing.T) {
			// OptCompression = OptBalanced | OptGorillaFloat. Gorilla
			// only fires on float slices and can only shrink the wire,
			// never grow it. So OptCompression must never be larger
			// than OptBalanced (on fixtures without float slices the
			// wires match byte for byte). A regression here means a
			// heavy codec leaked extra bytes under OptCompression.
			a, err := Marshal(in, OptCompression)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Marshal(in, OptBalanced)
			if err != nil {
				t.Fatal(err)
			}
			if len(a) > len(b) {
				t.Fatalf("OptCompression (%d) larger than OptBalanced (%d) — heavy codec grew the wire", len(a), len(b))
			}
		})
		t.Run(fx.name+"/Balanced=explicit", func(t *testing.T) {
			a, err := Marshal(in, OptBalanced)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Marshal(in, OptDense|OptQPack|OptShapeIntern|OptPairPred|OptMTF)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("OptBalanced bundle differs from its explicit composition")
			}
		})
	}
}

// TestOptions_GorillaFiresUnderCompression pins the contract that
// OptCompression diverges from OptBalanced by exactly OptGorillaFloat
// — when given a smooth float series, the OptCompression wire must
// be materially smaller because Gorilla fires; under OptBalanced the
// same payload stays on raw-LE. A drift here (Gorilla leaking into
// OptBalanced, or the gate going dark under OptCompression) would
// silently break either the speed contract or the size contract.
func TestOptions_GorillaFiresUnderCompression(t *testing.T) {
	// Quantised smooth series: 0.7 repeat rate + 0.1 steps. Same
	// shape as the bench fixture so the wire delta tracks the
	// published numbers.
	const n = 1024
	floats := make([]float64, n)
	value := 50.0
	for i := range floats {
		if i > 0 && i%3 != 0 { // ~67 % repeats — Gorilla-friendly
			floats[i] = value
			continue
		}
		if i&1 == 0 {
			value += 0.1
		} else {
			value -= 0.1
		}
		floats[i] = value
	}
	in := struct {
		F []float64 `qdf:"f"`
	}{F: floats}

	bBalanced, err := Marshal(in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	bCompress, err := Marshal(in, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	if len(bCompress) >= len(bBalanced)/2 {
		t.Fatalf("OptCompression did not engage Gorilla on smooth floats: balanced=%d compress=%d", len(bBalanced), len(bCompress))
	}
	// Sanity: both decode to the same value.
	var outB, outC struct {
		F []float64 `qdf:"f"`
	}
	if err := Unmarshal(bBalanced, &outB); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(bCompress, &outC); err != nil {
		t.Fatal(err)
	}
	if len(outB.F) != n || len(outC.F) != n {
		t.Fatalf("decode length mismatch: balanced=%d compress=%d want=%d", len(outB.F), len(outC.F), n)
	}
	for i := range outB.F {
		if outB.F[i] != outC.F[i] {
			t.Fatalf("decoded floats diverged at %d: balanced=%v compress=%v", i, outB.F[i], outC.F[i])
		}
	}
}

// TestOptions_TWithMatchesWith pins that MarshalT and Marshal
// produce the same wire bytes for the same options on the same value.
// Use a fixture with no Go map (whose iteration order is randomized
// and would mask the real equivalence we want to test).
func TestOptions_TWithMatchesWith(t *testing.T) {
	in := optsFixtures()[2].gen().([]optsEvent) // event_repeating_strings

	for _, opts := range optionsMatrix() {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			a, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			b, err := MarshalT(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			if string(a) != string(b) {
				t.Fatalf("generic vs non-generic wire diverged\n   with=%x\n  twith=%x", a, b)
			}
		})
	}
}

// TestOptions_Has unit-tests the bit-mask accessor. Cheap, but
// catches an accidental sign-flip or rename of the constants.
func TestOptions_Has(t *testing.T) {
	all := OptDense | OptQPack | OptShapeIntern | OptPairPred | OptMTF
	cases := []struct {
		opts Options
		bit  Options
		want bool
	}{
		{OptSpeed, OptDense, false},
		{OptSpeed, OptQPack, false},
		{OptDense, OptDense, true},
		{OptDense, OptQPack, false},
		{OptBalanced, OptShapeIntern, true},
		{OptBalanced, OptPairPred, true},
		{OptCompression, OptMTF, true},
		{all, OptDense | OptMTF, true},        // combined bit check
		{OptDense, OptDense | OptQPack, true}, // partial overlap still hits
	}
	for _, c := range cases {
		if got := c.opts.Has(c.bit); got != c.want {
			t.Fatalf("Options(%b).Has(%b)=%v want %v", c.opts, c.bit, got, c.want)
		}
	}
}

// TestOptions_AppendParity pins that AppendMarshal with an empty
// destination yields the same bytes as Marshal for every option
// combination. Skips fixtures containing Go maps because randomized
// iteration order would produce false negatives that mask real bugs.
func TestOptions_AppendParity(t *testing.T) {
	for _, fx := range optsFixtures() {
		if fx.name == "combo_mixed" {
			continue // contains map[string]string — see TestOptions_CrossEncodeUnmarshal for that fixture
		}
		in := fx.gen()
		for _, opts := range optionsMatrix() {
			name := fmt.Sprintf("%s/opts=%05b", fx.name, opts)
			t.Run(name, func(t *testing.T) {
				a, err := Marshal(in, opts)
				if err != nil {
					t.Fatal(err)
				}
				b, err := AppendMarshal(nil, in, opts)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(a, b) {
					t.Fatalf("Append wire diverged\n   marshal=%x\n    append=%x", a, b)
				}
			})
		}
	}
}

// TestOptions_AppendReusesBuffer verifies AppendMarshal appends
// to the caller's slice without dropping prior content. Defensive:
// the entry point is easy to mis-implement by truncating dst.
func TestOptions_AppendReusesBuffer(t *testing.T) {
	in := optsFixtures()[3].gen() // combo_mixed
	dst := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	pre := len(dst)
	out, err := AppendMarshal(dst, in, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) <= pre {
		t.Fatalf("AppendMarshal did not append: pre=%d post=%d", pre, len(out))
	}
	if !bytes.Equal(out[:pre], dst[:pre]) {
		t.Fatalf("prefix changed: want=%x got=%x", dst[:pre], out[:pre])
	}
}

// TestOptions_AppendTWithParity is the generic counterpart of
// TestOptions_AppendParity for a typed fixture. Uses a map-free
// fixture so random iteration order does not flake the byte compare.
func TestOptions_AppendTWithParity(t *testing.T) {
	in := optsFixtures()[2].gen().([]optsEvent) // event_repeating_strings
	for _, opts := range optionsMatrix() {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			a, err := MarshalT(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			b, err := AppendMarshalT(nil, in, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("TWith Append wire diverged\n   marshal=%x\n    append=%x", a, b)
			}
		})
	}
}

// TestOptions_CrossEncodeUnmarshal verifies the contract claimed in
// the README: every Options combination on the encode side decodes
// through the single Unmarshal entry point on the decode side. The
// decoder is wire-self-describing; the user picks an encode profile,
// the receiver does not need to know which one.
func TestOptions_CrossEncodeUnmarshal(t *testing.T) {
	in := optsFixtures()[3].gen().(optsCombo)
	for _, opts := range optionsMatrix() {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			b, err := Marshal(in, opts)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var out optsCombo
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("Unmarshal: %v  wire=%x", err, b)
			}
			if !reflect.DeepEqual(in, out) {
				t.Fatalf("decode mismatch\n want=%+v\n  got=%+v", in, out)
			}
		})
	}
}

// TestOptions_StreamEncoderWithRoundTrip exercises every options
// combination through the NewStreamEncoderWith path so a Dense
// stream that drops MTF or Shape (or any other bit) still survives
// a full encode-then-decode through StreamDecoder.
func TestOptions_StreamEncoderWithRoundTrip(t *testing.T) {
	type item struct {
		Service string   `qdf:"service"`
		Region  string   `qdf:"region"`
		Status  int      `qdf:"status"`
		Tags    []string `qdf:"tags"`
	}
	items := make([]item, 12)
	for i := range items {
		items[i] = item{
			Service: "billing",
			Region:  "eu-west-1",
			Status:  200 + i%3,
			Tags:    []string{"prod", "v3"},
		}
	}
	for _, opts := range optionsMatrix() {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			var w bytes.Buffer
			enc := NewStreamEncoderWith(&w, opts)
			for _, it := range items {
				if err := enc.Encode(it); err != nil {
					t.Fatalf("encode: %v", err)
				}
			}
			if err := enc.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			dec := NewStreamDecoder(&w)
			for i, want := range items {
				var got item
				if err := dec.Decode(&got); err != nil {
					t.Fatalf("[%d] decode: %v", i, err)
				}
				if !reflect.DeepEqual(want, got) {
					t.Fatalf("[%d] mismatch\n want=%+v\n  got=%+v", i, want, got)
				}
			}
		})
	}
}

// TestOptions_ZeroOpts pins the OptSpeed=0 contract: no codecs, Fast
// mode, no intern emission even on heavily repeating input.
func TestOptions_ZeroOpts(t *testing.T) {
	in := optsFixtures()[9].gen() // long_string_repeats
	out, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	// No state-table tags must appear in Fast wire.
	for _, tag := range []byte{
		tagInternStr, tagInternBin, tagStateRef, tagStateRepeat,
		tagStateMTF, tagStatePair, tagMapShape,
	} {
		if bytes.IndexByte(out, tag) >= 0 {
			t.Fatalf("OptSpeed wire contains Dense tag 0x%02X: %x", tag, out[:64])
		}
	}
	var decoded []string
	if err := Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, decoded) {
		t.Fatal("OptSpeed roundtrip mismatch")
	}
}

// TestOptions_TimeRoundTrip covers time.Time across every options
// combination. DeepEqual is unreliable on time.Time (the loc pointer
// can differ between UTC and Local while the instant is identical),
// so we compare via Time.Equal and the Note string instead.
func TestOptions_TimeRoundTrip(t *testing.T) {
	in := optsTimed{When: time.Unix(1_700_000_000, 12345).UTC(), Note: "boot"}
	for _, opts := range optionsMatrix() {
		t.Run(fmt.Sprintf("opts=%05b", opts), func(t *testing.T) {
			b, err := Marshal(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			var out optsTimed
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("Unmarshal: %v wire=%x", err, b)
			}
			if !in.When.Equal(out.When) || in.Note != out.Note {
				t.Fatalf("time mismatch:\n want=%+v\n  got=%+v", in, out)
			}
		})
	}
}

// TestOptions_DenseAloneSavesBytesOnRepeats — without QPack or any
// of the predictor bits, OptDense by itself must still shrink wire
// over OptSpeed on a string-heavy payload, courtesy of the intern
// table and the always-on Markov-0 repeat collapse.
func TestOptions_DenseAloneSavesBytesOnRepeats(t *testing.T) {
	in := optsFixtures()[9].gen() // long_string_repeats
	fast, err := Marshal(in, OptSpeed)
	if err != nil {
		t.Fatal(err)
	}
	dense, err := Marshal(in, OptDense)
	if err != nil {
		t.Fatal(err)
	}
	if len(dense) >= len(fast) {
		t.Fatalf("OptDense alone failed to shrink wire on repeats: fast=%d dense=%d",
			len(fast), len(dense))
	}
}
