package qdf

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
)

// cycNode is a self-referential (cyclic) type: building its descriptor requires
// the *cycNode pointer descriptor to close over the still-building cycNode
// struct descriptor. The pre-fix code published the inner *cycNode descriptor
// to the global typeCache as soon as ITS own fillDesc returned — while the
// enclosing cycNode struct descriptor's .encode/.decode were still unwritten.
// A second goroutine that loaded the prematurely-published descriptor then read
// those fields concurrently with the first goroutine writing them (data race +
// latent nil-func dispatch panic).
type cycNode struct {
	V    int64    `qdf:"v"`
	Self *cycNode `qdf:"self"`
}

// TestCyclicDesc_ConcurrentColdBuildNoRace forces repeated concurrent COLD
// builds of a cyclic type (deleting the cache entries each round to reopen the
// build window) and round-trips a value through every goroutine. Run with
// -race: the pre-fix premature publish trips the detector; the deferred-publish
// fix is clean.
func TestCyclicDesc_ConcurrentColdBuildNoRace(t *testing.T) {
	tCyc := reflect.TypeFor[cycNode]()
	tPtr := reflect.TypeFor[*cycNode]()
	val := cycNode{V: 1, Self: &cycNode{V: 2}}

	workers := runtime.GOMAXPROCS(0) * 4
	if workers < 8 {
		workers = 8
	}

	for round := 0; round < 300; round++ {
		// Force a cold build: drop the cached descriptors so every round
		// rebuilds the graph and re-opens the concurrent-publish window.
		typeCache.Delete(tCyc)
		typeCache.Delete(tPtr)

		var wg sync.WaitGroup
		start := make(chan struct{})
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				buf, err := Marshal(val, OptBalanced)
				if err != nil {
					t.Errorf("round %d Marshal: %v", round, err)
					return
				}
				var out cycNode
				if err := Unmarshal(buf, &out); err != nil {
					t.Errorf("round %d Unmarshal: %v", round, err)
					return
				}
				if out.V != 1 || out.Self == nil || out.Self.V != 2 {
					t.Errorf("round %d wrong value: %+v", round, out)
				}
			}()
		}
		close(start)
		wg.Wait()
		if t.Failed() {
			return
		}
	}
}
