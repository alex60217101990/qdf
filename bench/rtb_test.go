package bench

import (
	"reflect"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkRTB_64 runs the full codec matrix (json/msgpack/qdf×4 tiers,
// encode+decode) on a batch of 64 OpenRTB-style bid requests.
func BenchmarkRTB_64(b *testing.B) {
	runCodecMatrix(b, mkRTBBatch(64), func() *[]BidRequest { return new([]BidRequest) })
}

// BenchmarkRTB_1024 runs the full codec matrix on a batch of 1024 bid requests.
func BenchmarkRTB_1024(b *testing.B) {
	runCodecMatrix(b, mkRTBBatch(1024), func() *[]BidRequest { return new([]BidRequest) })
}

// TestRTB_Roundtrip verifies that mkRTBBatch produces payloads that survive a
// Marshal → Unmarshal round-trip under each qdf tier and that the decoded
// value matches the original.
func TestRTB_Roundtrip(t *testing.T) {
	batch := mkRTBBatch(32)

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
			var got []BidRequest
			if err := qdf.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tier.name, err)
			}
			if !reflect.DeepEqual(batch, got) {
				t.Fatalf("%s: round-trip mismatch\nwant len=%d got len=%d",
					tier.name, len(batch), len(got))
			}
			t.Logf("%s: wire=%d bytes, %d requests OK", tier.name, len(data), len(got))
		})
	}
}
