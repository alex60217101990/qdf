package qdf

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
)

// TestBlockDepthGuard verifies the block decoders bound recursion depth: a block
// body could itself carry tagPackBlock/tagZoneChunk, so nested payloads must not
// overflow the stack. At the depth cap the decoder returns ErrCycleDetected
// without touching the buffer.
func TestBlockDepthGuard(t *testing.T) {
	for _, fn := range []func(d *Decoder) error{
		func(d *Decoder) error { return d.readBlockInt64Into(new([]int64)) },
		func(d *Decoder) error { return d.readBlockUint64Into(new([]uint64)) },
	} {
		d := &Decoder{maxDepth: DefaultMaxDepth, depth: DefaultMaxDepth}
		if err := fn(d); !errors.Is(err, ErrCycleDetected) {
			t.Fatalf("want ErrCycleDetected at depth cap, got %v", err)
		}
	}
}

// blockArchetypes returns int64 columns: regime-shifting ones (where the
// per-block codec should win) and flat ones (where it must not fire / never
// grow). Deterministic seed for reproducibility.
func blockArchetypes() []struct {
	name   string
	regime bool
	gen    func() []int64
} {
	const N = 8192
	rng := rand.New(rand.NewSource(42))
	return []struct {
		name   string
		regime bool
		gen    func() []int64
	}{
		{"sorted-then-random", true, func() []int64 {
			s := make([]int64, N)
			for i := range N / 2 {
				s[i] = int64(1_700_000_000 + i)
			}
			for i := N / 2; i < N; i++ {
				s[i] = rng.Int63n(1 << 40)
			}
			return s
		}},
		{"bursty-timestamps", true, func() []int64 {
			s := make([]int64, N)
			v := int64(1_700_000_000)
			for i := range s {
				if i%512 == 0 {
					v += rng.Int63n(1 << 30)
				}
				v += rng.Int63n(3)
				s[i] = v
			}
			return s
		}},
		{"mixed-card-id", true, func() []int64 {
			s := make([]int64, N)
			for i := range s {
				if (i/1024)%2 == 0 {
					s[i] = int64(rng.Intn(4))
				} else {
					s[i] = rng.Int63n(1 << 48)
				}
			}
			return s
		}},
		{"rle-then-noise", true, func() []int64 {
			s := make([]int64, N)
			for i := range N * 3 / 4 {
				s[i] = 200
			}
			for i := N * 3 / 4; i < N; i++ {
				s[i] = rng.Int63n(1 << 20)
			}
			return s
		}},
		{"flat-low-card-enum", false, func() []int64 {
			s := make([]int64, N)
			for i := range s {
				s[i] = int64(rng.Intn(6))
			}
			return s
		}},
		{"flat-uniform-random", false, func() []int64 {
			s := make([]int64, N)
			for i := range s {
				s[i] = rng.Int63n(1 << 32)
			}
			return s
		}},
		{"flat-tight-monotonic", false, func() []int64 {
			s := make([]int64, N)
			for i := range s {
				s[i] = int64(1_700_000_000 + i*3)
			}
			return s
		}},
		{"flat-small-values", false, func() []int64 {
			s := make([]int64, N)
			for i := range s {
				s[i] = int64(1000 + rng.Intn(200))
			}
			return s
		}},
		{"short-below-floor", false, func() []int64 { // < blockCodecMinLen
			s := make([]int64, 300)
			for i := range s {
				s[i] = int64(i)
			}
			return s
		}},
	}
}

// marshalBaseline encodes v with the block codec disabled (whole-column path).
func marshalBaselineI64(t *testing.T, v []int64, opts Options) []byte {
	t.Helper()
	blockCodecEnabled = false
	defer func() { blockCodecEnabled = true }()
	b, err := Marshal(v, opts)
	if err != nil {
		t.Fatalf("baseline marshal: %v", err)
	}
	return b
}

func TestBlockRoundtripI64(t *testing.T) {
	opts := []Options{OptSpeed, OptQPack, OptBalanced, OptCompression}
	for _, a := range blockArchetypes() {
		for _, o := range opts {
			s := a.gen()
			b, err := Marshal(s, o)
			if err != nil {
				t.Fatalf("%s/%v marshal: %v", a.name, o, err)
			}
			var out []int64
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("%s/%v unmarshal: %v", a.name, o, err)
			}
			if len(out) != len(s) {
				t.Fatalf("%s/%v len %d != %d", a.name, o, len(out), len(s))
			}
			for i := range s {
				if out[i] != s[i] {
					t.Fatalf("%s/%v [%d] = %d != %d", a.name, o, i, out[i], s[i])
				}
			}
		}
	}
}

