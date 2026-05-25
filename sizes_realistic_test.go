package qdf

import (
	"encoding/json"
	"testing"
)

// Reports encoded sizes (no perf measurement) for the realistic
// corpus shapes. Output goes into docs/BENCH.md via copy-paste.
func TestSizes_RealisticCorpus(t *testing.T) {
	telem := makeTelemetryBatch(1000)
	metric := makeMetricSeries(1024)

	jb, _ := json.Marshal(telem)
	fb, _ := Marshal(telem)
	qb, _ := MarshalQPack(telem)
	db, _ := MarshalDense(telem)
	t.Logf("TelemetryBatch n=1000  json=%d  qdf_fast=%d  qdf_qpack=%d  qdf_dense=%d", len(jb), len(fb), len(qb), len(db))

	jb, _ = json.Marshal(metric)
	fb, _ = Marshal(metric)
	qb, _ = MarshalQPack(metric)
	db, _ = MarshalDense(metric)
	t.Logf("MetricSeries n=1024    json=%d  qdf_fast=%d  qdf_qpack=%d  qdf_dense=%d", len(jb), len(fb), len(qb), len(db))
}
