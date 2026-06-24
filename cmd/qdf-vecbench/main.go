// Command qdf-vecbench sweeps quality knobs independently for each method,
// collects rate–distortion operating points, writes them to rd.csv, and
// prints a matched-rel-error headline table that interpolates bytes/vector at
// fixed rel-error targets so every method is compared at equal quality.
package main

import (
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/alex60217101990/qdf/internal/vecquant"
)

// hadamardSeed is the fixed rotation seed for TurboQuant-scalar; matches the
// value used in the qdf lossy codec so the comparison is fair.
const tqSeed uint64 = 0x51ed270b9f4d8c3a

// blockOverheadBytes is the fixed on-wire overhead for a qdf Block:
// Dim(8) + Count(8) + Seed(8) + Delta(8) = 32 bytes.
const blockOverheadBytes = 32

// qdfBudgets are the MaxRelError values swept for the qdf-lossy method.
var qdfBudgets = []float64{0.005, 0.01, 0.02, 0.05, 0.10, 0.20, 0.30}

// bitSweep is the set of bit-widths swept for naive and TurboQuant-scalar.
var bitSweep = []int{3, 4, 5, 6, 7, 8, 10}

// pqBitsPerSubSweep and pqSubspacesSweep drive the PQ operating-point grid.
var pqBitsPerSubSweep = []int{4, 6, 8}
var pqSubspacesSweep = []int{4, 8, 16}

// relErrTargets are the fixed rel-error values used for the matched headline.
var relErrTargets = []float64{0.02, 0.05, 0.10}

// row holds the metrics for one (method, knob) operating point.
type row struct {
	method      string
	budget      float64 // knob value (eps for qdf, bits for scalar/PQ)
	bytesPerVec float64
	relErr      float64
	recall10    float64
	encMBs      float64
	decMBs      float64
}

