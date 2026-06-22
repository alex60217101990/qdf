package qdf

import (
	"errors"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// bnRegTimeStamped exercises the fields-less-struct clone path (time.Time is
// entirely unexported): a field-by-field deepClone would zero it and the
// baseFP check would reject every patch. See deepClone's struct handling.
type bnRegTimeStamped struct {
	ID   string
	TS   time.Time
	Tags []string
}

func TestBaselineTimeFieldRoundTrip(t *testing.T) {
	r := NewBaselineRegistry[bnRegTimeStamped]()
	s0 := bnRegTimeStamped{ID: "a", TS: time.Unix(1700000000, 12345).UTC(), Tags: []string{"x"}}
	r.Register(&s0)
	s1want := bnRegTimeStamped{ID: "b", TS: s0.TS.Add(90 * time.Minute), Tags: []string{"x", "y"}}
	patch, err := Diff(s0, s1want, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	s1, err := r.Apply(patch)
	if err != nil {
		t.Fatalf("Apply with time.Time field: %v", err)
	}
	if !reflect.DeepEqual(*s1, s1want) {
		t.Fatalf("reconstructed = %+v, want %+v", *s1, s1want)
	}
	runtime.KeepAlive(s0)
	runtime.KeepAlive(s1)
}

func TestDeepCloneDeepIndependence(t *testing.T) {
	type inner struct {
		Xs []int
	}
	type outer struct {
		P *inner
		M map[string]*inner
		S [][]int
	}
	v := outer{
		P: &inner{Xs: []int{1, 2}},
		M: map[string]*inner{"k": {Xs: []int{3, 4}}},
		S: [][]int{{5, 6}},
	}
	c := deepClone(&v)
	if !reflect.DeepEqual(*c, v) {
		t.Fatal("deep clone not equal to original")
	}
	// Mutating every nested container in the clone must not touch the original.
	c.P.Xs[0] = 99
	c.M["k"].Xs[0] = 99
	c.S[0][0] = 99
	if v.P.Xs[0] != 1 {
		t.Error("clone shares pointer-field backing with original")
	}
	if v.M["k"].Xs[0] != 3 {
		t.Error("clone shares map-value backing with original")
	}
	if v.S[0][0] != 5 {
		t.Error("clone shares nested-slice backing with original")
	}
	// The clone's pointer/map-value must be distinct allocations.
	if c.P == v.P {
		t.Error("clone pointer field aliases original pointer")
	}
	if c.M["k"] == v.M["k"] {
		t.Error("clone map value aliases original map value")
	}
}

func TestDeepCloneTimeField(t *testing.T) {
	v := bnRegTimeStamped{ID: "a", TS: time.Unix(1700000000, 999).UTC()}
	c := deepClone(&v)
	if !c.TS.Equal(v.TS) {
		t.Fatalf("clone zeroed time.Time: orig=%v clone=%v", v.TS, c.TS)
	}
	td, err := descOf(reflect.TypeFor[bnRegTimeStamped]())
	if err != nil {
		t.Fatal(err)
	}
	if valueFingerprint(td, unsafe.Pointer(&v)) != valueFingerprint(td, unsafe.Pointer(c)) {
		t.Fatal("clone fingerprint differs from original for a time.Time-bearing type")
	}
}

func TestBaselineRegistryEmpty(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	if r == nil {
		t.Fatal("NewBaselineRegistry returned nil")
	}
	if got := r.Len(); got != 0 {
		t.Fatalf("empty registry Len() = %d, want 0", got)
	}
}

func TestDeepCloneFidelity(t *testing.T) {
	td, err := descOf(reflect.TypeFor[dnTop]())
	if err != nil {
		t.Fatal(err)
	}
	for seed := range int64(200) {
		v := gen[dnTop](seed)
		c := deepClone(&v)
		if !reflect.DeepEqual(*c, v) {
			t.Fatalf("seed %d: clone not DeepEqual to original", seed)
		}
		fpV := valueFingerprint(td, unsafe.Pointer(&v))
		fpC := valueFingerprint(td, unsafe.Pointer(c))
		if fpV != fpC {
			t.Fatalf("seed %d: clone fingerprint %x != original %x", seed, fpC, fpV)
		}
	}
}

func TestDeepCloneNilVsEmpty(t *testing.T) {
	type S struct {
		NilSlice   []int
		EmptySlice []int
		NilMap     map[string]int
		EmptyMap   map[string]int
	}
	v := S{EmptySlice: []int{}, EmptyMap: map[string]int{}}
	c := deepClone(&v)
	if c.NilSlice != nil {
		t.Error("nil slice became non-nil")
	}
	if c.EmptySlice == nil {
		t.Error("empty slice became nil")
	}
	if c.NilMap != nil {
		t.Error("nil map became non-nil")
	}
	if c.EmptyMap == nil {
		t.Error("empty map became nil")
	}
}

func TestDeepCloneIndependent(t *testing.T) {
	type S struct {
		Xs []int
		M  map[string]int
	}
	v := S{Xs: []int{1, 2, 3}, M: map[string]int{"a": 1}}
	c := deepClone(&v)
	c.Xs[0] = 99
	c.M["a"] = 99
	if v.Xs[0] != 1 {
		t.Error("mutating clone slice touched original")
	}
	if v.M["a"] != 1 {
		t.Error("mutating clone map touched original")
	}
}

func TestRegisterIdMatchesDiffBaseFP(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	v := gen[dnTop](7)
	id := r.Register(&v)

	patch, err := Diff(v, gen[dnTop](8), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	h, _, err := readPatchHeader(patch)
	if err != nil {
		t.Fatal(err)
	}
	if h.flags&flagPatchBaseFP == 0 {
		t.Fatal("patch unexpectedly has no baseFP")
	}
	if h.baseFP != id {
		t.Fatalf("Register id %x != Diff baseFP %x", id, h.baseFP)
	}
	if got := r.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
	// The registry holds v only through a weak.Pointer; keep it reachable
	// across Len() or -race GC may reclaim it and Len() prunes to 0.
	runtime.KeepAlive(v)
}

func TestBaselineApplyHappyPath(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	s0 := gen[dnTop](1)
	s1want := gen[dnTop](2)

	r.Register(&s0)
	patch, err := Diff(s0, s1want, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	s1, err := r.Apply(patch)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !reflect.DeepEqual(*s1, s1want) {
		t.Fatal("Apply result != expected new value")
	}
	if !reflect.DeepEqual(s0, gen[dnTop](1)) {
		t.Fatal("Apply mutated the registered baseline")
	}
	s2want := gen[dnTop](3)
	patch2, err := Diff(*s1, s2want, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := r.Apply(patch2)
	if err != nil {
		t.Fatalf("Apply chain step 2: %v", err)
	}
	if !reflect.DeepEqual(*s2, s2want) {
		t.Fatal("chained Apply result != expected")
	}
	runtime.KeepAlive(s1)
}

func TestBaselineApplyNoFingerprint(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	s0 := gen[dnTop](1)
	r.Register(&s0)
	patch, err := Diff(s0, gen[dnTop](2), OptBalanced|OptDeltaNoBaseFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Apply(patch); !errors.Is(err, ErrBaselineRequired) {
		t.Fatalf("Apply error = %v, want ErrBaselineRequired", err)
	}
	runtime.KeepAlive(s0)
}

func TestBaselineApplyEvicted(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	s0 := gen[dnTop](1)
	patch, err := Diff(s0, gen[dnTop](2), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	r.Register(&s0)
	// Live strong ref → resolves.
	if _, err := r.Apply(patch); err != nil {
		t.Fatalf("Apply with live baseline: %v", err)
	}
	runtime.KeepAlive(s0)

	// Separate registry whose only baseline goes out of scope → GC reclaims it.
	r2 := NewBaselineRegistry[dnTop]()
	func() {
		local := gen[dnTop](1)
		r2.Register(&local)
	}()
	runtime.GC()
	runtime.GC()
	if _, err := r2.Apply(patch); !errors.Is(err, ErrBaselineEvicted) {
		t.Fatalf("Apply after GC = %v, want ErrBaselineEvicted", err)
	}
}

func TestBaselineApplyUnknownBaseline(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	patch, err := Diff(gen[dnTop](1), gen[dnTop](2), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Apply(patch); !errors.Is(err, ErrBaselineEvicted) {
		t.Fatalf("Apply of never-registered baseline = %v, want ErrBaselineEvicted", err)
	}
}

func TestBaselineStreamChain(t *testing.T) {
	tiers := []Options{OptSpeed, OptBalanced, OptCompression}
	for _, tier := range tiers {
		r := NewBaselineRegistry[dnTop]()
		prev := gen[dnTop](1000)
		cur := &prev
		r.Register(cur)
		for step := int64(1001); step < 1030; step++ {
			next := gen[dnTop](step)
			patch, err := Diff(*cur, next, tier)
			if err != nil {
				t.Fatalf("tier %v step %d Diff: %v", tier, step, err)
			}
			got, err := r.Apply(patch)
			if err != nil {
				t.Fatalf("tier %v step %d Apply: %v", tier, step, err)
			}
			if !reflect.DeepEqual(*got, next) {
				t.Fatalf("tier %v step %d: reconstructed != expected", tier, step)
			}
			// The registry holds baselines through non-pinning weak pointers, so
			// the baseline this step's patch is based on (*cur) must stay reachable
			// until Apply has resolved it. Without this, the compiler treats cur as
			// dead right after Diff reads it (the next op overwrites cur), letting
			// the GC reclaim the intermediate baseline before Apply -> flaky
			// ErrBaselineEvicted at a chain step. Keep cur live across Apply.
			runtime.KeepAlive(cur)
			cur = got
		}
		runtime.KeepAlive(cur)
	}
}

func TestBaselineConcurrent(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	base := gen[dnTop](1)
	r.Register(&base)
	patch, err := Diff(base, gen[dnTop](2), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range 200 {
				switch i % 3 {
				case 0:
					v := gen[dnTop](int64(g*1000 + i))
					r.Register(&v)
					runtime.KeepAlive(v)
				case 1:
					if got, err := r.Apply(patch); err == nil {
						runtime.KeepAlive(got)
					}
				case 2:
					_ = r.Len()
				}
			}
		}(g)
	}
	wg.Wait()
	runtime.KeepAlive(base)
}

func TestBaselineApplyHostile(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	base := gen[dnTop](1)
	r.Register(&base)
	good, err := Diff(base, gen[dnTop](2), OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	seeds := make([][]byte, 0, 4+len(good))
	seeds = append(seeds, nil, []byte{}, []byte{0x00}, []byte("not a patch at all"))
	for i := range len(good) {
		trunc := make([]byte, i)
		copy(trunc, good[:i])
		seeds = append(seeds, trunc)
	}
	for i, b := range seeds {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("seed %d panicked: %v", i, rec)
				}
			}()
			_, _ = r.Apply(b)
		}()
	}
	runtime.KeepAlive(base)
}

func TestBaselineLenDropsToZero(t *testing.T) {
	r := NewBaselineRegistry[dnTop]()
	func() {
		for i := range int64(16) {
			v := gen[dnTop](i)
			r.Register(&v)
			runtime.KeepAlive(v)
		}
	}()
	runtime.GC()
	runtime.GC()
	if got := r.Len(); got != 0 {
		t.Fatalf("Len() = %d after release+GC, want 0 (registry must not pin)", got)
	}
}

func FuzzBaselineApply(f *testing.F) {
	base := gen[dnTop](1)
	good, _ := Diff(base, gen[dnTop](2), OptBalanced)
	f.Add(good)
	f.Add([]byte{})
	f.Add([]byte("garbage"))
	f.Fuzz(func(t *testing.T, patch []byte) {
		r := NewBaselineRegistry[dnTop]()
		b := gen[dnTop](1)
		r.Register(&b)
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("panic on patch %x: %v", patch, rec)
			}
		}()
		got, err := r.Apply(patch)
		if err == nil && got == nil {
			t.Fatal("Apply returned nil,nil")
		}
		runtime.KeepAlive(b)
		runtime.KeepAlive(got)
	})
}
