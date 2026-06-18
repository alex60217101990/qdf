package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/vmihailenco/msgpack/v5"
)

// extCodec is a reference serializer benchmarked alongside qdf on the same data,
// so the table shows qdf's wire size and CPU relative to the stdlib JSON codec
// and a typical reflection-based msgpack codec. Both have Marshal/Unmarshal
// signatures identical to encoding/json, so one pair of funcs describes each.
type extCodec struct {
	name      string
	marshal   func(v any) ([]byte, error)
	unmarshal func(data []byte, ptr any) error
}

// baselines are the external codecs compared against qdf. They have no qdf
// Options or decode modes, so they print with bundle = the codec name and
// dec = "-".
var baselines = []extCodec{
	{"json", json.Marshal, json.Unmarshal},
	{"msgpack", msgpack.Marshal, msgpack.Unmarshal},
}

// benchExtTyped mirrors benchTyped for an external codec over the typed Info
// payloads. A round-trip mismatch is reported as a warning (not fatal): a
// reference codec may legitimately not preserve every value the way qdf does,
// and the timing is still useful.
func benchExtTyped(iters int, c extCodec, vals []*Info) stat {
	var st stat
	st.n = len(vals)
	var lastV Info
	var lastBuf []byte
	for _, p := range vals {
		v := *p

		buf, err := c.marshal(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s marshal (typed): %v\n", c.name, err)
			os.Exit(1)
		}
		var rt Info
		if err := c.unmarshal(buf, &rt); err != nil {
			fmt.Fprintf(os.Stderr, "%s unmarshal (typed): %v\n", c.name, err)
			os.Exit(1)
		}
		if !reflect.DeepEqual(v, rt) {
			fmt.Fprintf(os.Stderr, "warning: %s typed round-trip differs from source (reference codec)\n", c.name)
		}
		st.wire += uint64(len(buf))
		lastV, lastBuf = v, buf

		ns, b, al := benchOp(iters, func() { encSink, _ = c.marshal(v) })
		st.addSer(ns, b, al)

		ns, b, al = benchOp(iters, func() {
			var out Info
			_ = c.unmarshal(buf, &out)
			decInfo = out
		})
		st.addDeser(ns, b, al)
	}
	st.serLiveKiB = liveHeapKiB(
		func() { keepBuf = nil },
		func() { b, _ := c.marshal(lastV); keepBuf = append(keepBuf, b) })
	st.deserLiveKiB = liveHeapKiB(
		func() { keepInfo = nil },
		func() { var o Info; _ = c.unmarshal(lastBuf, &o); keepInfo = append(keepInfo, o) })
	return st
}

// benchExtMap mirrors benchMap for an external codec over the map[string]any
// payloads.
func benchExtMap(iters int, c extCodec, vals []map[string]any) stat {
	var st stat
	st.n = len(vals)
	var lastV map[string]any
	var lastBuf []byte
	for _, v := range vals {
		buf, err := c.marshal(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s marshal (map): %v\n", c.name, err)
			os.Exit(1)
		}
		var rt map[string]any
		if err := c.unmarshal(buf, &rt); err != nil {
			fmt.Fprintf(os.Stderr, "%s unmarshal (map): %v\n", c.name, err)
			os.Exit(1)
		}
		if !reflect.DeepEqual(v, rt) {
			fmt.Fprintf(os.Stderr, "warning: %s map round-trip differs from source (reference codec)\n", c.name)
		}
		st.wire += uint64(len(buf))
		lastV, lastBuf = v, buf

		ns, b, al := benchOp(iters, func() { encSink, _ = c.marshal(v) })
		st.addSer(ns, b, al)

		ns, b, al = benchOp(iters, func() {
			var out map[string]any
			_ = c.unmarshal(buf, &out)
			decMap = out
		})
		st.addDeser(ns, b, al)
	}
	st.serLiveKiB = liveHeapKiB(
		func() { keepBuf = nil },
		func() { b, _ := c.marshal(lastV); keepBuf = append(keepBuf, b) })
	st.deserLiveKiB = liveHeapKiB(
		func() { keepMap = nil },
		func() { var o map[string]any; _ = c.unmarshal(lastBuf, &o); keepMap = append(keepMap, o) })
	return st
}
