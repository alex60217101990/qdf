package qdf

import "testing"

// TestRecycledMapsCounterMatchesLists gates the one invariant dropRecycledMaps
// introduced: d.recycledMaps must equal the total length of the per-type free
// lists, at every point a decode can observe it.
//
// The counter exists because dropRecycledMaps keeps the map's keys (and their
// slices' capacity) while emptying them, so len(d.mapFreeList) can no longer
// answer "is anything recycled?" — it stays non-zero over empty lists. Three
// call sites pop from those lists and one pushes; a future edit that forgets to
// move the counter would either lose recycling silently (counter too low) or
// send reuseOrMakeMap hunting through empty lists on every decode (too high).
// Neither shows up as a failure anywhere else.
func TestRecycledMapsCounterMatchesLists(t *testing.T) {
	type row struct {
		Name string            `qdf:"name"`
		Tags map[string]string `qdf:"tags"`
	}

	mk := func(n int) []byte {
		src := make([]row, n)
		for i := range src {
			src[i] = row{
				Name: "svc",
				Tags: map[string]string{"env": "prod", "ver": "1.2.3"},
			}
		}
		w, err := Marshal(src, OptSpeed)
		if err != nil {
			t.Fatalf("marshal %d: %v", n, err)
		}
		return w
	}
	// Two lengths, alternated, under OptSpeed. Both details are load-bearing and
	// both were found by watching the free list stay empty:
	//
	// A DIFFERENT length is what makes decode-slice-reuse zero the old elements,
	// which is the only path that harvests their maps onto the free list. Two
	// same-length decodes take reuseOrMakeMap's direct-target branch and never
	// touch the list — the first version of this test did exactly that and
	// passed with the counter deliberately broken three different ways.
	//
	// And OptSpeed keeps the payload on the row-major decoder. Under OptBalanced
	// this shape goes columnar, which never reaches harvestMaps at all, so the
	// free list stays empty however the lengths vary.
	wires := [][]byte{mk(64), mk(48)}

	check := func(d *Decoder, when string) {
		t.Helper()
		total := 0
		for _, lst := range d.mapFreeList {
			total += len(lst)
		}
		if d.recycledMaps != total {
			t.Fatalf("%s: recycledMaps=%d but the lists hold %d", when, d.recycledMaps, total)
		}
	}

	dec := NewDecoder()

	var out []row
	for i := range 6 {
		w := wires[i%2]
		dec.SetInput(w)
		if err := decodeReflect(dec, &out); err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
		check(dec, "after decode")
		if out[0].Tags["env"] != "prod" || out[0].Tags["ver"] != "1.2.3" {
			t.Fatalf("decode %d: tags did not round-trip: %v", i, out[0].Tags)
		}
	}
	// And after the reset that empties the lists without freeing them.
	dec.dropRecycledMaps()
	check(dec, "after dropRecycledMaps")
	if dec.recycledMaps != 0 {
		t.Fatalf("dropRecycledMaps left recycledMaps=%d", dec.recycledMaps)
	}
}
