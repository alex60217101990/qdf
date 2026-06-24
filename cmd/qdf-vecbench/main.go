// Command qdf-vecbench sweeps quality budgets over an embedding corpus,
// encodes each batch with the qdf lossy codec and three Go baselines, and
// prints a rate-distortion comparison table together with recall@10 and
// throughput columns. Results are also written to rd.csv for offline plotting.
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

// budgetSweep defines the rel-error targets used in the sweep.
var budgetSweep = []float64{0.30, 0.20, 0.10, 0.05, 0.02}

// bitSweep maps a target rel-error to a fixed bit-width for the scalar
// baselines (naive + TurboQuant-scalar) so the sweep remains comparable.
func bitsForRelError(rel float64) int {
	switch {
	case rel >= 0.25:
		return 4
	case rel >= 0.15:
		return 5
	case rel >= 0.08:
		return 6
	case rel >= 0.03:
		return 7
	default:
		return 8
	}
}

// row holds the metrics for one (method, budget) point.
type row struct {
	method      string
	budget      float64
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

	// qdf lossy codec
	for _, eps := range budgetSweep {
		b := vecquant.Budget{Kind: vecquant.KindRelError, Val: eps}

		t0 := time.Now()
		bl := vecquant.Encode(corpus, b)
		encDur := time.Since(t0)

		t1 := time.Now()
		recon := bl.Decode()
		decDur := time.Since(t1)

		wire := len(bl.Coords) + 8 + 8 + 8 // Seed(8)+Dim(8)+Delta(8) overhead approx
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

	// Naive scalar baseline
	for _, eps := range budgetSweep {
		bits := bitsForRelError(eps)

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
			budget:      eps,
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	// TurboQuant-scalar baseline (Hadamard rotation + scalar quant, no entropy)
	for _, eps := range budgetSweep {
		bits := bitsForRelError(eps)
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
			budget:      eps,
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	// PQ baseline — fixed 8 subspaces, bit-width mapped from rel-error target.
	// PQ codebook training is amortised; bytes/vector reflects codes only.
	subspaces := 8
	for _, eps := range budgetSweep {
		bits := bitsForRelError(eps)
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
		// enc cost includes training; dec cost is lookup-only
		encMBs := (totalBytes / 1e6) / encDur.Seconds()
		decMBs := (totalBytes / 1e6) / decDur.Seconds()
		relErr := avgRelError(corpus, recon)
		rec := recall10(corpus, recon, groundTruth)

		rows = append(rows, row{
			method:      fmt.Sprintf("pq-%dsub-%dbit", subspaces, bits),
			budget:      eps,
			bytesPerVec: bpv,
			relErr:      relErr,
			recall10:    rec,
			encMBs:      encMBs,
			decMBs:      decMBs,
		})
	}

	printTable(rows)
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

func printTable(rows []row) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "method\tbudget\tbytes/vec\trel-err\trecall@10\tenc MB/s\tdec MB/s")
	fmt.Fprintln(w, "------\t------\t---------\t-------\t---------\t--------\t--------")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.4f\t%.4f\t%.1f\t%.1f\n",
			r.method, r.budget, r.bytesPerVec, r.relErr, r.recall10, r.encMBs, r.decMBs)
	}
	w.Flush()
}

func writeCSV(path string, rows []row) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	_ = cw.Write([]string{"method", "budget", "bytes_per_vec", "rel_err", "recall10", "enc_mbs", "dec_mbs"})
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
