// Command qdf-bench measures representative qdf serialize/deserialize
// performance over the adalanche-sampledata local-machine dumps
// (github.com/lkarlslund/adalanche-sampledata). With no -datapath it downloads
// the sample-data tarball into a temp dir, extracts just the localmachine dumps,
// and removes everything when the run ends; pass -datapath to use a local clone.
//
// For each encode option bundle (every Opt* flag exercised at least once) and
// each decode mode (copy / nocopy / arena) it reports ser/deser ns/op, B/op, and
// allocs/op (averaged over the sample files) plus the wire size, for two payload
// representations — typed Go structs and a dynamic map[string]any — and verifies
// a lossless round trip before timing. Each measurement is warmed up first and
// isolated to the qdf call alone (see benchOp).
//
// A single binary reports the qdf build tags it was compiled with; run.sh builds
// and runs one binary per tag combination, since build tags cannot change at
// runtime.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/alex60217101990/qdf"
)

// bundles is the full encode-option matrix. It covers each of the three presets
// (Speed / Balanced / Compression) and exercises every individual Opt* flag at
// least once over a valid base (most state-ref codecs require OptDense, the
// float/string codecs ride on OptQPack). OptDeltaNoBaseFingerprint is omitted —
// it only affects Diff/Apply, not the Marshal/Unmarshal path this bench times.
var bundles = []struct {
	name string
	opts qdf.Options
}{
	{"Speed", qdf.OptSpeed},
	{"Dense", qdf.OptDense},
	{"Dense+QPack", qdf.OptDense | qdf.OptQPack},
	{"Balanced", qdf.OptBalanced},
	{"Bal+Gorilla", qdf.OptBalanced | qdf.OptGorillaFloat},
	{"Bal+RANS", qdf.OptBalanced | qdf.OptRANS},
	{"Bal+FSST", qdf.OptBalanced | qdf.OptFSST},
	{"Bal+ColIndex", qdf.OptBalanced | qdf.OptColumnIndex},
	{"Bal+MapShape", qdf.OptBalanced | qdf.OptMapShape},
	{"Bal+Canonical", qdf.OptBalanced | qdf.OptCanonical},
	{"Compression", qdf.OptCompression},
	{"Comp+ColIndex", qdf.OptCompression | qdf.OptColumnIndex},
}

// summaryBundle is the qdf option set used in the final qdf-vs-json-vs-msgpack
// diff table — the recommended default, so the headline comparison is against
// qdf as most callers would run it.
const summaryBundle = "Balanced"

// bundleOpts returns the qdf.Options for a bundle name (falls back to Balanced).
func bundleOpts(name string) qdf.Options {
	for _, b := range bundles {
		if b.name == name {
			return b.opts
		}
	}
	return qdf.OptBalanced
}

// mapDecModes is the decode-option axis for the map representation. The decode
// QueryOptions (WithNoCopy / WithArena) only ride on the dynamic Unmarshal path;
// the typed UnmarshalT API takes no options, so typed rows only ever run "copy".
//   - copy   — default: every string/[]byte is copied out of the input buffer.
//   - nocopy — WithNoCopy: decoded strings alias the input buffer (near-zero
//     alloc, ~2x faster on string-heavy data; lifetime-bound to the buffer).
//   - arena  — WithArena: copied string bodies are bump-packed into a reused
//     arena (reset per op to model per-message reuse) instead of one alloc each.
var mapDecModes = []struct {
	name      string
	build     func(a *qdf.Arena) qdf.QueryOption // nil => plain copy decode
	usesArena bool                               // build needs a reused arena
}{
	{"copy", nil, false},
	{"nocopy", func(*qdf.Arena) qdf.QueryOption { return qdf.WithNoCopy() }, false},
	{"arena", qdf.WithArena, true},
}

// stat accumulates per-(repr,bundle) results across the sample files. Time,
// bytes and allocs are summed (averaged on print). serLiveKiB / deserLiveKiB are
// a single retained-live-heap measurement on the last file's result (see
// retainedKiB) — NOT a per-file sum, so they are printed as-is.
type stat struct {
	serNs, deserNs           int64
	serB, deserB             uint64
	serAllocs, deserAllocs   uint64
	serLiveKiB, deserLiveKiB int64 // live heap one result occupies (see retainedKiB)
	wire                     uint64
	n                        int
}

// bundleRow holds one option bundle's measured stats (typed, and the copy-decode
// map row) for the matrix-vs-baseline overview.
type bundleRow struct {
	name    string
	typed   stat
	mapCopy stat
}

