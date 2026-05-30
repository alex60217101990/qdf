package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleUnmarshal_predicatePushdown keeps only the rows matching typed
// predicates (AND-ed), decoding just the projected columns of those rows.
//
// Predicate pushdown only fires when the payload is columnar, which the
// encoder picks for a sufficiently large []struct batch, so this example
// builds a 20-row batch rather than a handful of literals.
func ExampleUnmarshal_predicatePushdown() {
	type Event struct {
		TS    int64  `qdf:"ts"`
		Level string `qdf:"level"`
		Code  int32  `qdf:"code"`
	}
	in := make([]Event, 0, 20)
	for i := range 20 {
		lvl, code := "INFO", int32(200)
		switch i % 5 {
		case 0:
			lvl, code = "ERROR", 500
		case 1:
			lvl, code = "ERROR", 404
		case 2:
			lvl, code = "WARN", 503
		}
		in = append(in, Event{TS: int64(i + 1), Level: lvl, Code: code})
	}
	buf, _ := qdf.Marshal(in, qdf.OptBalanced|qdf.OptColumnIndex)

	type Out struct {
		TS   int64 `qdf:"ts"`
		Code int32 `qdf:"code"`
	}
	var hot []Out // filter on level (not in Out), return ts+code
	_ = qdf.Unmarshal(buf, &hot,
		qdf.Where("level", func(s string) bool { return s == "ERROR" }),
		qdf.Where("code", func(c int32) bool { return c >= 500 }))

	for _, e := range hot {
		fmt.Printf("ts=%d code=%d\n", e.TS, e.Code)
	}
	// Output:
	// ts=1 code=500
	// ts=6 code=500
	// ts=11 code=500
	// ts=16 code=500
}
