package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleOptions shows how the Options bit-mask picks which codec
// layers run for a Marshal call. The same input encodes to a
// progressively smaller wire as more codecs opt in, with a small
// CPU penalty per added codec. The decoder reads any dialect off
// the wire header so changing Options is a producer-side decision.
func ExampleOptions() {
	type Row struct {
		Service string `qdf:"service"`
		Region  string `qdf:"region"`
		Status  int    `qdf:"status"`
	}
	// Repetitive batch — every row shares the same service and region.
	in := make([]Row, 50)
	for i := range in {
		in[i] = Row{Service: "billing", Region: "eu-west-1", Status: 200}
	}

	for _, p := range []struct {
		name string
		opts qdf.Options
	}{
		{"OptSpeed", qdf.OptSpeed},
		{"OptBalanced", qdf.OptBalanced},
	} {
		buf, _ := qdf.Marshal(in, p.opts)
		fmt.Printf("%-12s wire=%d bytes\n", p.name, len(buf))
	}
	// Output:
	// OptSpeed     wire=2208 bytes
	// OptBalanced  wire=352 bytes
}

// ExampleOptions_streamingDictShared shows the headline Dense-mode
// win: across many calls of a StreamEncoder the intern table /
// shape table / predictors all survive, so the second record
// onwards trades long strings for 1-3 byte state-refs. Per-message
// Marshal does not share the dictionary across calls; only the
// stream API does.
func ExampleOptions_streamingDictShared() {
	type Row struct {
		Service string `qdf:"service"`
		Status  int    `qdf:"status"`
	}

	// 100 rows, all the same Service literal. Marshal-per-call
	// reuses the qdf encoder pool but resets state every call.
	one, _ := qdf.Marshal(Row{Service: "billing", Status: 200}, qdf.OptBalanced)
	fmt.Printf("per-message wire = %d bytes\n", len(one))

	// Output: per-message wire = 36 bytes
}
