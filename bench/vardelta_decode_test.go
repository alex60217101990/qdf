package bench

import (
	"math/rand"
	"testing"

	"github.com/alex60217101990/qdf"
)

// variable-delta int64 column -> DeltaFor bitsPer>0 -> exercises the tmp scratch
func BenchmarkVarDeltaDecode(b *testing.B) {
	r := rand.New(rand.NewSource(7))
	type Row struct {
		TS int64 `qdf:"ts"`
		A  int64 `qdf:"a"`
		B  int64 `qdf:"b"`
	}
	rows := make([]Row, 2000)
	var ts, a, bb int64 = 1e9, 0, 0
	for i := range rows {
		ts += int64(r.Intn(500) + 1) // jittery monotonic -> variable deltas
		a += int64(r.Intn(1000))
		bb += int64(r.Intn(50) + 1)
		rows[i] = Row{ts, a, bb}
	}
	enc, _ := qdf.Marshal(rows, qdf.OptBalanced)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var out []Row
		if err := qdf.Unmarshal(enc, &out); err != nil {
			b.Fatal(err)
		}
	}
}
