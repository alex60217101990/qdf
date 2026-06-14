package bench

import (
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkOTLP_4x64 runs the full codec matrix on 4 resources × 64 spans/scope.
func BenchmarkOTLP_4x64(b *testing.B) {
	v := mkOTLPBatch(4, 64)
	runCodecMatrix(b, v, func() *TraceExport { return new(TraceExport) })
}

// BenchmarkOTLP_4x512 runs the full codec matrix on 4 resources × 512 spans/scope.
func BenchmarkOTLP_4x512(b *testing.B) {
	v := mkOTLPBatch(4, 512)
	runCodecMatrix(b, v, func() *TraceExport { return new(TraceExport) })
}

// TestOTLP_Roundtrip verifies that mkOTLPBatch payloads survive a
// Marshal → Unmarshal round-trip under each qdf tier.
func TestOTLP_Roundtrip(t *testing.T) {
	batch := mkOTLPBatch(2, 16)

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
		t.Run(tier.name, func(t *testing.T) {
			data, err := qdf.Marshal(batch, tier.opts)
			if err != nil {
				t.Fatalf("Marshal(%s): %v", tier.name, err)
			}
			var got TraceExport
			if err := qdf.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tier.name, err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Fatalf("%s: round-trip mismatch", tier.name)
			}
			totalSpans := 0
			for _, rs := range got.ResourceSpans {
				for _, sc := range rs.Scopes {
					totalSpans += len(sc.Spans)
				}
			}
			t.Logf("%s: wire=%d bytes, %d resources, %d spans OK",
				tier.name, len(data), len(got.ResourceSpans), totalSpans)
		})
	}
}
