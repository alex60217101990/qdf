package qdf

import (
	"sync"
	"testing"
)

// FSSTDict.Marshal must enable FSST + its columnar prerequisites itself, so a
// bare opts (even OptSpeed/0) still compresses — not silently a no-op.
func TestFSSTDictImpliesPrerequisites(t *testing.T) {
	rows := mkRows(genURLs(1000))
	d := TrainFSSTDictStrings(genURLs(1000))
	plain, _ := Marshal(rows, OptBalanced)
	for _, opt := range []Options{0, OptSpeed, OptDense, OptCompression} {
		b, err := d.Marshal(rows, opt)
		if err != nil {
			t.Fatalf("opt=%d: %v", opt, err)
		}
		if len(b) >= len(plain) {
			t.Fatalf("opt=%d: dict.Marshal did not compress (%d >= OptBalanced %d) — prerequisites not applied", opt, len(b), len(plain))
		}
		var back []fsstRow
		if err := Unmarshal(b, &back); err != nil {
			t.Fatalf("opt=%d decode: %v", opt, err)
		}
		assertRowsEqual(t, "implies", rows, back)
	}
}

func TestFSSTDictRoundTrip(t *testing.T) {
	rows := mkRows(genURLs(1000))
	d := TrainFSSTDictStrings(genURLs(1000))
	b, err := d.Marshal(rows, OptBalanced)
	if err != nil {
		t.Fatal(err)
	}
	var back []fsstRow
	if err := Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	assertRowsEqual(t, "dict", rows, back)
}

// A dictionary-produced wire is self-describing: every FSST column still carries
// its table, so a plain Unmarshal (no dict) decodes it, and the bytes match what
// per-batch training would accept.
func TestFSSTDictSelfDescribing(t *testing.T) {
	rows := mkRows(genURLs(1000))
	d := TrainFSSTDictStrings(genURLs(1000))
	withDict, err := d.AppendMarshal(nil, rows, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	var back []fsstRow
	if err := Unmarshal(withDict, &back); err != nil {
		t.Fatal(err)
	}
	assertRowsEqual(t, "self-describing", rows, back)
}

func TestFSSTDictDeterministic(t *testing.T) {
	samples := genURLs(500)
	rows := mkRows(genURLs(1000))
	a, _ := TrainFSSTDictStrings(samples).Marshal(rows, OptBalanced)
	b, _ := TrainFSSTDictStrings(samples).Marshal(rows, OptBalanced)
	if string(a) != string(b) {
		t.Fatal("same samples + same rows must produce identical wire")
	}
}

// TestFSSTDictNeverLarger: the dict path must not grow the wire vs OptBalanced.
func TestFSSTDictNeverLarger(t *testing.T) {
	for _, msgs := range [][]string{genURLs(1000), genRandom(1000), repeatEach([]string{"A", "B"}, 1000)} {
		rows := mkRows(msgs)
		base, _ := Marshal(rows, OptBalanced)
		d := TrainFSSTDictStrings(msgs)
		withDict, _ := d.Marshal(rows, OptBalanced)
		if len(withDict) > len(base) {
			t.Fatalf("dict grew wire: %d > %d", len(withDict), len(base))
		}
		var back []fsstRow
		if err := Unmarshal(withDict, &back); err != nil {
			t.Fatal(err)
		}
		assertRowsEqual(t, "nl", rows, back)
	}
}

// An immutable dictionary is safe to share across goroutines.
func TestFSSTDictConcurrent(t *testing.T) {
	rows := mkRows(genURLs(512))
	d := TrainFSSTDictStrings(genURLs(512))
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				b, err := d.Marshal(rows, OptBalanced)
				if err != nil {
					t.Error(err)
					return
				}
				var back []fsstRow
				if err := Unmarshal(b, &back); err != nil {
					t.Error(err)
					return
				}
				if len(back) != len(rows) {
					t.Errorf("len %d want %d", len(back), len(rows))
					return
				}
			}
		}()
	}
	wg.Wait()
}
