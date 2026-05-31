package bench

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// 6 int64 predicate columns AND-ed, 1% match -> eval-heavy (non-nullable dense path)
func BenchmarkWidePredicateEval(b *testing.B) {
	type Row struct {
		A, B, C, D, E, F int64
		G                string `qdf:"g"`
	}
	rows := make([]Row, 4000)
	for i := range rows {
		rows[i] = Row{int64(i), int64(i * 2), int64(i * 3), int64(i), int64(i), int64(i), "x"}
	}
	enc, _ := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnIndex)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var out []struct {
			A int64 `qdf:"A"`
		}
		_ = qdf.Unmarshal(enc, &out,
			qdf.Where("A", func(v int64) bool { return v < 40 }),
			qdf.Where("B", func(v int64) bool { return v >= 0 }),
			qdf.Where("C", func(v int64) bool { return v >= 0 }),
			qdf.Where("D", func(v int64) bool { return v >= 0 }),
			qdf.Where("E", func(v int64) bool { return v >= 0 }),
			qdf.Where("F", func(v int64) bool { return v >= 0 }),
			qdf.Select("A"))
	}
}
