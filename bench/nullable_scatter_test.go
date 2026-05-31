package bench

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

// BenchmarkNullableScatter measures predicate-pushdown that projects nullable
// (*T) columns: the scatter path used to reflect.New per present row; now it
// fills one backing slab per column.
func BenchmarkNullableScatter(b *testing.B) {
	type Row struct {
		A int64  `qdf:"a"`
		P *int64 `qdf:"p"`
		Q *int64 `qdf:"q"`
	}
	rows := make([]Row, 4000)
	for i := range rows {
		v := int64(i)
		w := int64(i * 2)
		rows[i] = Row{int64(i), &v, &w}
	}
	enc, _ := qdf.Marshal(rows, qdf.OptBalanced|qdf.OptColumnIndex)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var out []Row
		_ = qdf.Unmarshal(enc, &out, qdf.Where("a", func(v int64) bool { return v >= 0 }))
	}
}
