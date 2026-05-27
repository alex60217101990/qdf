package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// Example demonstrates the simplest possible round-trip: Marshal a
// value into a freshly-allocated byte slice and Unmarshal it back
// into a typed receiver.
func Example() {
	type Event struct {
		ID     int    `qdf:"id"`
		Source string `qdf:"source"`
	}

	in := Event{ID: 42, Source: "ingest"}
	buf, err := qdf.Marshal(in, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}

	var out Event
	if err := qdf.Unmarshal(buf, &out); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", out)
	// Output: {ID:42 Source:ingest}
}

// ExampleAppendMarshal shows the zero-extra-copy form. The caller
// keeps a reusable byte buffer and asks Marshal to append into it,
// returning the extended slice. Combined with a sync.Pool of
// buffers this drives encode allocations to zero on the hot path.
func ExampleAppendMarshal() {
	type Tick struct {
		Ts    int64   `qdf:"ts"`
		Price float64 `qdf:"price"`
	}

	buf := make([]byte, 0, 64) // caller-owned, reusable
	t := Tick{Ts: 1_700_000_000, Price: 99.5}

	var err error
	buf, err = qdf.AppendMarshal(buf, t, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}

	var out Tick
	if err := qdf.Unmarshal(buf, &out); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", out)
	// Output: {Ts:1700000000 Price:99.5}
}

// ExampleMarshalT shows the generic typed wrapper. T is fixed at
// the call site, so the compiler skips the reflect.TypeOf step
// that Marshal(v any, opts) pays inside; encode/decode of value
// types stays on the stack.
func ExampleMarshalT() {
	type Point struct {
		X int32 `qdf:"x"`
		Y int32 `qdf:"y"`
	}

	buf, err := qdf.MarshalT(Point{X: 3, Y: 4}, qdf.OptSpeed)
	if err != nil {
		panic(err)
	}

	var p Point
	if err := qdf.UnmarshalT(buf, &p); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", p)
	// Output: {X:3 Y:4}
}

// Example_roundTripSlice demonstrates encoding a slice of structs.
// QDF's reflect path resolves the element type once per typeDesc;
// every element after the first emits its tag stream straight
// from the cached descriptor.
func Example_roundTripSlice() {
	type Row struct {
		Service string `qdf:"service"`
		Status  int    `qdf:"status"`
	}

	rows := []Row{
		{"auth", 200},
		{"billing", 503},
		{"auth", 200},
	}

	buf, _ := qdf.Marshal(rows, qdf.OptBalanced)

	var got []Row
	_ = qdf.Unmarshal(buf, &got)
	for _, r := range got {
		fmt.Printf("%s=%d\n", r.Service, r.Status)
	}
	// Output:
	// auth=200
	// billing=503
	// auth=200
}
