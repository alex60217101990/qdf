package bench

import (
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkAccess_1024 runs the full codec matrix on a 1024-entry access-log
// batch — the free-text shape the other payloads do not cover.
func BenchmarkAccess_1024(b *testing.B) {
	v := mkAccessBatch(1024)
	runCodecMatrix(b, v, func() *AccessBatch { return new(AccessBatch) })
}

// BenchmarkAccess_8192 is the same shape at batch scale, where a shared
// substring table has enough rows to amortise.
func BenchmarkAccess_8192(b *testing.B) {
	v := mkAccessBatch(8192)
	runCodecMatrix(b, v, func() *AccessBatch { return new(AccessBatch) })
}

// TestAccess_Roundtrip verifies that mkAccessBatch payloads survive a
// Marshal → Unmarshal round-trip under each qdf tier.
func TestAccess_Roundtrip(t *testing.T) {
	batch := mkAccessBatch(64)

	tiers := []struct {
		name string
		opts qdf.Options
	}{
		{"speed", qdf.OptSpeed},
		{"balanced", qdf.OptBalanced},
		{"qpack", qdf.OptQPack},
		{"compression", qdf.OptCompression},
		{"balanced_fsst", qdf.OptBalanced | qdf.OptFSST},
	}
	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			blob, err := qdf.Marshal(batch, tier.opts)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got AccessBatch
			if err := qdf.Unmarshal(blob, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Errorf("round-trip mismatch")
			}
		})
	}
}