func (s *stat) addSer(ns int64, b, allocs uint64) {
	s.serNs += ns
	s.serB += b
	s.serAllocs += allocs
}

func (s *stat) addDeser(ns int64, b, allocs uint64) {
	s.deserNs += ns
	s.deserB += b
	s.deserAllocs += allocs
}

// keep* retain results ALIVE during a live-heap measurement (see liveHeapKiB).
// Package-level so the closures that append to them don't themselves escape per
// call; reset to nil between measurements to free the prior set.
var (
	keepBuf  [][]byte
	keepInfo []Info
	keepMap  []map[string]any
)

// liveSamples is how many results liveHeapKiB holds alive at once. The live heap
// is read as a delta of runtime.MemStats.HeapAlloc, which has span-level
// granularity (tens-to-hundreds of KiB); a single ~1 MiB result is swamped by
// that noise, so we hold this many and divide, amortizing the granularity to
// well under a percent.
const liveSamples = 24

// liveHeapKiB measures the RESIDENT memory ONE operation's result occupies. It
// resets the keepers, produces liveSamples results (each retained via a keeper),
// forces GC on both sides, and returns the live-heap delta divided by the sample
// count. Because every result is kept alive simultaneously, this is the settled
// working set the codec's output requires — and for the arena decode mode, where
// produce decodes into one un-reset arena, the amortized figure correctly
// includes the arena buffer the decoded strings alias (the memory the arena
// holds that participates in the decode). This replaces the old getrusage Maxrss
// column, a monotonic process high-water mark that showed a value only for the
// first op to grow the heap and 0 for every op after — not attributable per op.
func liveHeapKiB(reset, produce func()) int64 {
	reset()
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	for range liveSamples {
		produce()
	}
	runtime.GC()
	runtime.ReadMemStats(&m1)
	reset() // drop the held results so they don't linger across the whole run
	d := max(int64(m1.HeapAlloc)-int64(m0.HeapAlloc), 0)
	return d / int64(liveSamples) / 1024
}

func main() { os.Exit(run()) }

