package bench

import (
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkIoT_32x256 runs the full codec matrix on 32 devices × 256 samples.
func BenchmarkIoT_32x256(b *testing.B) {
	v := mkIoTBatch(32, 256)
	runCodecMatrix(b, v, func() *IoTBatch { return new(IoTBatch) })
}

// BenchmarkIoT_128x512 runs the full codec matrix on 128 devices × 512 samples.
func BenchmarkIoT_128x512(b *testing.B) {
	v := mkIoTBatch(128, 512)
	runCodecMatrix(b, v, func() *IoTBatch { return new(IoTBatch) })
}

// TestIoT_Roundtrip verifies that mkIoTBatch payloads survive a
// Marshal → Unmarshal round-trip under each qdf tier.
func TestIoT_Roundtrip(t *testing.T) {
	batch := mkIoTBatch(4, 32)

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
			var got IoTBatch
			if err := qdf.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tier.name, err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Fatalf("%s: round-trip mismatch", tier.name)
			}
			t.Logf("%s: wire=%d bytes, %d devices OK", tier.name, len(data), len(got.Devices))
		})
	}
}
