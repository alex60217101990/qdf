package bench

import (
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkLogs_1024 runs the full codec matrix on a 1024-entry log batch.
func BenchmarkLogs_1024(b *testing.B) {
	v := mkLogBatch(1024)
	runCodecMatrix(b, v, func() *LogBatchLE { return new(LogBatchLE) })
}

// BenchmarkEvents_1024 runs the full codec matrix on a 1024-entry event batch.
func BenchmarkEvents_1024(b *testing.B) {
	v := mkEventBatch(1024)
	runCodecMatrix(b, v, func() *EventBatch { return new(EventBatch) })
}

// TestLogs_Roundtrip verifies that mkLogBatch payloads survive a
// Marshal → Unmarshal round-trip under each qdf tier.
func TestLogs_Roundtrip(t *testing.T) {
	batch := mkLogBatch(64)

	tiers := []struct {
		name string
		opts qdf.Options
	}{
		{"speed", qdf.OptSpeed},
		{"balanced", qdf.OptBalanced},
		{"qpack", qdf.OptQPack},
		{"compression", qdf.OptCompression},
	}

	for _, tier := range tiers {
		tier := tier
		t.Run(tier.name, func(t *testing.T) {
			data, err := qdf.Marshal(batch, tier.opts)
			if err != nil {
				t.Fatalf("Marshal(%s): %v", tier.name, err)
			}
			var got LogBatchLE
			if err := qdf.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tier.name, err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Fatalf("%s: round-trip mismatch", tier.name)
			}
			t.Logf("%s: wire=%d bytes, %d records OK", tier.name, len(data), len(got.Records))
		})
	}
}

// TestEvents_Roundtrip verifies that mkEventBatch payloads survive a
// Marshal → Unmarshal round-trip under each qdf tier.
func TestEvents_Roundtrip(t *testing.T) {
	batch := mkEventBatch(64)

	tiers := []struct {
		name string
		opts qdf.Options
	}{
		{"speed", qdf.OptSpeed},
		{"balanced", qdf.OptBalanced},
		{"qpack", qdf.OptQPack},
		{"compression", qdf.OptCompression},
	}

	for _, tier := range tiers {
		tier := tier
		t.Run(tier.name, func(t *testing.T) {
			data, err := qdf.Marshal(batch, tier.opts)
			if err != nil {
				t.Fatalf("Marshal(%s): %v", tier.name, err)
			}
			var got EventBatch
			if err := qdf.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tier.name, err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Fatalf("%s: round-trip mismatch", tier.name)
			}
			t.Logf("%s: wire=%d bytes, %d records OK", tier.name, len(data), len(got.Records))
		})
	}
}
