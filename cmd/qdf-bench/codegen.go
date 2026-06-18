package main

import (
	"fmt"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/alex60217101990/qdf"
)

// printCodegen runs and prints the codegen-vs-reflect comparison on real data:
// the []Service and []Task slices gathered from every host. The reflect path
// encodes the real Service / Task types (no MarshalQDF, so qdf walks them with
// reflection); the codegen path encodes the GenService / GenTask defined types,
// whose qdfgen-generated MarshalQDF / UnmarshalQDF methods qdf uses instead — no
// reflection, no per-value descriptor lookup. GenTask carries the same dynamic
// map[string]any Definition field, so its codegen row also proves code
// generation handles arbitrary values (via the EncodeValue / DecodeValue
// fallback), not only static schema.
//
// Codegen emits a fixed Fast-framed body and ignores Options by contract, so a
// single bundle (summaryBundle) is used for both sides — the delta is purely
// reflect vs generated code on identical data.
func printCodegen(iters int, typed []*Info) {
	var services []Service
	var tasks []Task
	for _, p := range typed {
		services = append(services, p.Services...)
		tasks = append(tasks, p.Tasks...)
	}
	if len(services) == 0 && len(tasks) == 0 {
		return
	}
	genServices := make([]GenService, len(services))
	for i := range services {
		genServices[i] = GenService(services[i])
	}
	genTasks := make([]GenTask, len(tasks))
	for i := range tasks {
		genTasks[i] = GenTask(tasks[i])
	}
	opts := bundleOpts(summaryBundle)

	fmt.Printf("\n=== CODEGEN vs REFLECT on real data (all hosts combined): qdfgen-generated\n"+
		"    MarshalQDF/UnmarshalQDF vs the reflection path, same values. []Task carries\n"+
		"    a map[string]any field — its codegen row proves codegen handles dynamic data. ===\n"+
		"    (%d services, %d tasks)\n", len(services), len(tasks))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "payload\tpath\tser_ns\tser_B\tser_alloc\tdeser_ns\tdeser_B\tdeser_alloc\twire_B")

	// Service: reflect vs codegen.
	printCodegenRow(w, "[]Service", "reflect",
		func() ([]byte, error) { return qdf.Marshal(services, opts) },
		func(b []byte) error { var out []Service; return qdf.Unmarshal(b, &out) },
		func(b []byte) bool {
			var out []Service
			return qdf.Unmarshal(b, &out) == nil && reflect.DeepEqual(services, out)
		},
		iters)
	printCodegenRow(w, "[]Service", "codegen",
		func() ([]byte, error) { return qdf.Marshal(genServices, opts) },
		func(b []byte) error { var out []GenService; return qdf.Unmarshal(b, &out) },
		func(b []byte) bool {
			var out []GenService
			return qdf.Unmarshal(b, &out) == nil && reflect.DeepEqual(genServices, out)
		},
		iters)
	// Task: reflect vs codegen (codegen exercises the dynamic Definition field).
	printCodegenRow(w, "[]Task", "reflect",
		func() ([]byte, error) { return qdf.Marshal(tasks, opts) },
		func(b []byte) error { var out []Task; return qdf.Unmarshal(b, &out) },
		func(b []byte) bool {
			var out []Task
			return qdf.Unmarshal(b, &out) == nil && reflect.DeepEqual(tasks, out)
		},
		iters)
	printCodegenRow(w, "[]Task", "codegen",
		func() ([]byte, error) { return qdf.Marshal(genTasks, opts) },
		func(b []byte) error { var out []GenTask; return qdf.Unmarshal(b, &out) },
		func(b []byte) bool {
			var out []GenTask
			return qdf.Unmarshal(b, &out) == nil && reflect.DeepEqual(genTasks, out)
		},
		iters)

	// GenHost{Services,Tasks}: the THREADED codegen path — host.MarshalQDF threads
	// one encoder through every element, so struct field-name shape-interning fires
	// (names written once per type, not per record). Compare its wire_B to the sum
	// of the per-record []Service + []Task codegen rows above to see the win; the
	// bare-slice codegen rows open a fresh encoder per element and can't intern.
	host := GenHost{Services: genServices, Tasks: genTasks}
	printCodegenRow(w, "GenHost", "codegen-threaded",
		func() ([]byte, error) { return qdf.Marshal(host, opts) },
		func(b []byte) error { var out GenHost; return qdf.Unmarshal(b, &out) },
		func(b []byte) bool {
			var out GenHost
			return qdf.Unmarshal(b, &out) == nil && reflect.DeepEqual(host, out)
		},
		iters)
	w.Flush()
}

// printCodegenRow times one encode + decode of a whole slice (all hosts) and
// prints a row. A round-trip equality gate runs once, outside timing.
func printCodegenRow(w *tabwriter.Writer, payload, path string,
	marshal func() ([]byte, error), unmarshal func([]byte) error, roundtripOK func([]byte) bool, iters int) {

	buf, err := marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "codegen %s/%s marshal: %v\n", payload, path, err)
		os.Exit(1)
	}
	if !roundtripOK(buf) {
		fmt.Fprintf(os.Stderr, "codegen %s/%s ROUND-TRIP MISMATCH — aborting\n", payload, path)
		os.Exit(1)
	}
	serNs, serB, serAl := benchOp(iters, func() { encSink, _ = marshal() })
	deserNs, deserB, deserAl := benchOp(iters, func() { _ = unmarshal(buf) })
	fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		payload, path, serNs, serB, serAl, deserNs, deserB, deserAl, len(buf))
}