func TestBlockRoundtripU64(t *testing.T) {
	for _, a := range blockArchetypes() {
		si := a.gen()
		s := make([]uint64, len(si))
		for i, v := range si {
			if v < 0 {
				v = -v
			}
			s[i] = uint64(v)
		}
		for _, o := range []Options{OptQPack, OptBalanced, OptCompression} {
			b, err := Marshal(s, o)
			if err != nil {
				t.Fatalf("%s/%v marshal: %v", a.name, o, err)
			}
			var out []uint64
			if err := Unmarshal(b, &out); err != nil {
				t.Fatalf("%s/%v unmarshal: %v", a.name, o, err)
			}
			if len(out) != len(s) {
				t.Fatalf("%s/%v len mismatch", a.name, o)
			}
			for i := range s {
				if out[i] != s[i] {
					t.Fatalf("%s/%v [%d] mismatch", a.name, o, i)
				}
			}
		}
	}
}

// TestBlockNeverLarger: the block wire is never larger than the whole-column
// baseline; flat columns are byte-identical (block did not fire); at least one
// regime column is strictly smaller (the codec earns its keep).
func TestBlockNeverLarger(t *testing.T) {
	regimeWon := false
	for _, a := range blockArchetypes() {
		s := a.gen()
		block, err := Marshal(s, OptQPack)
		if err != nil {
			t.Fatalf("%s marshal: %v", a.name, err)
		}
		base := marshalBaselineI64(t, s, OptQPack)
		if len(block) > len(base) {
			t.Fatalf("%s: block %d > baseline %d (never-larger violated)", a.name, len(block), len(base))
		}
		if !a.regime {
			if !bytes.Equal(block, base) {
				t.Fatalf("%s: flat column not byte-identical to baseline (block %d, base %d)",
					a.name, len(block), len(base))
			}
		}
		if a.regime && len(block) < len(base) {
			regimeWon = true
			t.Logf("%s: %d -> %d (%.1f%% smaller)", a.name, len(base), len(block),
				100*float64(len(base)-len(block))/float64(len(base)))
		}
	}
	if !regimeWon {
		t.Fatal("no regime column won — block codec never fired")
	}
}

// TestBlockSelectiveDecode: a predicate over one column narrows the row set,
// and a projected-only block column is then decoded for matched rows only —
// untouched blocks are skipped (asserted via the instrumentation counter), and
// the result equals a full decode + manual filter.
func TestBlockSelectiveDecode(t *testing.T) {
	type BRow struct {
		Sel int64 // predicate column (low-card, not block)
		Val int64 // block-tagged regime column, projected but not referenced
	}
	const N = 4096
	rng := rand.New(rand.NewSource(7))
	rows := make([]BRow, N)
	for i := range rows {
		// Val: sorted-then-random regime so it stores as a block column.
		if i < N/2 {
			rows[i].Val = int64(1_700_000_000 + i)
		} else {
			rows[i].Val = rng.Int63n(1 << 40)
		}
		// Sel == 1 only for the first 512 rows → matched rows cluster in the
		// first 1-2 blocks, so most blocks need never be decoded.
		if i < 512 {
			rows[i].Sel = 1
		}
	}

	b, err := Marshal(rows, OptBalanced|OptColumnIndex)
	if err != nil {
		t.Fatal(err)
	}

	blockSelectiveBlocksDecoded.Store(0)
	var out []BRow
	if err := Unmarshal(b, &out,
		Where("Sel", func(v int64) bool { return v == 1 }),
		Select("Sel", "Val")); err != nil {
		t.Fatal(err)
	}

	// Expected = full filter.
	var want []BRow
	for _, r := range rows {
		if r.Sel == 1 {
			want = append(want, r)
		}
	}
	if len(out) != len(want) {
		t.Fatalf("len %d != %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("[%d] = %+v != %+v", i, out[i], want[i])
		}
	}
	if blockSelectiveBlocksDecoded.Load() == 0 {
		t.Fatal("selective path did not run (Val column was not block-tagged?)")
	}
	if blockSelectiveBlocksDecoded.Load() > 2 {
		t.Fatalf("selective decoded %d blocks; matched rows span only the first ~2 blocks",
			blockSelectiveBlocksDecoded.Load())
	}
	t.Logf("selective decoded %d block(s) for %d matched rows", blockSelectiveBlocksDecoded.Load(), len(out))
}

