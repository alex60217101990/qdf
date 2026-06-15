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
