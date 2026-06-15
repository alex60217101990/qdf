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
		Tags:  map[string]int{},
		Arr:   [4]byte{byte(r.Intn(256)), byte(r.Intn(256)), 0, 0},
	}
	for i := 0; i < r.Intn(4); i++ {
		rec.Tags[[]string{"k1", "k2", "k3"}[r.Intn(3)]] = r.Intn(50)
	}
	for i := 0; i < r.Intn(5); i++ {
		rec.Items = append(rec.Items, fuzzInner{
			A: int32(r.Intn(100)),
			B: []string{"x", "y", "z"}[r.Intn(3)],
			C: []byte{byte(r.Intn(256))},
		})
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
	c.Tags = map[string]int{}
	for k, v := range r.Tags {
		c.Tags[k] = v
	}
	c.Items = append([]fuzzInner(nil), r.Items...)
	for i := range c.Items {
		c.Items[i].C = append([]byte(nil), r.Items[i].C...)
	}
	if r.Opt != nil {
		o := *r.Opt
		o.C = append([]byte(nil), r.Opt.C...)
		c.Opt = &o
	}
	return c
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
