// Command qdf-bench measures representative qdf serialize/deserialize
// performance over the adalanche-sampledata local-machine dumps
// (github.com/lkarlslund/adalanche-sampledata). For each option bundle it reports
// ser/deser ns/op, B/op, and allocs/op (averaged over the sample files), the wire
// size, for two payload representations — typed Go structs and a dynamic
// map[string]any — and verifies a lossless round trip before timing.
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
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/alex60217101990/qdf"
)

var bundles = []struct {
	name string
	opts qdf.Options
}{
	{"Speed", qdf.OptSpeed},
	{"Balanced", qdf.OptBalanced},
	{"Bal+ColIndex", qdf.OptBalanced | qdf.OptColumnIndex},
	{"Bal+MapShape", qdf.OptBalanced | qdf.OptMapShape},
	{"Bal+Canonical", qdf.OptBalanced | qdf.OptCanonical},
	{"Compression", qdf.OptCompression},
	{"Comp+ColIndex", qdf.OptCompression | qdf.OptColumnIndex},
}

// stat accumulates per-(repr,bundle) results across the sample files. Time,
// bytes and allocs are summed (averaged on print); RSS growth is kept as the
// peak (max) observed during any single op loop, since resident-set growth is a
// high-water mark, not an additive per-op quantity.
type stat struct {
	serNs, deserNs         int64
	serB, deserB           uint64
	serAllocs, deserAllocs uint64
	serRSSKiB, deserRSSKiB int64 // peak process-RSS growth measured AROUND just that op's loop
	wire                   uint64
	n                      int
}

func (s *stat) addSer(ns int64, b, allocs uint64, rssKiB int64) {
	s.serNs += ns
	s.serB += b
	s.serAllocs += allocs
	s.serRSSKiB = max(s.serRSSKiB, rssKiB)
}

func (s *stat) addDeser(ns int64, b, allocs uint64, rssKiB int64) {
	s.deserNs += ns
	s.deserB += b
	s.deserAllocs += allocs
	s.deserRSSKiB = max(s.deserRSSKiB, rssKiB)
}

func main() {
	datapath := flag.String("datapath", "", "path to a clone of github.com/lkarlslund/adalanche-sampledata")
	iters := flag.Int("iters", 200, "iterations per measured operation")
	flag.Parse()

	if *datapath == "" {
		fmt.Fprintln(os.Stderr, `qdf-bench: -datapath is required.

Clone the sample data first:
  git clone https://github.com/lkarlslund/adalanche-sampledata
then:
  qdf-bench -datapath ./adalanche-sampledata`)
		os.Exit(2)
	}

	glob := filepath.Join(*datapath, "goad", "localmachine", "*.json")
	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "qdf-bench: no localmachine JSON files at %s\n", glob)
		os.Exit(1)
	}

	// Load both representations of every file once.
	typed := make([]*Info, 0, len(files))
	dyn := make([]map[string]any, 0, len(files))
	for _, f := range files {
		ti, err := loadTyped(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", f, err)
			os.Exit(1)
		}
		mi, err := loadMap(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", f, err)
			os.Exit(1)
		}
		typed = append(typed, ti)
		dyn = append(dyn, mi)
	}

	fmt.Printf("qdf-bench — adalanche localmachine dumps\n")
	fmt.Printf("build tags : %s\n", buildTagLabel())
	fmt.Printf("files      : %d   iters/op : %d\n\n", len(files), *iters)

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "repr\tbundle\tser_ns\tser_B\tser_alloc\tser_rssKiB\tdeser_ns\tdeser_B\tdeser_alloc\tdeser_rssKiB\twire_B")

	for _, b := range bundles {
		printRow(w, "typed", b.name, benchTyped(*iters, typed, b.opts))
		printRow(w, "map", b.name, benchMap(*iters, dyn, b.opts))
	}
	w.Flush()

	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) == nil {
		fmt.Printf("\nwhole-process peak RSS: %s (loading + both reprs + all bundles — context only;\n"+
			"  the ser_rssKiB / deser_rssKiB columns are scoped to just the encode / decode loop,\n"+
			"  and the precise per-op memory is ser_B / deser_B)\n", rssString(ru.Maxrss))
	}
}

// sinks defeat dead-code elimination: the benchmarked qdf call writes its result
// here so the compiler cannot drop it as unused.
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

		ns, b, al, rss := benchOp(iters, func() { encSink, _ = qdf.MarshalT(v, opts) })
		st.addSer(ns, b, al, rss)

		ns, b, al, rss = benchOp(iters, func() {
			var out Info
			_ = qdf.UnmarshalT(buf, &out)
			decInfo = out
		})
		st.addDeser(ns, b, al, rss)
	}
	return st
}

// benchMap is benchTyped's map[string]any counterpart, with the same isolation:
// the timed loop runs only the qdf op.
func benchMap(iters int, vals []map[string]any, opts qdf.Options) stat {
	var st stat
	st.n = len(vals)
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

		ns, b, al, rss := benchOp(iters, func() { encSink, _ = qdf.Marshal(v, opts) })
		st.addSer(ns, b, al, rss)

		ns, b, al, rss = benchOp(iters, func() {
			var out map[string]any
			_ = qdf.Unmarshal(buf, &out)
			decMap = out
		})
		st.addDeser(ns, b, al, rss)
	}
	return st
}

// benchOp times fn over iters runs and returns ns/op plus bytes and allocations
// per op (from MemStats deltas, matching testing's B/op and allocs/op) and the
// process-RSS growth observed AROUND THIS LOOP ONLY (getrusage Maxrss delta in
// KiB). The RSS figure is scoped to the marshal/unmarshal loop — not the whole
// program — but is still a process-level high-water mark, so it is coarse: the
// precise per-op memory is bPerOp.
func benchOp(iters int, fn func()) (nsPerOp int64, bPerOp, allocsPerOp uint64, rssGrowthKiB int64) {
	var m0, m1 runtime.MemStats
	var ru0, ru1 syscall.Rusage
	runtime.GC()
	runtime.ReadMemStats(&m0)
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru0)
	t0 := time.Now()
	for range iters {
		fn()
	}
	nsPerOp = time.Since(t0).Nanoseconds() / int64(iters)
	runtime.ReadMemStats(&m1)
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru1)
	bPerOp = (m1.TotalAlloc - m0.TotalAlloc) / uint64(iters)
	allocsPerOp = (m1.Mallocs - m0.Mallocs) / uint64(iters)
	rssGrowthKiB = maxrssDeltaKiB(ru0.Maxrss, ru1.Maxrss)
	return
}

// maxrssDeltaKiB normalizes the getrusage Maxrss delta to KiB (Linux reports
// KiB, darwin reports bytes), clamped at 0.
func maxrssDeltaKiB(before, after int64) int64 {
	d := after - before
	if d < 0 {
		return 0
	}
	if runtime.GOOS == "linux" {
		return d // already KiB
	}
	return d / 1024 // darwin: bytes -> KiB
}

func printRow(w *tabwriter.Writer, repr, bundle string, s stat) {
	n := uint64(s.n)
	if n == 0 {
		n = 1
	}
	fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		repr, bundle,
		s.serNs/int64(s.n), s.serB/n, s.serAllocs/n, s.serRSSKiB,
		s.deserNs/int64(s.n), s.deserB/n, s.deserAllocs/n, s.deserRSSKiB,
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