// run is main's body as a function returning an exit code, so every defer
// (sample-data cleanup, StopCPUProfile) runs on a normal return instead of being
// skipped by an os.Exit in an error path.
func run() int {
	datapath := flag.String("datapath", "", "path to a local clone of github.com/lkarlslund/adalanche-sampledata (default: download to a temp dir and clean up afterwards)")
	iters := flag.Int("iters", 200, "iterations per measured operation")
	cpuprofile := flag.String("cpuprofile", "", "write a CPU profile to this file (go tool pprof)")
	memprofile := flag.String("memprofile", "", "write a heap profile to this file after the run (go tool pprof)")
	flag.Parse()

	if *iters < 1 {
		*iters = 1 // benchOp divides by iters; never let it be zero
	}

	// Resolve the localmachine JSON files. With -datapath we use a caller-owned
	// local clone (offline / repeated runs); otherwise we download the
	// sample-data tarball into a temp dir, extract just the dumps, and remove
	// everything when the run ends.
	var glob string
	if *datapath != "" {
		glob = filepath.Join(*datapath, "goad", "localmachine", "*.json")
	} else {
		fmt.Fprintln(os.Stderr, "qdf-bench: downloading adalanche-sampledata to a temp dir…")
		dir, cleanup, err := fetchSampleData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "qdf-bench: %v\n", err)
			return 1
		}
		defer cleanup()
		glob = filepath.Join(dir, "*.json")
	}

	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "qdf-bench: no localmachine JSON files at %s\n", glob)
		return 1
	}

	// Load both representations of every file once.
	typed := make([]*Info, 0, len(files))
	dyn := make([]map[string]any, 0, len(files))
	for _, f := range files {
		ti, err := loadTyped(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", f, err)
			return 1
		}
		mi, err := loadMap(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", f, err)
			return 1
		}
		typed = append(typed, ti)
		dyn = append(dyn, mi)
	}

	fmt.Println("qdf-bench — adalanche localmachine dumps")
	fmt.Printf("build tags : %s\n", buildTagLabel())
	fmt.Printf("files      : %d   iters/op : %d\n\n", len(files), *iters)

	// Start CPU profiling here — after all the os.Exit-able setup (flag parse,
	// data download/load) — so the deferred StopCPUProfile is never skipped by an
	// early exit, and the profile covers only the measured work.
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "qdf-bench: create cpuprofile: %v\n", err)
			return 1
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "qdf-bench: start cpuprofile: %v\n", err)
			return 1
		}
		// StopCPUProfile flushes the profile; close the file after it.
		defer func() { pprof.StopCPUProfile(); f.Close() }()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "repr\tbundle\tdec\tser_ns\tser_B\tser_alloc\tser_liveKiB\tdeser_ns\tdeser_B\tdeser_alloc\tdeser_liveKiB\twire_B")

	// Stats captured for the summaries. matrix holds every bundle (typed + the
	// copy-decode map row, the mode comparable to json/msgpack which have no
	// zero-copy decode) so the matrix-vs-baseline overview can rate every branch;
	// sumQ* hold summaryBundle for the detailed default-vs-both-baselines table.
	var matrix []bundleRow
	var sumQTyped, sumQMap, sumJTyped, sumJMap, sumMTyped, sumMMap stat
	for _, b := range bundles {
		tt := benchTyped(*iters, typed, b.opts)
		printRow(w, "typed", b.name, "copy", tt)
		// Encode is decode-mode-independent: measure it once, reuse across modes.
		ser := benchMapSer(*iters, dyn, b.opts)
		var mapCopy stat
		for _, dm := range mapDecModes {
			deser := benchMapDeser(*iters, dyn, b.opts, dm.build, dm.usesArena)
			row := withDeser(ser, deser)
			printRow(w, "map", b.name, dm.name, row)
			if dm.name == "copy" {
				mapCopy = row
			}
		}
		matrix = append(matrix, bundleRow{b.name, tt, mapCopy})
		if b.name == summaryBundle {
			sumQTyped, sumQMap = tt, mapCopy
		}
	}
	// Reference codecs on the same data: encoding/json and msgpack, so the qdf
	// numbers above can be read relative to a familiar baseline.
	for _, c := range baselines {
		t := benchExtTyped(*iters, c, typed)
		m := benchExtMap(*iters, c, dyn)
		printRow(w, "typed", c.name, "-", t)
		printRow(w, "map", c.name, "-", m)
		switch c.name {
		case "json":
			sumJTyped, sumJMap = t, m
		case "msgpack":
			sumMTyped, sumMMap = t, m
		}
	}
	w.Flush()

	printMatrixVsBaseline(matrix, sumMTyped, sumMMap)
	printSummary(sumQTyped, sumJTyped, sumMTyped, sumQMap, sumJMap, sumMMap)
	printCodegen(*iters, typed)
	printStreaming(*iters, typed, dyn)

	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) == nil {
		fmt.Printf("\nwhole-process peak RSS: %s — this is the WHOLE process, not per op: it holds\n"+
			"  all %d sample files in BOTH representations (typed structs + map[string]any) at\n"+
			"  once, plus the Go runtime baseline and bench machinery. It is NOT a qdf encode/decode\n"+
			"  cost. Per-op memory is ser_B / deser_B (allocated) and ser_liveKiB / deser_liveKiB\n"+
			"  (resident, retained by one result).\n", rssString(ru.Maxrss), len(files))
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "qdf-bench: create memprofile: %v\n", err)
			return 1
		}
		defer f.Close()
		runtime.GC() // settle the heap so the profile reflects live, not garbage
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "qdf-bench: write memprofile: %v\n", err)
		}
	}

	// Anchor the sinks the timed loops wrote to: keeping them alive here means the
	// compiler cannot prove the benchmarked Marshal/Unmarshal results are dead and
	// elide the calls, and the linter sees the package-level sinks as used.
	runtime.KeepAlive(encSink)
	runtime.KeepAlive(decInfo)
	runtime.KeepAlive(decMap)
	return 0
}

// sinks defeat dead-code elimination: the benchmarked qdf call writes its result
// here so the compiler cannot drop it as unused (anchored via runtime.KeepAlive
// at the end of main).
var (
	encSink []byte
	decInfo Info
	decMap  map[string]any
)

