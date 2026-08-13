package main

import (
	"reflect"
	"testing"

	"github.com/alex60217101990/qdf"
)

// A wire written by the reflect encoder must decode into the generated twin.
//
// It does not, from sixteen elements up: reflect switches to the hybrid
// columnar container there, and a generated decoder accepts only the ONE column
// split baked into it at generation time —
//
//	if !slices.Equal(names, qdfHybNames_X) || !slices.Equal(kinds, qdfHybKinds_X) {
//	    return qdf.ErrTypeMismatch
//	}
//
// — while reflect chooses its split by probing the data. The refusal is correct
// in itself: reading on would produce wrong values. What is missing is a way to
// fall back.
//
// Service and GenService are defined types over one another with identical
// layouts, so this is the same value by construction, not two similar ones. The
// producer here is a plain library user with no generated code; the consumer
// only has the generated type. Neither is doing anything unusual.
func TestReflectWireDecodesIntoGeneratedType(t *testing.T) {
	for _, n := range []int{2, 15, 16, 17, 64, 512} {
		plain := mkServices(n)
		for _, o := range []struct {
			name string
			opts qdf.Options
		}{
			{"speed", qdf.OptSpeed},
			{"balanced", qdf.OptBalanced},
			{"compression", qdf.OptCompression},
		} {
			wire, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			var got []GenService
			if err := qdf.Unmarshal(wire, &got); err != nil {
				t.Errorf("n=%-3d %-12s root=0x%02x: %v", n, o.name, wire[5], err)
				continue
			}
			want := make([]GenService, n)
			for k := range plain {
				want[k] = GenService(plain[k])
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("n=%d %s: decoded value differs", n, o.name)
			}
		}
	}
}

// The reverse direction already works and must keep working: a generated
// producer's wire read by a consumer that only has the plain type.
func TestGeneratedWireDecodesIntoPlainType(t *testing.T) {
	for _, n := range []int{2, 16, 64} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			name string
			opts qdf.Options
		}{
			{"speed", qdf.OptSpeed},
			{"balanced", qdf.OptBalanced},
			{"compression", qdf.OptCompression},
		} {
			wire, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			var got []Service
			if err := qdf.Unmarshal(wire, &got); err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			if !reflect.DeepEqual(got, plain) {
				t.Fatalf("n=%d %s: decoded value differs", n, o.name)
			}
		}
	}
}

// Encoding must be untouched by the decode fallback.
//
// The bytes are compared against bytes produced in the same run rather than
// against a stored digest: the property is that nothing here reaches the
// encoder, and a stored digest would also fail for an unrelated deliberate wire
// change and teach the next reader to update it without thinking.
//
// Re-encoding AFTER a fallback decode is the part that matters. The fallback
// builds a columnar plan and caches it; if that plan ever leaked into the
// encode path, the first encode of a process would differ from the second, and
// only an ordering like this one would notice.
func TestDecodeFallbackLeavesEncodingAlone(t *testing.T) {
	// Counts the rows where the plain twin actually produced a columnar frame.
	// The divergence check below compares the two root tags, and a comparison
	// only means something while one side really is columnar — without this the
	// whole assertion would go quiet the day reflect stopped transposing, and
	// the test would keep passing.
	columnarRows := 0
	for _, n := range []int{2, 15, 16, 17, 64, 512} {
		plain := mkServices(n)
		gen := make([]GenService, n)
		for k := range plain {
			gen[k] = GenService(plain[k])
		}
		for _, o := range []struct {
			name string
			opts qdf.Options
		}{
			{"speed", qdf.OptSpeed},
			{"balanced", qdf.OptBalanced},
			{"compression", qdf.OptCompression},
			{"balanced+rans", qdf.OptBalanced | qdf.OptRANS},
			{"balanced+colindex", qdf.OptBalanced | qdf.OptColumnIndex},
		} {
			a, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			var round []GenService
			if err := qdf.Unmarshal(a, &round); err != nil {
				t.Fatalf("n=%d %s: decode: %v", n, o.name, err)
			}
			b, err := qdf.Marshal(plain, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s: %v", n, o.name, err)
			}
			if string(a) != string(b) {
				t.Errorf("n=%d %s: re-encoding after a fallback decode changed %d bytes to %d",
					n, o.name, len(a), len(b))
			}
			g1, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s gen: %v", n, o.name, err)
			}
			g2, err := qdf.Marshal(gen, o.opts)
			if err != nil {
				t.Fatalf("n=%d %s gen: %v", n, o.name, err)
			}
			if string(g1) != string(g2) {
				t.Errorf("n=%d %s: the generated type's wire is not stable", n, o.name)
			}
			// The generated type must still take its own codec rather than the
			// columnar container its plain twin takes. If the fallback had
			// reached the encoder, this root tag would follow the reflect one.
			if n >= 16 && a[5] == 0xF7 {
				columnarRows++
				if g1[5] == 0xF7 {
					t.Errorf("n=%d %s: the generated type now encodes columnar (0xF7) like its "+
						"plain twin — the fallback reached the encoder", n, o.name)
				}
			}
		}
	}
	if columnarRows == 0 {
		t.Fatal("no row produced a columnar frame from the plain type, so the divergence " +
			"check above never ran and this test proved nothing about the encoder")
	}
}
