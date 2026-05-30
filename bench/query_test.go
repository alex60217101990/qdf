package bench

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

type wideRow struct {
	A   int64  `qdf:"a"`
	B   int64  `qdf:"b"`
	C   int64  `qdf:"c"`
	D   int64  `qdf:"d"`
	E   int64  `qdf:"e"`
	F   int64  `qdf:"f"`
	G   int64  `qdf:"g"`
	H   int64  `qdf:"h"`
	Lvl string `qdf:"lvl"`
}

func mkWide(n int, hitMod int) []wideRow {
	out := make([]wideRow, n)
	for i := range out {
		out[i] = wideRow{
			A: int64(i), B: int64(i * 2), C: int64(i * 3), D: int64(i),
			E: int64(i), F: int64(i), G: int64(i), H: int64(i),
		}
		if hitMod > 0 && i%hitMod == 0 {
			out[i].Lvl = "ERROR"
		} else {
			out[i].Lvl = "INFO"
		}
	}
	return out
}

func benchQuerySel(b *testing.B, hitMod int) {
	rows := mkWide(2000, hitMod)
	enc, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnIndex)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(enc)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var got []struct {
			A int64 `qdf:"a"`
			B int64 `qdf:"b"`
		}
		if err := qdf.Unmarshal(enc, &got, qdf.Where("lvl", func(s string) bool { return s == "ERROR" })); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery_Selectivity(b *testing.B) {
	b.Run("hit_1pct", func(b *testing.B) { benchQuerySel(b, 100) })
	b.Run("hit_50pct", func(b *testing.B) { benchQuerySel(b, 2) })
	b.Run("hit_100pct", func(b *testing.B) { benchQuerySel(b, 1) })
}

// BenchmarkQuery_VsFullManual compares pushdown against full decode + manual filter.
func BenchmarkQuery_VsFullManual(b *testing.B) {
	rows := mkWide(2000, 100)
	enc, _ := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnIndex)
	b.Run("pushdown", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var got []struct {
				A int64 `qdf:"a"`
				B int64 `qdf:"b"`
			}
			_ = qdf.Unmarshal(enc, &got, qdf.Where("lvl", func(s string) bool { return s == "ERROR" }))
		}
	})
	b.Run("full_then_filter", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			var all []wideRow
			_ = qdf.Unmarshal(enc, &all)
			var got []wideRow
			for _, r := range all {
				if r.Lvl == "ERROR" {
					got = append(got, r)
				}
			}
			_ = got
		}
	})
}
