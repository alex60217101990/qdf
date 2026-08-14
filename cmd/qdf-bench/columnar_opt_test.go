package main

import (
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// The bit must be its own, not an alias of a neighbour.
// transposed reports whether a wire's root is one of the columnar containers.
// Which one depends on the SHAPE — a struct with a residual field takes the
// hybrid form (0xF7), a fully columnar-eligible one takes the pure form (0xEF) —
// so a test that pins a single tag fails on a legitimate shape change and blames
// the wrong thing.
func transposed(wire []byte) bool {
	return wire[5] == 0xF7 || wire[5] == 0xEF
}

func TestColumnarGeneratedBitIsDistinct(t *testing.T) {
	for _, o := range []struct {
		name string
		opts qdf.Options
	}{
		{"OptDense", qdf.OptDense},
		{"OptQPack", qdf.OptQPack},
		{"OptStringAlphabet", qdf.OptStringAlphabet},
		{"OptColumnIndex", qdf.OptColumnIndex},
		{"OptRANS", qdf.OptRANS},
	} {
		if qdf.OptColumnarGenerated&o.opts != 0 {
			t.Errorf("OptColumnarGenerated overlaps %s", o.name)
		}
	}
	if qdf.OptBalanced&qdf.OptColumnarGenerated != 0 {
		t.Error("OptBalanced includes OptColumnarGenerated — it must be opt-in")
	}
	if qdf.OptCompression&qdf.OptColumnarGenerated != 0 {
		t.Error("OptCompression includes OptColumnarGenerated — it must be opt-in")
	}
}

// With the bit, a slice of a generated type must produce EXACTLY the bytes its
// plain twin already produces — not merely fewer bytes. A size assertion would
// pass on a wire that was simply different, which is the failure mode that
// matters here: the whole safety argument is that this is the plain type's
// existing format, so every reader already handles it.
func TestColumnarGeneratedMatchesThePlainTwin(t *testing.T) {
	for _, n := range []int{16, 17, 64, 512} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			name string
			opts qdf.Options
		}{
			{"balanced", qdf.OptBalanced},
			{"compression", qdf.OptCompression},
		} {
			want, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := qdf.Marshal(gen, o.opts|qdf.OptColumnarGenerated)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("n=%d %s: generated wire is %d bytes, plain twin is %d — not the same format",
					n, o.name, len(got), len(want))
			}
		}
	}
}

// Without the bit nothing moves. Asserted against bytes produced in the same run,
// so an unrelated deliberate wire change elsewhere does not force an edit here.
func TestColumnarGeneratedIsOffByDefault(t *testing.T) {
	for _, n := range []int{16, 64, 512} {
		gen := make([]GenService, n)
		for k, s := range mkServices(n) {
			gen[k] = GenService(s)
		}
		a, err := qdf.Marshal(gen, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		b, err := qdf.Marshal(gen, qdf.OptBalanced|qdf.OptColumnarGenerated)
		if err != nil {
			t.Fatal(err)
		}
		if transposed(a) {
			t.Errorf("n=%d: the default already encodes columnar (0x%02x) — the bit is not a choice", n, a[5])
		}
		if !transposed(b) {
			t.Errorf("n=%d: with the bit the root tag is 0x%02x, neither columnar container — "+
				"the bit did nothing", n, b[5])
		}
	}
}

// Both directions must round-trip, with the bit and without it.
func TestColumnarGeneratedRoundTrips(t *testing.T) {
	for _, n := range []int{16, 64, 512} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			name string
			opts qdf.Options
		}{
			{"balanced+col", qdf.OptBalanced | qdf.OptColumnarGenerated},
			{"compression+col", qdf.OptCompression | qdf.OptColumnarGenerated},
		} {
			wire, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatal(err)
			}
			var intoGen []GenService
			if err := qdf.Unmarshal(wire, &intoGen); err != nil {
				t.Fatalf("n=%d %s into the generated type: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(intoGen, gen) {
				t.Fatalf("n=%d %s: value differs after a round trip into the generated type", n, o.name)
			}
			var intoPlain []Service
			if err := qdf.Unmarshal(wire, &intoPlain); err != nil {
				t.Fatalf("n=%d %s into the plain type: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(intoPlain, plain) {
				t.Fatalf("n=%d %s: value differs after a round trip into the plain type", n, o.name)
			}
		}
	}
}
