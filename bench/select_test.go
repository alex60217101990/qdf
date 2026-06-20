package bench

import (
	"testing"

	qdf "github.com/alex60217101990/qdf"
)

// BenchmarkSelect_FullVsSubset proves the selective columnar decode win:
// a payload Marshaled from a 16-field wide struct with OptColumnIndex can
// be decoded into a 3-field narrow struct slice while skipping the other
// 13 columns via the column-length index. The "subset" sub-benchmark
// should be materially faster and allocate less than "full".
func BenchmarkSelect_FullVsSubset(b *testing.B) {
	type wide struct {
		F0  int64   `qdf:"f0"`
		F1  int64   `qdf:"f1"`
		F2  int64   `qdf:"f2"`
		F3  uint64  `qdf:"f3"`
		F4  uint64  `qdf:"f4"`
		F5  float64 `qdf:"f5"`
		F6  float64 `qdf:"f6"`
		F7  bool    `qdf:"f7"`
		F8  bool    `qdf:"f8"`
		F9  string  `qdf:"f9"`
		F10 string  `qdf:"f10"`
		F11 string  `qdf:"f11"`
		F12 int64   `qdf:"f12"`
		F13 float64 `qdf:"f13"`
		F14 string  `qdf:"f14"`
		F15 string  `qdf:"f15"`
	}
	type narrow struct {
		F0  int64  `qdf:"f0"`
		F8  bool   `qdf:"f8"`
		F15 string `qdf:"f15"`
	}

	const rowsN = 1000
	rows := make([]wide, rowsN)
	statuses := []string{"ok", "warn", "error", "fatal", "info"}
	for i := range rows {
		rows[i] = wide{
			F0:  int64(i),
			F1:  int64(i * 3),
			F2:  int64(-i),
			F3:  uint64(i) * 7,
			F4:  uint64(i*i) & 0xffff,
			F5:  float64(i) * 1.5,
			F6:  float64(i)/3.0 + 0.25,
			F7:  i%2 == 0,
			F8:  i%3 == 0,
			F9:  "host-" + statuses[i%len(statuses)],
			F10: statuses[i%len(statuses)],
			F11: "region-eu-west-1",
			F12: int64(i % 256),
			F13: float64(i) * 0.001,
			F14: "payload-" + statuses[(i+1)%len(statuses)],
			F15: statuses[(i+2)%len(statuses)],
		}
	}

	enc, err := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnIndex)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("full", func(b *testing.B) {
		b.SetBytes(int64(len(enc))) // SetBytes does not propagate from the parent
		b.ReportAllocs()
		for b.Loop() {
			var out []wide
			if err := qdf.Unmarshal(enc, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("subset", func(b *testing.B) {
		b.SetBytes(int64(len(enc)))
		b.ReportAllocs()
		for b.Loop() {
			var out []narrow
			if err := qdf.Unmarshal(enc, &out); err != nil {
				b.Fatal(err)
			}
		}
	})
}
