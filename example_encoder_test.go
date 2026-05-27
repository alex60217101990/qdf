package qdf_test

import (
	"fmt"

	"github.com/alex60217101990/qdf"
)

// ExampleEncoder drives the encoder by hand instead of via the
// top-level Marshal pool. Use this when you need to:
//   - keep a reusable buffer alive across many encodes;
//   - call PreIntern with a known hot string pool;
//   - mix WriteX primitives with reflect-driven EncodeValue.
func ExampleEncoder() {
	enc := qdf.NewEncoderWith(qdf.OptSpeed)

	// Primitives + struct fields can be mixed by hand.
	enc.WriteMapHeader(2)
	enc.WriteString("name")
	enc.WriteString("alice")
	enc.WriteString("age")
	enc.WriteInt(33)

	dec := qdf.NewDecoderOnBuf(enc.Bytes())
	n, _ := dec.ReadMapHeader()
	for range n {
		k, _ := dec.ReadString()
		switch k {
		case "name":
			v, _ := dec.ReadString()
			fmt.Printf("%s=%s\n", k, v)
		case "age":
			v, _ := dec.ReadInt()
			fmt.Printf("%s=%d\n", k, v)
		}
	}
	// Output:
	// name=alice
	// age=33
}

// ExampleEncoder_PreIntern registers a known string pool against
// the encoder so subsequent WriteString calls with the same
// backing pointer skip the intern table's hash + slot probe and
// emit a state-ref directly. Use this when the caller knows the
// hot vocabulary up front (service names, region codes, enum-
// like fields).
//
// The caller must keep the registered strings' backing memory
// alive for the lifetime of the next encode call. String
// literals embedded in a slice, global, or struct field stay
// alive automatically; short-lived stack strings do not.
func ExampleEncoder_PreIntern() {
	type Row struct {
		Service string `qdf:"service"`
		Region  string `qdf:"region"`
		Status  int    `qdf:"status"`
	}

	// Stable hot pool — backing slices live for the program's
	// lifetime, so the registered pointers stay valid across
	// many encode calls on the same encoder.
	services := []string{"billing", "auth", "ingest"}
	regions := []string{"eu-west-1", "us-east-1"}

	enc := qdf.NewEncoderWith(qdf.OptBalanced)
	enc.PreIntern(services...)
	enc.PreIntern(regions...)

	rows := []Row{
		{services[0], regions[0], 200},
		{services[1], regions[1], 500},
		{services[2], regions[0], 200},
	}
	if err := enc.EncodeValue(rows); err != nil {
		panic(err)
	}

	var out []Row
	_ = qdf.Unmarshal(enc.Bytes(), &out)
	for _, r := range out {
		fmt.Printf("%s/%s=%d\n", r.Service, r.Region, r.Status)
	}
	// Output:
	// billing/eu-west-1=200
	// auth/us-east-1=500
	// ingest/eu-west-1=200
}

// ExampleEncoder_EncodeValue drives the reflect-based encode path
// against any Go value directly through the encoder, bypassing
// the pool used by qdf.Marshal. Combine with PreIntern, custom
// SetBuffer, or SetIntern tuning for a long-lived encoder that
// shapes its state for a specific workload.
func ExampleEncoder_EncodeValue() {
	type Config struct {
		Name  string `qdf:"name"`
		Limit int    `qdf:"limit"`
	}

	enc := qdf.NewEncoderWith(qdf.OptBalanced)
	if err := enc.EncodeValue(Config{Name: "service.cpu", Limit: 1000}); err != nil {
		panic(err)
	}

	var out Config
	_ = qdf.Unmarshal(enc.Bytes(), &out)
	fmt.Printf("%+v\n", out)
	// Output: {Name:service.cpu Limit:1000}
}
