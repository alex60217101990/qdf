// query demonstrates predicate pushdown: filtering a columnar batch without
// fully decoding it.
//
// When a []struct is encoded columnar, qdf stores each field as its own column.
// A WHERE predicate (WhereCmp / WhereRange / Where) is evaluated against the
// relevant column(s), and only the matching rows — and only the columns you keep
// — are materialized. You query the bytes instead of decoding everything first.
//
//	go run ./examples/query
package main

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

type Event struct {
	Level string `qdf:"level"`
	Msg   string `qdf:"msg"`
	Code  int32  `qdf:"code"`
}

func main() {
	const n = 5000
	batch := make([]Event, n)
	for i := range batch {
		level := "INFO"
		switch {
		case i%20 == 0:
			level = "ERROR"
		case i%5 == 0:
			level = "WARN"
		}
		batch[i] = Event{Level: level, Code: int32(200 + (i%6)*100), Msg: "..."}
	}

	data, err := qdf.Marshal(batch, qdf.OptBalanced)
	if err != nil {
		panic(err)
	}

	// Typed bound predicate on a numeric column (>= 400).
	var serverErrors []Event
	if err := qdf.Unmarshal(data, &serverErrors, qdf.WhereCmp("code", qdf.GE, int32(400))); err != nil {
		panic(err)
	}

	// Range predicate on the same column (300..399 inclusive).
	var redirects []Event
	if err := qdf.Unmarshal(data, &redirects, qdf.WhereRange("code", int32(300), int32(399))); err != nil {
		panic(err)
	}

	// Arbitrary closure predicate on a string column.
	var onlyErr []Event
	if err := qdf.Unmarshal(data, &onlyErr, qdf.Where("level", func(s string) bool { return s == "ERROR" })); err != nil {
		panic(err)
	}

	fmt.Printf("encoded:       %d rows in %d bytes\n", n, len(data))
	fmt.Printf("code >= 400:   %d rows (selective decode)\n", len(serverErrors))
	fmt.Printf("code in 300-399: %d rows\n", len(redirects))
	fmt.Printf("level == ERROR: %d rows\n", len(onlyErr))
}
