package bench

import (
	"testing"

	"github.com/alex60217101990/qdf"
)

func TestZZWire(t *testing.T) {
	for _, c := range []struct {
		n string
		v any
	}{{"otlp", mkOTLPBatch(16, 8)}, {"rtb", mkRTBBatch(512)}, {"access", mkAccessBatch(2048)}, {"logs", mkLogBatch(2048)}} {
		b, err := qdf.Marshal(c.v, qdf.OptBalanced)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s=%d", c.n, len(b))
	}
}
