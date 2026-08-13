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