// benchTyped measures qdf encode/decode of the typed Info payloads. Each file's
// value is dereferenced ONCE before timing, and the timed loop calls only the
// qdf op (no slice indexing, map lookup, struct copy beyond the value-receiver
// API, or round-trip check) — so ns/B/allocs/RSS are attributable to the
// encode/decode alone. The round-trip equality is checked once, outside timing.
func benchTyped(iters int, vals []*Info, opts qdf.Options) stat {
	var st stat
	st.n = len(vals)
	var lastV Info
	var lastBuf []byte
	for _, p := range vals {
		v := *p

		buf, err := qdf.MarshalT(v, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
			os.Exit(1)
		}
		var rt Info
		if qdf.UnmarshalT(buf, &rt) != nil || !reflect.DeepEqual(v, rt) {
			fmt.Fprintln(os.Stderr, "ROUND-TRIP MISMATCH (typed) — aborting")
			os.Exit(1)
		}
		st.wire += uint64(len(buf))
		lastV, lastBuf = v, buf

		ns, b, al := benchOp(iters, func() { encSink, _ = qdf.MarshalT(v, opts) })
		st.addSer(ns, b, al)

		ns, b, al = benchOp(iters, func() {
			var out Info
			_ = qdf.UnmarshalT(buf, &out)
			decInfo = out
		})
		st.addDeser(ns, b, al)
	}
	st.serLiveKiB = liveHeapKiB(
		func() { keepBuf = nil },
		func() { b, _ := qdf.MarshalT(lastV, opts); keepBuf = append(keepBuf, b) })
	st.deserLiveKiB = liveHeapKiB(
		func() { keepInfo = nil },
		func() { var o Info; _ = qdf.UnmarshalT(lastBuf, &o); keepInfo = append(keepInfo, o) })
	return st
}

// benchMapSer measures qdf encode of the map[string]any payloads (and records
// the wire size). Encode is independent of the decode mode, so it is measured
// ONCE per bundle and the result reused across all decode modes — this both
// avoids redundant work and guarantees the ser_* / wire_B columns are identical
// across a bundle's decode rows by construction. The round-trip gate uses a
// plain copy decode, independent of the decode mode under test.
func benchMapSer(iters int, vals []map[string]any, opts qdf.Options) stat {
	var st stat
	st.n = len(vals)
	var lastV map[string]any
	for _, v := range vals {
		buf, err := qdf.Marshal(v, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
			os.Exit(1)
		}
		var rt map[string]any
		if qdf.Unmarshal(buf, &rt) != nil || !reflect.DeepEqual(v, rt) {
			fmt.Fprintln(os.Stderr, "ROUND-TRIP MISMATCH (map) — aborting")
			os.Exit(1)
		}
		st.wire += uint64(len(buf))
		lastV = v

		ns, b, al := benchOp(iters, func() { encSink, _ = qdf.Marshal(v, opts) })
		st.addSer(ns, b, al)
	}
	st.serLiveKiB = liveHeapKiB(
		func() { keepBuf = nil },
		func() { b, _ := qdf.Marshal(lastV, opts); keepBuf = append(keepBuf, b) })
	return st
}

// benchMapDeser measures qdf decode of the map[string]any payloads under one
// decode mode. build selects the decode QueryOption (nil => plain copy decode);
// for the arena mode a single arena is reused and reset per op so the timing
// models per-message reuse rather than unbounded growth across the loop. The
// input buffer is produced once per file (untimed); the timed loop runs only the
// qdf.Unmarshal call, with no branch or option construction inside it.
func benchMapDeser(iters int, vals []map[string]any, opts qdf.Options, build func(*qdf.Arena) qdf.QueryOption, usesArena bool) stat {
	var st stat
	st.n = len(vals)

	var arena *qdf.Arena
	if usesArena {
		arena = qdf.NewArena()
	}

	var lastBuf []byte
	for _, v := range vals {
		buf, err := qdf.Marshal(v, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
			os.Exit(1)
		}
		lastBuf = buf

		var decFn func()
		switch {
		case build == nil:
			decFn = func() {
				var out map[string]any
				_ = qdf.Unmarshal(buf, &out)
				decMap = out
			}
		case arena != nil:
			qopt := build(arena)
			decFn = func() {
				var out map[string]any
				arena.Reset()
				_ = qdf.Unmarshal(buf, &out, qopt)
				decMap = out
			}
		default:
			qopt := build(nil)
			decFn = func() {
				var out map[string]any
				_ = qdf.Unmarshal(buf, &out, qopt)
				decMap = out
			}
		}
		ns, b, al := benchOp(iters, decFn)
		st.addDeser(ns, b, al)
	}

	// Live heap of one decoded result. For the arena mode, decode all samples
	// into a FRESH un-reset arena and keep every map alive: the arena then holds
	// every sample's strings, so the amortized figure INCLUDES the arena's
	// resident cost (the memory it holds that participates in the decode) — the
	// piece the old metric missed. copy keeps full copied results; nocopy keeps
	// maps that alias the single input buffer (lower live).
	var liveArena *qdf.Arena
	st.deserLiveKiB = liveHeapKiB(
		func() {
			keepMap = nil
			if usesArena {
				liveArena = qdf.NewArena()
			}
		},
		func() {
			var out map[string]any
			if build == nil {
				_ = qdf.Unmarshal(lastBuf, &out)
			} else {
				_ = qdf.Unmarshal(lastBuf, &out, build(liveArena))
			}
			keepMap = append(keepMap, out)
		})
	return st
}

