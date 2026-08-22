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

// BenchmarkIoT_JSONText_32x256 measures the hand-written jsontext codec on the
// same payload the matrix above uses, so its rows can be read beside them.
//
// This arm answers "how fast can JSON go on this shape if you remove the
// reflection entirely" — its honest counterpart is qdf_codegen, not the reflect
// tiers. The encoder and decoder are reused across iterations, which is the
// point: a fresh one per message would put back the allocation this exists to
// remove.
func BenchmarkIoT_JSONText_32x256(b *testing.B) {
	v := mkIoTBatch(32, 256)

	enc := newJSONTextEncoder()
	wire, err := enc.marshalIoTBatch(&v)
	if err != nil {
		b.Fatal(err)
	}
	size := len(wire)
	payload := append([]byte(nil), wire...)

	b.Run("encode/jsontext", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(size))
		e := newJSONTextEncoder()
		for b.Loop() {
			if _, err := e.marshalIoTBatch(&v); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(size), "wire-B")
	})

	b.Run("decode/jsontext", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(size))
		d := newJSONTextDecoder()
		var out IoTBatch
		for b.Loop() {
			if err := d.unmarshalIoTBatch(payload, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
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
