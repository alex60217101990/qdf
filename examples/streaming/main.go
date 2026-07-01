// streaming shows the multi-message win: a StreamEncoder keeps ONE intern
// dictionary across every message in the stream, so a string that repeats from
// one message to the next (service names, region codes, status labels) is
// written in full once and referenced by a 1-byte id thereafter.
//
// Encoding each message independently (or with a per-record format) re-emits
// those strings every time.
//
//	go run ./examples/streaming
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/alex60217101990/qdf"
)

type Event struct {
	Service string `qdf:"service" json:"service"`
	Region  string `qdf:"region"  json:"region"`
	Code    int32  `qdf:"code"    json:"code"`
}

func main() {
	services := []string{"auth", "api", "billing"}
	regions := []string{"eu-west-1", "us-east-1"}

	const n = 1000
	events := make([]Event, n)
	for i := range events {
		events[i] = Event{services[i%3], regions[i%2], int32(200 + (i%4)*100)}
	}

	// One stream, one shared intern dictionary across all n messages.
	var sink bytes.Buffer
	enc := qdf.NewStreamEncoderWith(&sink, qdf.OptBalanced)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			panic(err)
		}
	}
	if err := enc.Flush(); err != nil {
		panic(err)
	}
	_ = enc.Close()
	streamed := sink.Len()

	// Baseline: encode each message on its own (no cross-message dictionary).
	independent := 0
	for _, e := range events {
		b, err := qdf.Marshal(e, qdf.OptBalanced)
		if err != nil {
			panic(err)
		}
		independent += len(b)
	}
	jsonBytes, _ := json.Marshal(events)

	// Decode the whole stream back.
	dec := qdf.NewStreamDecoder(&sink)
	decoded := 0
	for {
		var out Event
		if dec.Decode(&out) != nil {
			break
		}
		decoded++
	}

	fmt.Printf("messages:            %d\n", n)
	fmt.Printf("encoding/json:       %7d bytes\n", len(jsonBytes))
	fmt.Printf("qdf, per-message:    %7d bytes\n", independent)
	fmt.Printf("qdf, shared stream:  %7d bytes  (%.1fx smaller than per-message)\n",
		streamed, float64(independent)/float64(streamed))
	fmt.Printf("decoded back:        %d messages\n", decoded)
}
