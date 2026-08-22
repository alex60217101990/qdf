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

// BenchmarkRTB_JSONText_1024 is the encode ceiling for JSON on a string-heavy
// payload: the struct walk written out by hand, with the encoder and its buffer
// reused between messages.
//
// This is the shape where qdf_compression loses encode CPU to json/v2, so
// knowing what JSON can actually reach here — rather than what its reflection
// path happens to cost — is the point of the arm.
func BenchmarkRTB_JSONText_1024(b *testing.B) {
	v := mkRTBBatch(1024)
	enc := newJSONTextEncoder()
	wire, err := enc.marshalRTBBatch(v)
	if err != nil {
		b.Fatal(err)
	}
	size := len(wire)

	b.Run("encode/jsontext", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(size))
		e := newJSONTextEncoder()
		for b.Loop() {
			if _, err := e.marshalRTBBatch(v); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(size), "wire-B")
	})
}