func main() {
	synthetic := flag.Bool("synthetic", false, "always use synthetic Gaussian corpus")
	n := flag.Int("n", 5000, "number of vectors")
	dim := flag.Int("dim", 256, "vector dimension")
	output := flag.String("out", "rd.csv", "CSV output file")
	flag.Parse()

	var corpus [][]float64
	if *synthetic || embURL == "" {
		if !*synthetic {
			log.Print("no pinned corpus configured; using synthetic Gaussian corpus")
		}
		corpus = loadSynthetic(*n, *dim, 42)
	} else {
		tmp := "corpus.bin"
		if err := fetchPinned(tmp); err != nil {
			log.Printf("corpus download failed (%v); falling back to synthetic", err)
			corpus = loadSynthetic(*n, *dim, 42)
		} else {
			var err error
			corpus, err = loadF32File(tmp, *dim)
			if err != nil {
				log.Printf("corpus parse failed (%v); falling back to synthetic", err)
				corpus = loadSynthetic(*n, *dim, 42)
			}
		}
	}

	groundTruth := buildKNN(corpus, 10)

	var rows []row

	// ── qdf-lossy: sweep MaxRelError budgets ──────────────────────────────────
	for _, eps := range qdfBudgets {
		b := vecquant.Budget{Kind: vecquant.KindRelError, Val: eps}

		t0 := time.Now()
		bl := vecquant.Encode(corpus, b)
		encDur := time.Since(t0)

		t1 := time.Now()
		recon := bl.Decode()
		decDur := time.Since(t1)

		// blockOverheadBytes covers the four fixed 8-byte fields
		// (Dim, Count, Seed, Delta); Coords is the variable-length payload.
		wire := len(bl.Coords) + blockOverheadBytes
		bpv := float64(wire) / float64(len(corpus))
		totalBytes := float64(len(corpus)) * float64(*dim) * 8
		encMBs := (totalBytes / 1e6) / encDur.Seconds()
		decMBs := (totalBytes / 1e6) / decDur.Seconds()

		relErr := avgRelError(corpus, recon)
		rec := recall10(corpus, recon, groundTruth)

		rows = append(rows, row{
			method:      "qdf-lossy",
			budget:      eps,
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	// ── Naive scalar: sweep bit-widths ────────────────────────────────────────
	for _, bits := range bitSweep {
		var totalWire int
		recon := make([][]float64, len(corpus))

		t0 := time.Now()
		qs := make([][]uint16, len(corpus))
		mns := make([]float64, len(corpus))
		deltas := make([]float64, len(corpus))
		for i, v := range corpus {
			qs[i], mns[i], deltas[i] = naiveScalarEncode(v, bits)
			totalWire += naiveScalarBytes(qs[i], bits)
		}
		encDur := time.Since(t0)

		t1 := time.Now()
		for i := range corpus {
			recon[i] = naiveScalarDecode(qs[i], mns[i], deltas[i])
		}
		decDur := time.Since(t1)

		bpv := float64(totalWire) / float64(len(corpus))
		totalBytes := float64(len(corpus)) * float64(*dim) * 8
		encMBs := (totalBytes / 1e6) / encDur.Seconds()
		decMBs := (totalBytes / 1e6) / decDur.Seconds()
		relErr := avgRelError(corpus, recon)
		rec := recall10(corpus, recon, groundTruth)

		rows = append(rows, row{
			method:      fmt.Sprintf("naive-%dbit", bits),
			budget:      float64(bits),
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	// ── TurboQuant-scalar: sweep bit-widths ───────────────────────────────────
	for _, bits := range bitSweep {
		pdim := nextPow2(*dim)

		var totalWire int
		recon := make([][]float64, len(corpus))

		t0 := time.Now()
		qs := make([][]uint16, len(corpus))
		mns := make([]float64, len(corpus))
		deltas := make([]float64, len(corpus))
		pdims := make([]int, len(corpus))
		for i, v := range corpus {
			qs[i], mns[i], deltas[i], pdims[i] = turboQuantScalarEncode(v, bits, tqSeed)
			totalWire += turboQuantScalarBytes(qs[i], bits)
		}
		encDur := time.Since(t0)
		_ = pdim

		t1 := time.Now()
		for i := range corpus {
			recon[i] = turboQuantScalarDecode(qs[i], mns[i], deltas[i], pdims[i], *dim, tqSeed)
		}
		decDur := time.Since(t1)

		bpv := float64(totalWire) / float64(len(corpus))
		totalBytes := float64(len(corpus)) * float64(*dim) * 8
		encMBs := (totalBytes / 1e6) / encDur.Seconds()
		decMBs := (totalBytes / 1e6) / decDur.Seconds()
		relErr := avgRelError(corpus, recon)
		rec := recall10(corpus, recon, groundTruth)

		rows = append(rows, row{
			method:      fmt.Sprintf("tq-scalar-%dbit", bits),
			budget:      float64(bits),
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	// ── PQ: sweep (subspaces, bits) grid ─────────────────────────────────────
	// Note: PQ "encode" time includes codebook training, which is a one-time
	// setup cost amortised over the full dataset in real systems; its enc MB/s
	// figure is not directly comparable to per-vector encode throughput.
	for _, subspaces := range pqSubspacesSweep {
		// skip configurations where dim is not evenly divisible
		if (*dim)%subspaces != 0 {
			continue
		}
		for _, bits := range pqBitsPerSubSweep {
			recon := make([][]float64, len(corpus))

			t0 := time.Now()
			books := trainPQ(corpus, subspaces, bits)
			for i, v := range corpus {
				recon[i] = pqReconstruct(v, books, subspaces)
			}
			encDur := time.Since(t0)

			t1 := time.Now()
			for i, v := range corpus {
				recon[i] = pqReconstruct(v, books, subspaces)
			}
			decDur := time.Since(t1)

			bpv := pqBytesPerVector(subspaces, bits)
			totalBytes := float64(len(corpus)) * float64(*dim) * 8
			// enc time includes codebook training (one-time cost)
			encMBs := (totalBytes / 1e6) / encDur.Seconds()
			decMBs := (totalBytes / 1e6) / decDur.Seconds()
			relErr := avgRelError(corpus, recon)
			rec := recall10(corpus, recon, groundTruth)

			rows = append(rows, row{
				method:      fmt.Sprintf("pq-%dsub-%dbit", subspaces, bits),
				budget:      float64(bits),
				bytesPerVec: bpv,
				relErr:      relErr,
				recall10:    rec,
				encMBs:      encMBs,
				decMBs:      decMBs,
			})
		}
	}

	printRDTable(rows)
	printHeadlineTable(rows)

	if err := writeCSV(*output, rows); err != nil {
		log.Printf("csv write failed: %v", err)
	} else {
		fmt.Printf("\nrd.csv written to %s\n", *output)
	}
}

// nextPow2 is a local helper (mirrors hadamard.NextPow2 without importing hadamard directly).
func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// avgRelError returns the mean per-vector relative L2 error.
func avgRelError(orig, recon [][]float64) float64 {
	var sum float64
	for i := range orig {
		var se, ne float64
		for j := range orig[i] {
			d := orig[i][j] - recon[i][j]
			se += d * d
			ne += orig[i][j] * orig[i][j]
		}
		sum += math.Sqrt(se / (ne + 1e-30))
	}
	return sum / float64(len(orig))
}

// knnResult stores the ground-truth top-k neighbours for each query vector.
type knnResult [][]int

// recallSampleCap limits the number of query vectors used for recall@10 to
// keep the O(n²) exact-KNN computation tractable on large corpora.
const recallSampleCap = 500

// buildKNN computes exact top-k nearest neighbours for the first
// min(len(corpus), recallSampleCap) query vectors using L2 distance.
// Limiting the query set keeps the O(n²) cost under control while
// remaining statistically representative for recall estimation.
func buildKNN(corpus [][]float64, k int) knnResult {
	n := len(corpus)
	queries := n
	if queries > recallSampleCap {
		queries = recallSampleCap
	}
	result := make(knnResult, queries)
	for i := 0; i < queries; i++ {
		type idxDist struct {
			idx  int
			dist float64
		}
		dists := make([]idxDist, 0, n-1)
		for j := range corpus {
			if j == i {
				continue
			}
			dists = append(dists, idxDist{j, sqDist(corpus[i], corpus[j])})
		}
		sort.Slice(dists, func(a, b int) bool { return dists[a].dist < dists[b].dist })
		nn := make([]int, k)
		for ki := 0; ki < k && ki < len(dists); ki++ {
			nn[ki] = dists[ki].idx
		}
		result[i] = nn
	}
	return result
}

// recall10 computes the mean recall@10 over the sampled query set (gt).
// For each query i (indices 0..len(gt)-1) it ranks all reconstructed
// vectors by L2 distance, then reports the fraction of the ground-truth
// top-10 neighbours that appear in the reconstructed top-10.
func recall10(orig, recon [][]float64, gt knnResult) float64 {
	k := 10
	var sum float64
	for i, gtNN := range gt {
		type idxDist struct {
			idx  int
			dist float64
		}
		dists := make([]idxDist, 0, len(recon)-1)
		for j := range recon {
			if j == i {
				continue
			}
			dists = append(dists, idxDist{j, sqDist(recon[i], recon[j])})
		}
		sort.Slice(dists, func(a, b int) bool { return dists[a].dist < dists[b].dist })

		gtSet := make(map[int]struct{}, k)
		for _, idx := range gtNN {
			gtSet[idx] = struct{}{}
		}
		var hits int
		for ki := 0; ki < k && ki < len(dists); ki++ {
			if _, ok := gtSet[dists[ki].idx]; ok {
				hits++
			}
		}
		sum += float64(hits) / float64(k)
	}
	return sum / float64(len(gt))
}

// printRDTable prints all raw operating points (the full RD curve data).
func printRDTable(rows []row) {
	fmt.Println("\n── Rate–Distortion operating points (all methods, raw sweep) ──")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "method\tknob\tbytes/vec\trel-err\trecall@10\tenc MB/s\tdec MB/s")
	fmt.Fprintln(w, "------\t----\t---------\t-------\t---------\t--------\t--------")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%.4g\t%.2f\t%.4f\t%.4f\t%.1f\t%.1f\n",
			r.method, r.budget, r.bytesPerVec, r.relErr, r.recall10, r.encMBs, r.decMBs)
	}
	w.Flush()
}

// methodPoints returns all rows for a given method prefix (exact match or prefix).
func methodPoints(rows []row, method string) []row {
	var out []row
	for _, r := range rows {
		if r.method == method {
			out = append(out, r)
		}
	}
	return out
}

// methodPointsPrefix returns all rows whose method name starts with prefix.
func methodPointsPrefix(rows []row, prefix string) []row {
	var out []row
	for _, r := range rows {
		if len(r.method) >= len(prefix) && r.method[:len(prefix)] == prefix {
			out = append(out, r)
		}
	}
	return out
}

// interpBPV linearly interpolates bytes/vec at the target rel-error from a
// set of operating points. Points are sorted by relErr ascending; the two
// bracketing points are used. Returns (bpv, true) if R is bracketed, else
// (0, false) — no extrapolation is performed.
func interpBPV(pts []row, targetRelErr float64) (float64, bool) {
	if len(pts) == 0 {
		return 0, false
	}
	// sort by relErr ascending
	sorted := make([]row, len(pts))
	copy(sorted, pts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].relErr < sorted[j].relErr })

	// find bracketing pair [lo, hi] such that lo.relErr <= target <= hi.relErr
	lo, hi := -1, -1
	for i := 0; i < len(sorted)-1; i++ {
		if sorted[i].relErr <= targetRelErr && sorted[i+1].relErr >= targetRelErr {
			lo, hi = i, i+1
			break
		}
	}
	if lo < 0 {
		return 0, false
	}
	rLo, rHi := sorted[lo].relErr, sorted[hi].relErr
	bLo, bHi := sorted[lo].bytesPerVec, sorted[hi].bytesPerVec
	if rHi == rLo {
		return (bLo + bHi) / 2, true
	}
	t := (targetRelErr - rLo) / (rHi - rLo)
	return bLo + t*(bHi-bLo), true
}

// printHeadlineTable prints the matched-quality comparison: for each target
// rel-error, each method's bytes/vec is interpolated from its own RD curve.
// No extrapolation; missing cells are printed as "n/a".
func printHeadlineTable(rows []row) {
	// collect the distinct method names (preserving first-seen order)
	seen := map[string]struct{}{}
	var methods []string
	for _, r := range rows {
		if _, ok := seen[r.method]; !ok {
			seen[r.method] = struct{}{}
			methods = append(methods, r.method)
		}
	}

	// build per-method point sets
	pts := map[string][]row{}
	for _, m := range methods {
		pts[m] = methodPoints(rows, m)
	}

	fmt.Println("\n── Matched-quality headline (bytes/vec at equal rel-error, interpolated) ──")
	fmt.Println("(PQ enc MB/s includes one-time codebook training; not directly comparable to per-vector enc throughput)")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// header: rel-err | qdf | naive-best | tq-best | pq-best | qdf-vs-best %
	fmt.Fprintf(w, "rel-err\t%s\t%s\t%s\t%s\t%s\n",
		"qdf-lossy", "naive-best", "tq-best", "pq-best", "qdf-vs-best")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		"-------", "---------", "-------", "-------", "-----------")

	for _, target := range relErrTargets {
		qdfBPV, qdfOK := interpBPV(pts["qdf-lossy"], target)

		// naive-best: best (lowest) bytes/vec among all naive-Nbit methods at this target
		naiveBPV, naiveOK := bestInterp(rows, "naive-", pts, target)

		// tq-best
		tqBPV, tqOK := bestInterp(rows, "tq-scalar-", pts, target)

		// pq-best
		pqBPV, pqOK := bestInterp(rows, "pq-", pts, target)

		qdfStr := fmtBPV(qdfBPV, qdfOK)
		naiveStr := fmtBPV(naiveBPV, naiveOK)
		tqStr := fmtBPV(tqBPV, tqOK)
		pqStr := fmtBPV(pqBPV, pqOK)

		// headline %: qdf vs the single best baseline at this target
		vsStr := "n/a"
		if qdfOK {
			bestBaseline, bestOK := minBPV(
				naiveBPV, naiveOK,
				tqBPV, tqOK,
				pqBPV, pqOK,
			)
			if bestOK {
				pct := (bestBaseline - qdfBPV) / bestBaseline * 100
				if pct > 0 {
					vsStr = fmt.Sprintf("qdf −%.1f%%", pct)
				} else if pct < 0 {
					vsStr = fmt.Sprintf("qdf +%.1f%% (larger)", -pct)
				} else {
					vsStr = "equal"
				}
			}
		}

		fmt.Fprintf(w, "%.2f\t%s\t%s\t%s\t%s\t%s\n",
			target, qdfStr, naiveStr, tqStr, pqStr, vsStr)
	}
	w.Flush()
	fmt.Println()
}

// bestInterp finds the minimum interpolated bytes/vec among all methods whose
// name starts with prefix, at the given target rel-error. It collects all
// operating points from every method in the group and interpolates over the
// combined frontier — this gives the best achievable bytes/vec at that quality.
func bestInterp(rows []row, prefix string, _ map[string][]row, target float64) (float64, bool) {
	// Collect all points from methods whose name starts with prefix.
	var combined []row
	for _, r := range rows {
		if len(r.method) >= len(prefix) && r.method[:len(prefix)] == prefix {
			combined = append(combined, r)
		}
	}
	if len(combined) == 0 {
		return 0, false
	}
	// For each target rel-error, the best baseline is the one achieving that
	// quality with the fewest bytes. Interpolate over the Pareto frontier
	// (lowest bytes/vec for each relErr value). Build a reduced set: for each
	// distinct relErr keep the minimum bytes/vec.
	byErr := map[float64]float64{}
	for _, r := range combined {
		if prev, ok := byErr[r.relErr]; !ok || r.bytesPerVec < prev {
			byErr[r.relErr] = r.bytesPerVec
		}
	}
	var frontier []row
	for e, b := range byErr {
		frontier = append(frontier, row{relErr: e, bytesPerVec: b})
	}
	return interpBPV(frontier, target)
}

// minBPV returns the minimum bytes/vec among up to three (value, ok) pairs.
func minBPV(a float64, aOK bool, b float64, bOK bool, c float64, cOK bool) (float64, bool) {
	best := math.Inf(1)
	found := false
	for _, pair := range [][2]float64{{a, boolToFloat(aOK)}, {b, boolToFloat(bOK)}, {c, boolToFloat(cOK)}} {
		if pair[1] > 0 && pair[0] < best {
			best = pair[0]
			found = true
		}
	}
	return best, found
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func fmtBPV(bpv float64, ok bool) string {
	if !ok {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", bpv)
}

func writeCSV(path string, rows []row) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	_ = cw.Write([]string{"method", "knob", "bytes_per_vec", "rel_err", "recall10", "enc_mbs", "dec_mbs"})
	for _, r := range rows {
		_ = cw.Write([]string{
			r.method,
			strconv.FormatFloat(r.budget, 'f', -1, 64),
			strconv.FormatFloat(r.bytesPerVec, 'f', 4, 64),
			strconv.FormatFloat(r.relErr, 'f', 6, 64),
			strconv.FormatFloat(r.recall10, 'f', 4, 64),
			strconv.FormatFloat(r.encMBs, 'f', 2, 64),
			strconv.FormatFloat(r.decMBs, 'f', 2, 64),
		})
	}
	cw.Flush()
	return cw.Error()
}

// loadF32File reads a raw float32 binary file (row-major, no header) into a
// [][]float64 corpus. Vectors with fewer than dim floats remaining are dropped.
func loadF32File(path string, dim int) ([][]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	floatsPerVec := dim
	bytesPerVec := floatsPerVec * 4
	n := len(data) / bytesPerVec
	if n == 0 {
		return nil, fmt.Errorf("loadF32File: file too small for dim=%d", dim)
	}
	out := make([][]float64, n)
	for i := range out {
		v := make([]float64, floatsPerVec)
		off := i * bytesPerVec
		for j := range v {
			bits := binary.LittleEndian.Uint32(data[off+j*4:])
			v[j] = float64(math.Float32frombits(bits))
		}
		out[i] = v
	}
	return out, nil
}
