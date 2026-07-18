// telemetry demonstrates qdf's core win: it compresses *across* records.
//
// A batch of log lines repeats the same handful of service / level / region
// strings thousands of times. Per-record formats (json, msgpack, protobuf)
// re-encode those strings every row; qdf's Dense mode interns them once and
// columnar-compresses the numeric columns, so the wire shrinks with batch size.
//
//	go run ./examples/telemetry
package main

import (
	"encoding/json"
	"fmt"

	"github.com/alex60217101990/qdf"
)

type LogRecord struct {
	Service string `qdf:"service" json:"service"`
	Level   string `qdf:"level"   json:"level"`
	Region  string `qdf:"region"  json:"region"`
	Message string `qdf:"msg"     json:"msg"`
	Code    int32  `qdf:"code"    json:"code"`
}

func main() {
	services := []string{"auth", "api", "billing", "search"}
	levels := []string{"INFO", "WARN", "ERROR"}
	regions := []string{"eu-west-1", "us-east-1", "ap-south-1"}

	const n = 5000
	batch := make([]LogRecord, n)
	for i := range batch {
		batch[i] = LogRecord{
			Service: services[i%len(services)],
			Level:   levels[i%len(levels)],
			Region:  regions[i%len(regions)],
			Code:    int32(200 + (i%5)*100),
			Message: "request handled",
		}
	}

	jsonBytes, err := json.Marshal(batch)
	if err != nil {
		panic(err)
	}
	qdfBytes, err := qdf.Marshal(batch, qdf.OptBalanced)
	if err != nil {
		panic(err)
	}

	// Round-trip: decode straight back into the same shape.
	var back []LogRecord
	if err := qdf.Unmarshal(qdfBytes, &back); err != nil {
		panic(err)
	}
	ok := len(back) == len(batch) && back[0] == batch[0] && back[n-1] == batch[n-1]

	fmt.Printf("records:       %d\n", n)
	fmt.Printf("encoding/json: %7d bytes\n", len(jsonBytes))
	fmt.Printf("qdf balanced:  %7d bytes  (%.1fx smaller than json)\n",
		len(qdfBytes), float64(len(jsonBytes))/float64(len(qdfBytes)))
	fmt.Printf("round-trip ok: %v\n", ok)
}