// withDeser returns a copy of the encode-side stat ser with the decode-side
// fields filled from deser, so a single row carries both halves.
func withDeser(ser, deser stat) stat {
	ser.deserNs = deser.deserNs
	ser.deserB = deser.deserB
	ser.deserAllocs = deser.deserAllocs
	ser.deserLiveKiB = deser.deserLiveKiB
	return ser
}

// timingRounds is how many times benchOp re-times the iters loop. The reported
// ns/op is the MINIMUM across rounds: GC pauses and scheduler preemption can
// only ADD wall-clock to a round, so the fastest round is the cleanest estimate
// of the operation's own CPU cost — without disabling GC (which would understate
// the GC-bound decode path) or contaminating the figure with one-off noise.
const timingRounds = 3

// maxWarmupIters caps the unmeasured warmup pass. A handful of ops is enough to
// grow the encoder/decoder pools to their high-water buffer and settle the heap
// target; warming for the full iters would multiply the run time of the
// expensive bundles (rANS/FSST/Compression encode at ~10 ms/op) for no extra
// measurement quality.
const maxWarmupIters = 16

// benchOp measures fn attributed strictly to the marshal/unmarshal call, with
// the per-op work isolated three ways:
//
//   - A warmup pass primes the encoder/decoder pools and grows the heap to
//     steady state, so first-call lazy costs never land in the measured window.
//   - ns/op is the minimum over timingRounds re-timed loops (see above).
//   - B/op and allocs/op come from runtime.MemStats deltas across one GC-settled
//     round (deterministic per op, GC-independent — the same quantities as
//     testing's B/op and allocs/op).
//
// Live resident memory is measured separately, per result, by liveHeapKiB — not
// here, because a per-op resident figure is not meaningful inside a tight loop.
//
// fn must do nothing but the timed op (no option construction, slice indexing,
// or round-trip check) so every count is attributable to encode/decode alone.
func benchOp(iters int, fn func()) (nsPerOp int64, bPerOp, allocsPerOp uint64) {
	// Warmup: prime pools and grow the heap so the measured window sees only
	// steady-state cost. A small capped pass suffices (see maxWarmupIters).
	warmup := min(iters, maxWarmupIters)
	for range warmup {
		fn()
	}

	nsPerOp = -1
	for r := range timingRounds {
		// Measure allocs/B on the first round only: they are deterministic per
		// op, so one GC-settled window is exact, and folding it into the timing
		// loop avoids a separate iters pass.
		var m0, m1 runtime.MemStats
		if r == 0 {
			runtime.GC()
			runtime.ReadMemStats(&m0)
		}

		t0 := time.Now()
		for range iters {
			fn()
		}
		ns := time.Since(t0).Nanoseconds() / int64(iters)
		if nsPerOp < 0 || ns < nsPerOp {
			nsPerOp = ns
		}

		if r == 0 {
			runtime.ReadMemStats(&m1)
			bPerOp = (m1.TotalAlloc - m0.TotalAlloc) / uint64(iters)
			allocsPerOp = (m1.Mallocs - m0.Mallocs) / uint64(iters)
		}
	}
	return nsPerOp, bPerOp, allocsPerOp
}

func printRow(w *tabwriter.Writer, repr, bundle, dec string, s stat) {
	n := uint64(s.n)
	if n == 0 {
		n = 1
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		repr, bundle, dec,
		s.serNs/int64(s.n), s.serB/n, s.serAllocs/n, s.serLiveKiB,
		s.deserNs/int64(s.n), s.deserB/n, s.deserAllocs/n, s.deserLiveKiB,
		s.wire/n)
}

func rssString(maxrss int64) string {
	// Linux reports KiB, darwin reports bytes.
	bytes := maxrss
	if runtime.GOOS == "linux" {
		bytes = maxrss * 1024
	}
	return fmt.Sprintf("%.1f MiB", float64(bytes)/(1<<20))
}

// avgU returns sum/n (per-op average), guarding n==0.
func avgU(sum uint64, n int) uint64 {
	if n <= 0 {
		return sum
	}
	return sum / uint64(n)
}

