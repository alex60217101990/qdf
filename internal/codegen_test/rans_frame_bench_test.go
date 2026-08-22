package cgsample

import (
	"fmt"
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

func benchRF(b *testing.B, n int, opts qdf.Options) {
	rows := make([]GenRow, n)
	for i := range rows {
		id := fmt.Sprintf("%06d", i)
		rows[i] = GenRow{
			ID: int64(i), Name: "com.acme.platform.worker.service." + id,
			Inner: GenRowInner{X: i, Y: "/opt/acme/platform/bin/worker --shard=" + id},
		}
	}
	set := GenRowSet{Rows: rows}
	wire, err := qdf.Marshal(set, opts)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := qdf.Marshal(set, opts); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var out GenRowSet
			if err := qdf.Unmarshal(wire, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRF64Rans(b *testing.B)   { benchRF(b, 64, qdf.OptBalanced|qdf.OptRANS) }
func BenchmarkRF512Rans(b *testing.B)  { benchRF(b, 512, qdf.OptBalanced|qdf.OptRANS) }
func BenchmarkRF512Compr(b *testing.B) { benchRF(b, 512, qdf.OptCompression) }
func BenchmarkRF512Bal(b *testing.B)   { benchRF(b, 512, qdf.OptBalanced) }
