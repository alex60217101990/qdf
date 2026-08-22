package qdf

import (
	"math/rand"
	"reflect"
	"testing"
)

// This file builds confidence that Diff/Apply round-trip ARBITRARY values of a
// deeply-nested, every-shape type. The type intentionally uses only codec-
// lossless field kinds (int64/uint64/float64/string/[]byte/[N]byte/bool, nested
// structs, maps with string/int64 keys, slices, pointers) so the oracle can be a
// strict DeepEqual. Lossy kinds (int->int64 coercion, time.Time monotonic/zone
// stripping, interface{} dynamic-int widening) are documented codec behaviors
// and are exercised elsewhere; they are excluded here so a failure is provably a
// delta bug, not an inherent codec round-trip limitation.
//
// old and base are generated from the SAME seed: identical values backed by
// independent allocations, so Apply cannot alias old (no fragile deep-clone).

type dnLeaf struct {
	ID   int64
	U    uint64
	F    float64
	Name string
	Raw  []byte
	Arr  [5]byte
	On   bool
}

type dnMid struct {
	Vals   []int64
	Strs   []string
	Leaf   dnLeaf
	PLeaf  *dnLeaf
	Tags   map[string]int64
	ByID   map[int64]dnLeaf
	Flags  []bool
	Blobs  [][]byte
	PtrArr [3]*dnLeaf
}

type dnTop struct {
	Title    string
	Sections map[string]dnMid
	List     []*dnMid
	Matrix   [][]int64
	Top      dnMid
	PMid     *dnMid
	Nested   map[int64][]dnLeaf
}

// randValue fills rv (must be addressable/settable) with random data. depth
// bounds container sizes so generation stays fast; the fixed type bounds nesting.
func randValue(rv reflect.Value, r *rand.Rand, depth int) {
	switch rv.Kind() {
	case reflect.Bool:
		rv.SetBool(r.Intn(2) == 0)
	case reflect.Int64:
		rv.SetInt(int64(r.Uint64()))
	case reflect.Uint64:
		rv.SetUint(r.Uint64())
	case reflect.Float64:
		// finite, non-NaN: keep DeepEqual (==) consistent with the bit compare.
		rv.SetFloat(float64(r.Intn(1_000_000)) / float64(1+r.Intn(997)))
	case reflect.String:
		rv.SetString(randStr(r))
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			switch r.Intn(3) {
			case 0:
				rv.SetBytes(nil) // nil
			case 1:
				rv.SetBytes([]byte{}) // empty non-nil
			default:
				b := make([]byte, r.Intn(6))
				r.Read(b)
				rv.SetBytes(b)
			}
			return
		}
		switch r.Intn(4) {
		case 0:
			rv.Set(reflect.Zero(rv.Type())) // nil
		case 1:
			rv.Set(reflect.MakeSlice(rv.Type(), 0, 0)) // empty non-nil
		default:
			n := r.Intn(4)
			s := reflect.MakeSlice(rv.Type(), n, n)
			for i := range n {
				randValue(s.Index(i), r, depth+1)
			}
			rv.Set(s)
		}
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			for i := range rv.Len() {
				rv.Index(i).SetUint(uint64(r.Intn(256)))
			}
			return
		}
		for i := range rv.Len() {
			randValue(rv.Index(i), r, depth+1)
		}
	case reflect.Map:
		switch r.Intn(4) {
		case 0:
			rv.Set(reflect.Zero(rv.Type())) // nil
		case 1:
			rv.Set(reflect.MakeMap(rv.Type())) // empty non-nil
		default:
			m := reflect.MakeMap(rv.Type())
			n := r.Intn(4)
			kt, vt := rv.Type().Key(), rv.Type().Elem()
			for range n {
				k := reflect.New(kt).Elem()
				randValue(k, r, depth+1)
				v := reflect.New(vt).Elem()
				randValue(v, r, depth+1)
				m.SetMapIndex(k, v)
			}
			rv.Set(m)
		}
	case reflect.Pointer:
		if r.Intn(3) == 0 {
			rv.Set(reflect.Zero(rv.Type())) // nil
			return
		}
		p := reflect.New(rv.Type().Elem())
		randValue(p.Elem(), r, depth+1)
		rv.Set(p)
	case reflect.Struct:
		for _, field := range rv.Fields() {
			randValue(field, r, depth+1)
		}
	}
}

func randStr(r *rand.Rand) string {
	switch r.Intn(4) {
	case 0:
		return ""
	case 1:
		return []string{"a", "alpha", "x"}[r.Intn(3)] // intern-friendly repeats
	default:
		n := 1 + r.Intn(8)
		b := make([]byte, n)
		for i := range b {
			b[i] = byte('a' + r.Intn(26))
		}
		return string(b)
	}
}

func gen[T any](seed int64) T {
	var v T
	randValue(reflect.ValueOf(&v).Elem(), rand.New(rand.NewSource(seed)), 0)
	return v
}

func TestDiffApplyDeepNestArbitrary(t *testing.T) {
	tiers := []Options{OptBalanced, OptCompression, OptSpeed, OptDense | OptBalanced}
	for iter := range 4000 {
		seedOld := int64(iter)*2654435761 + 1
		seedNew := int64(iter)*40503 + 7
		old := gen[dnTop](seedOld)
		neu := gen[dnTop](seedNew)
		for _, opts := range tiers {
			patch, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatalf("iter %d opts=%v Diff: %v", iter, opts, err)
			}
			base := gen[dnTop](seedOld) // identical to old, independent allocations
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("iter %d opts=%v Apply: %v", iter, opts, err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("iter %d opts=%v: round-trip mismatch", iter, opts)
			}
			// No-op patch: Diff(x,x) applied to a copy of x yields x.
			selfPatch, err := Diff(neu, neu, opts)
			if err != nil {
				t.Fatalf("iter %d self-Diff: %v", iter, err)
			}
			self := gen[dnTop](seedNew)
			if err := Apply(&self, selfPatch); err != nil {
				t.Fatalf("iter %d self-Apply: %v", iter, err)
			}
			if !reflect.DeepEqual(self, neu) {
				t.Fatalf("iter %d opts=%v: no-op patch changed value", iter, opts)
			}
		}
	}
}

// FuzzDiffApplyDeepNest is the fuzz form of the above for continuous coverage.
func FuzzDiffApplyDeepNest(f *testing.F) {
	f.Add(int64(1), int64(2))
	f.Add(int64(42), int64(42))
	f.Add(int64(7), int64(99))
	f.Fuzz(func(t *testing.T, seedOld, seedNew int64) {
		old := gen[dnTop](seedOld)
		neu := gen[dnTop](seedNew)
		for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
			patch, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatalf("Diff opts=%v: %v", opts, err)
			}
			base := gen[dnTop](seedOld)
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("Apply opts=%v: %v", opts, err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("opts=%v: deep-nest round-trip mismatch", opts)
			}
		}
	})
}