// TestBlockSelectiveAdversarial exercises the selective decode integration
// across predicate shapes that the happy-path test does not: all-match,
// scattered-match (every block), zero-match, two deferred block columns, a block
// column used as BOTH predicate and projection (must not defer), and the
// no-ColumnIndex fallback. Each must equal a full decode + manual filter.
func TestBlockSelectiveAdversarial(t *testing.T) {
	type R struct {
		Sel int64 // predicate column
		A   int64 // block-tagged regime column
		B   int64 // second block-tagged regime column
	}
	const N = 4096
	rng := rand.New(rand.NewSource(11))
	rows := make([]R, N)
	for i := range rows {
		rows[i].Sel = int64(i % 10) // 0..9
		if i < N/2 {
			rows[i].A = int64(1_700_000_000 + i)
			rows[i].B = int64(2_000_000_000 + i*2)
		} else {
			rows[i].A = rng.Int63n(1 << 40)
			rows[i].B = rng.Int63n(1 << 44)
		}
	}

	filter := func(pred func(R) bool) []R {
		var w []R
		for _, r := range rows {
			if pred(r) {
				w = append(w, r)
			}
		}
		return w
	}
	eq := func(t *testing.T, name string, got, want []R) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: len %d != %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: [%d] %+v != %+v", name, i, got[i], want[i])
			}
		}
	}

	cases := []struct {
		name  string
		opts  Options
		pred  func(R) bool
		qpred func(int64) bool
	}{
		{"all-match", OptBalanced | OptColumnIndex, func(r R) bool { return r.Sel >= 0 }, func(v int64) bool { return v >= 0 }},
		{"scattered-every-block", OptBalanced | OptColumnIndex, func(r R) bool { return r.Sel == 3 }, func(v int64) bool { return v == 3 }},
		{"zero-match", OptBalanced | OptColumnIndex, func(r R) bool { return r.Sel == 99 }, func(v int64) bool { return v == 99 }},
		{"two-deferred", OptBalanced | OptColumnIndex, func(r R) bool { return r.Sel < 2 }, func(v int64) bool { return v < 2 }},
		{"no-colindex-fallback", OptBalanced, func(r R) bool { return r.Sel < 2 }, func(v int64) bool { return v < 2 }},
	}
	for _, c := range cases {
		b, err := Marshal(rows, c.opts)
		if err != nil {
			t.Fatalf("%s marshal: %v", c.name, err)
		}
		var out []R
		// Project ALL fields so the full-struct comparison is valid; A and B are
		// projected-not-referenced (only Sel is the predicate) → block-selective.
		if err := Unmarshal(b, &out,
			Where("Sel", c.qpred),
			Select("Sel", "A", "B")); err != nil {
			t.Fatalf("%s unmarshal: %v", c.name, err)
		}
		eq(t, c.name, out, filter(c.pred))
	}

	// Block column used as BOTH predicate and projection: must full-decode (not
	// defer), and filtering on it must be correct.
	{
		b, _ := Marshal(rows, OptBalanced|OptColumnIndex)
		var out []R
		if err := Unmarshal(b, &out,
			Where("A", func(v int64) bool { return v < 1_700_000_100 }),
			Select("Sel", "A", "B")); err != nil {
			t.Fatalf("pred-on-block: %v", err)
		}
		eq(t, "pred-on-block", out, filter(func(r R) bool { return r.A < 1_700_000_100 }))
	}
}

// TestBlockProperty fuzzes many random int64/uint64 columns (varied sizes incl.
// block-boundary edges, varied distributions) and asserts every one round-trips
// bit-exactly AND is never larger than the whole-column baseline.
func TestBlockProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	sizes := []int{511, 512, 513, 768, 1024, 1025, 2000, 4097}
	for iter := range 200 {
		n := sizes[rng.Intn(len(sizes))]
		s := make([]int64, n)
		mode := rng.Intn(6)
		seg := 1 + rng.Intn(8) // number of regime segments
		segLen := n/seg + 1
		for i := range s {
			switch (i/segLen + mode) % 6 {
			case 0:
				s[i] = int64(1_700_000_000 + i) // monotonic
			case 1:
				s[i] = rng.Int63n(1 << uint(8+rng.Intn(48))) // random varied width
			case 2:
				s[i] = int64(rng.Intn(8)) // low-card
			case 3:
				s[i] = 42 // constant run
			case 4:
				s[i] = int64(rng.Intn(3)) - 1 // tiny signed incl negatives
			default:
				s[i] = -rng.Int63n(1 << 30) // negative wide
			}
		}
		for _, o := range []Options{OptQPack, OptBalanced, OptCompression} {
			block, err := Marshal(s, o)
			if err != nil {
				t.Fatalf("iter %d n=%d %v marshal: %v", iter, n, o, err)
			}
			var out []int64
			if err := Unmarshal(block, &out); err != nil {
				t.Fatalf("iter %d n=%d %v unmarshal: %v", iter, n, o, err)
			}
			if len(out) != n {
				t.Fatalf("iter %d n=%d %v len %d", iter, n, o, len(out))
			}
			for i := range s {
				if out[i] != s[i] {
					t.Fatalf("iter %d n=%d %v [%d] %d != %d", iter, n, o, i, out[i], s[i])
				}
			}
			base := marshalBaselineI64(t, s, o)
			if len(block) > len(base) {
				t.Fatalf("iter %d n=%d %v never-larger VIOLATED: block %d > base %d",
					iter, n, o, len(block), len(base))
			}
		}
	}
}

