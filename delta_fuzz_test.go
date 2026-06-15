package qdf

import (
	"math/rand"
	"reflect"
	"testing"
)

type fuzzInner struct {
	A int32
	B string
	C []byte
}

type fuzzRec struct {
	ID    int
	Name  string
	Score float64
	Tags  map[string]int
	Items []fuzzInner
	Opt   *fuzzInner
	Arr   [4]byte
}

func randRec(r *rand.Rand) fuzzRec {
	rec := fuzzRec{
		ID:    r.Intn(1000),
		Name:  []string{"", "a", "alpha", "beta"}[r.Intn(4)],
		Score: float64(r.Intn(100)) / 7,
		Arr:   [4]byte{byte(r.Intn(256)), byte(r.Intn(256)), 0, 0},
	}
	switch r.Intn(3) {
	case 0:
		rec.Tags = nil
	case 1:
		rec.Tags = map[string]int{} // empty non-nil
	default:
		rec.Tags = map[string]int{}
		for i := 0; i < 1+r.Intn(3); i++ {
			rec.Tags[[]string{"k1", "k2", "k3"}[r.Intn(3)]] = r.Intn(50)
		}
	}
	switch r.Intn(3) {
	case 0:
		rec.Items = nil
	case 1:
		rec.Items = []fuzzInner{} // empty non-nil
	default:
		for i := 0; i < 1+r.Intn(4); i++ {
			rec.Items = append(rec.Items, fuzzInner{
				A: int32(r.Intn(100)),
				B: []string{"x", "y", "z"}[r.Intn(3)],
				C: []byte{byte(r.Intn(256))},
			})
		}
	}
	if r.Intn(2) == 0 {
		rec.Opt = &fuzzInner{A: int32(r.Intn(10)), B: "opt"}
	}
	return rec
}

// deepCopyRec produces an independent copy so Apply cannot alias old's backing
// arrays/maps/pointers.
func deepCopyRec(r fuzzRec) fuzzRec {
	c := r
	if r.Tags != nil {
		c.Tags = map[string]int{}
		for k, v := range r.Tags {
			c.Tags[k] = v
		}
	}
	if r.Items != nil {
		c.Items = make([]fuzzInner, len(r.Items))
		copy(c.Items, r.Items)
		for i := range c.Items {
			if r.Items[i].C != nil {
				c.Items[i].C = append([]byte(nil), r.Items[i].C...)
			}
		}
	}
	if r.Opt != nil {
		o := *r.Opt
		if r.Opt.C != nil {
			o.C = append([]byte(nil), r.Opt.C...)
		}
		c.Opt = &o
	}
	return c
}

func FuzzApplyHostile(f *testing.F) {
	type T struct {
		A int
		B string
		C []int
		M map[string]int
		S []string
	}
	good, _ := Diff(
		T{A: 1},
		T{A: 2, B: "x", C: []int{1, 2}, M: map[string]int{"k": 1}, S: []string{"p"}},
		OptBalanced,
	)
	f.Add(good)
	f.Add([]byte("QDP\x01"))
	f.Add([]byte{})
	f.Add([]byte("QDP\x01\x04")) // dense+rans-ish flag byte, truncated
	f.Fuzz(func(t *testing.T, patch []byte) {
		var base T
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on hostile patch (len=%d): %v", len(patch), r)
			}
		}()
		_ = Apply(&base, patch) // must return an error or nil, never panic/OOM
	})
}

func FuzzDiffApplyOracle(f *testing.F) {
	f.Add(int64(1), int64(2))
	f.Add(int64(42), int64(42)) // identical seed → identical values (no-op patch)
	f.Add(int64(7), int64(99))
	f.Fuzz(func(t *testing.T, seedOld, seedNew int64) {
		old := randRec(rand.New(rand.NewSource(seedOld)))
		neu := randRec(rand.New(rand.NewSource(seedNew)))
		for _, opts := range []Options{OptBalanced, OptCompression, OptSpeed} {
			patch, err := Diff(old, neu, opts)
			if err != nil {
				t.Fatalf("Diff opts=%v: %v", opts, err)
			}
			base := deepCopyRec(old)
			if err := Apply(&base, patch); err != nil {
				t.Fatalf("Apply opts=%v: %v", opts, err)
			}
			if !reflect.DeepEqual(base, neu) {
				t.Fatalf("oracle mismatch opts=%v\n old=%+v\n new=%+v\n got=%+v",
					opts, old, neu, base)
			}
		}
	})
}
