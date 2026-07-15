package qdf

import "testing"

func benchDeltaRec() ([]fuzzRec, []fuzzRec) {
	old := make([]fuzzRec, 1000)
	neu := make([]fuzzRec, 1000)
	for i := range old {
		old[i] = fuzzRec{ID: i, Name: "node", Score: 1.5,
			Items: []fuzzInner{{A: 1, B: "x"}}}
		neu[i] = old[i]
		neu[i].Items = append([]fuzzInner(nil), old[i].Items...) // independent backing
	}
	neu[500].Score = 99.9 // exactly one field in one element changes
	return old, neu
}

// BenchmarkDiffMap measures Diff on a canonical (OptCanonical) map[string]int,
// which exercises canonSortedMapKeys twice per call (once for updates, once for
// deletions). With 16 keys the old code allocated 16 reflect.Value holders per
// call; the new code allocates 1.
func BenchmarkDiffMap(b *testing.B) {
	old := map[string]int{
		"alpha": 1, "beta": 2, "gamma": 3, "delta": 4,
		"epsilon": 5, "zeta": 6, "eta": 7, "theta": 8,
		"iota": 9, "kappa": 10, "lambda": 11, "mu": 12,
		"nu": 13, "xi": 14, "omicron": 15, "pi": 16,
	}
	neu := make(map[string]int, len(old))
	for k, v := range old {
		neu[k] = v
	}
	neu["mu"] = 99 // one update — triggers canonSortedMapKeys for the full key set
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Diff(old, neu, OptCanonical|OptDense)
	}
}

func BenchmarkDiffApply(b *testing.B) {
	old, neu := benchDeltaRec()
	b.Run("Diff", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Diff(old, neu, OptBalanced)
		}
	})
	patch, _ := Diff(old, neu, OptBalanced)
	b.Run("Apply", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			base := make([]fuzzRec, len(old))
			copy(base, old)
			_ = Apply(&base, patch)
		}
	})
	b.Run("FullMarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = Marshal(neu, OptBalanced)
		}
	})
	full, _ := Marshal(neu, OptBalanced)
	b.Logf("patch size = %d bytes; full marshal = %d bytes (%.1fx smaller)",
		len(patch), len(full), float64(len(full))/float64(len(patch)))

	// Apply with the base fingerprint skipped: Apply no longer walks the whole
	// (large) base via reflect, only the tiny patch.
	patchNoFP, _ := Diff(old, neu, OptBalanced|OptDeltaNoBaseFingerprint)
	b.Run("ApplyNoBaseFP", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			base := make([]fuzzRec, len(old))
			copy(base, old)
			_ = Apply(&base, patchNoFP)
		}
	})
}