// ratio formats baseline/qdf for a lower-is-better metric: >1 means qdf is
// better (the baseline costs that many × more), <1 means qdf is WORSE and is
// flagged so the regression is impossible to miss.
func ratio(base, qval uint64) string {
	if qval == 0 {
		return "n/a"
	}
	r := float64(base) / float64(qval)
	if r < 1.0 {
		return fmt.Sprintf("%.2f× WORSE", r)
	}
	return fmt.Sprintf("%.2f×", r)
}

// printMatrixVsBaseline rates EVERY option bundle against msgpack — the stronger
// baseline (json is larger and slower across the board) — so the whole qdf matrix
// can be read against a familiar reference at a glance, not just the default
// bundle. Each cell is msgpack/qdf for a lower-is-better metric: >1 means that
// bundle beats msgpack by that factor, <1 is flagged WORSE.
func printMatrixVsBaseline(matrix []bundleRow, mT, mM stat) {
	fmt.Printf("\n=== qdf MATRIX vs msgpack — every option bundle rated against msgpack\n" +
		"    (the stronger baseline; json loses on every metric). ratio = msgpack / qdf\n" +
		"    (>1 ⇒ qdf better by that factor; <1 ⇒ qdf WORSE) ===\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "repr\tbundle\twire\tser_ns\tser_alloc\tdeser_ns\tdeser_alloc")
	row := func(repr, name string, q, base stat) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", repr, name,
			ratio(avgU(base.wire, base.n), avgU(q.wire, q.n)),
			ratio(avgU(uint64(base.serNs), base.n), avgU(uint64(q.serNs), q.n)),
			ratio(avgU(base.serAllocs, base.n), avgU(q.serAllocs, q.n)),
			ratio(avgU(uint64(base.deserNs), base.n), avgU(uint64(q.deserNs), q.n)),
			ratio(avgU(base.deserAllocs, base.n), avgU(q.deserAllocs, q.n)))
	}
	for _, b := range matrix {
		row("typed", b.name, b.typed, mT)
	}
	for _, b := range matrix {
		row("map", b.name, b.mapCopy, mM)
	}
	w.Flush()
}

// printSummary prints the headline qdf(summaryBundle)-vs-json-vs-msgpack diff for
// both representations: every main lower-is-better metric with the qdf advantage
// (or regression) against each baseline made explicit.
func printSummary(qT, jT, mT, qM, jM, mM stat) {
	fmt.Printf("\n=== SUMMARY: qdf(%s) vs encoding/json vs msgpack — ratio = baseline / qdf\n"+
		"    (>1 ⇒ qdf better by that factor; <1 ⇒ qdf WORSE) ===\n", summaryBundle)

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "repr\tmetric\tqdf\tjson\tmsgpack\tqdf/json\tqdf/msgpack")

	emit := func(repr string, q, j, m stat) {
		type row struct {
			name       string
			q, j, mval uint64
		}
		rows := []row{
			{"ser_ns", avgU(uint64(q.serNs), q.n), avgU(uint64(j.serNs), j.n), avgU(uint64(m.serNs), m.n)},
			{"ser_B", avgU(q.serB, q.n), avgU(j.serB, j.n), avgU(m.serB, m.n)},
			{"ser_alloc", avgU(q.serAllocs, q.n), avgU(j.serAllocs, j.n), avgU(m.serAllocs, m.n)},
			{"ser_liveKiB", uint64(q.serLiveKiB), uint64(j.serLiveKiB), uint64(m.serLiveKiB)},
			{"deser_ns", avgU(uint64(q.deserNs), q.n), avgU(uint64(j.deserNs), j.n), avgU(uint64(m.deserNs), m.n)},
			{"deser_B", avgU(q.deserB, q.n), avgU(j.deserB, j.n), avgU(m.deserB, m.n)},
			{"deser_alloc", avgU(q.deserAllocs, q.n), avgU(j.deserAllocs, j.n), avgU(m.deserAllocs, m.n)},
			{"deser_liveKiB", uint64(q.deserLiveKiB), uint64(j.deserLiveKiB), uint64(m.deserLiveKiB)},
			{"wire_B", avgU(q.wire, q.n), avgU(j.wire, j.n), avgU(m.wire, m.n)},
		}
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\t%s\n",
				repr, r.name, r.q, r.j, r.mval, ratio(r.j, r.q), ratio(r.mval, r.q))
		}
	}
	emit("typed", qT, jT, mT)
	emit("map", qM, jM, mM)
	w.Flush()
}