// TestBlockColumnarFullDecode exercises the scratch-reusing Into decode path
// (decodeColumnInto / the codegen ReadIntColumn path) with a block-tagged
// column — a non-query columnar round-trip.
func TestBlockColumnarFullDecode(t *testing.T) {
	type R struct {
		A int64
		V int64
		U uint64
	}
	const N = 4096
	rng := rand.New(rand.NewSource(3))
	rows := make([]R, N)
	for i := range rows {
		rows[i].A = int64(i)
		if i < N/2 { // regime → V and U store as block columns
			rows[i].V = int64(1_700_000_000 + i)
			rows[i].U = uint64(1_700_000_000 + i)
		} else {
			rows[i].V = rng.Int63n(1 << 40)
			rows[i].U = rng.Uint64() & (1<<40 - 1)
		}
	}
	for _, o := range []Options{OptBalanced, OptCompression, OptBalanced | OptColumnIndex} {
		b, err := Marshal(rows, o)
		if err != nil {
			t.Fatalf("%v marshal: %v", o, err)
		}
		var out []R
		if err := Unmarshal(b, &out); err != nil {
			t.Fatalf("%v unmarshal: %v", o, err)
		}
		if len(out) != N {
			t.Fatalf("%v len %d", o, len(out))
		}
		for i := range rows {
			if out[i] != rows[i] {
				t.Fatalf("%v [%d] = %+v != %+v", o, i, out[i], rows[i])
			}
		}
	}
}

// TestBlockMalformed: corrupted block wire must error, never panic / OOM.
func TestBlockMalformed(t *testing.T) {
	s := blockArchetypes()[0].gen() // sorted-then-random → block form
	good, err := Marshal(s, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	idx := bytes.IndexByte(good, tagPackBlock)
	if idx < 0 {
		t.Fatal("no block tag in wire")
	}

	mutate := func(name string, f func(b []byte)) {
		b := append([]byte(nil), good...)
		f(b)
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s: panic %v", name, r)
			}
		}()
		var out []int64
		_ = Unmarshal(b, &out) // error is fine; a panic/OOM is not
	}

	mutate("bad blkLog", func(b []byte) { b[idx+2] = 7 }) // not 8 or 10
	mutate("kind byte garbage", func(b []byte) { b[idx+1] = 0x55 })
	// Truncation (an actually shorter slice):
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("truncate: panic %v", r)
			}
		}()
		var out []int64
		_ = Unmarshal(good[:idx+3], &out)
	}()

	// Deterministic byte-flip fuzz over the block region.
	for off := idx; off < len(good); off++ {
		for _, bitp := range []byte{0x01, 0x80, 0xff} {
			b := append([]byte(nil), good...)
			b[off] ^= bitp
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("flip off=%d bit=%#x: panic %v", off, bitp, r)
					}
				}()
				var out []int64
				_ = Unmarshal(b, &out)
			}()
		}
	}
}

// FuzzBlockDecode: arbitrary bytes decoded as []int64 must never panic.
func FuzzBlockDecode(f *testing.F) {
	s := blockArchetypes()[1].gen()
	if b, err := Marshal(s, OptQPack); err == nil {
		f.Add(b)
	}
	f.Fuzz(func(_ *testing.T, data []byte) {
		var out []int64
		_ = Unmarshal(data, &out)
	})
}

// TestBlockWireTag: a regime column's wire actually carries tagPackBlock.
func TestBlockWireTag(t *testing.T) {
	s := blockArchetypes()[1].gen() // bursty-timestamps
	b, err := Marshal(s, OptQPack)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.IndexByte(b, tagPackBlock) < 0 {
		t.Fatal("wire does not contain tagPackBlock")
	}
}
