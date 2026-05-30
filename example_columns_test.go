package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleUnmarshalColumns decodes only a named subset of columns from a
// columnar []struct payload. The producer opts in to OptColumnIndex so the
// payload carries a column-length index; the decoder then advances past the
// columns it was not asked for instead of decoding them. Here a 4-field
// struct batch is decoded into a 2-field typed subset, naming the wanted
// columns explicitly.
func ExampleUnmarshalColumns() {
	type Event struct {
		TS      int64  `qdf:"ts"`
		Level   string `qdf:"level"`
		Service string `qdf:"service"`
		Code    int32  `qdf:"code"`
	}
	in := make([]Event, 4)
	levels := []string{"INFO", "WARN", "ERROR", "INFO"}
	for i := range in {
		in[i] = Event{
			TS:      int64(1700000000 + i),
			Level:   levels[i],
			Service: "billing",
			Code:    int32(200 + i),
		}
	}

	// OptColumnIndex writes the column-length index; the default columnar
	// wire is byte-identical without it.
	buf, _ := qdf.Marshal(in, qdf.OptBalanced|qdf.OptColumnIndex)

	// Decode only the "level" and "service" columns into a typed subset.
	// The "ts" and "code" columns are skipped via the index, not decoded.
	type Subset struct {
		Level   string `qdf:"level"`
		Service string `qdf:"service"`
	}
	var out []Subset
	if err := qdf.UnmarshalColumns(buf, &out, "level", "service"); err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, e := range out {
		fmt.Printf("level=%-5s service=%s\n", e.Level, e.Service)
	}
	// Output:
	// level=INFO  service=billing
	// level=WARN  service=billing
	// level=ERROR service=billing
	// level=INFO  service=billing
}
