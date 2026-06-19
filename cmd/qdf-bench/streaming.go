package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/alex60217101990/qdf"
	"github.com/vmihailenco/msgpack/v5"
)

// streamCase is one streaming codec×representation: encode every value of a
// batch into a single stream, then decode the whole stream back. encodeBatch
// writes all n values to w; decodeBatch reads exactly n values from r. The two
// are measured separately (see benchStream) so the table separates stream
// encode from stream decode, per value.
type streamCase struct {
	codec       string
	repr        string
	n           int
	encodeBatch func(w io.Writer) error
	decodeBatch func(r io.Reader) error
	// cleanup releases the persistent encoder/decoder (returned to a pool) after
	// the case is measured. A streaming codec constructs ONE encoder/decoder and
	// the batch closures Reset it per call, so the heavy per-stream construction
	// (e.g. qdf's intern table) is paid once, outside the timed loop — matching
	// real streaming use and keeping the per-message figures comparable. nil if
	// the codec has nothing to release.
	cleanup func()
}

// streamStat is the per-value result of a streaming batch.
type streamStat struct {
	serNs, deserNs         int64
	serB, deserB           uint64
	serAllocs, deserAllocs uint64
	wirePerVal             int
}

// benchStream measures stream encode and stream decode of a whole batch,
// reported per value. Encode times resetting a reused bytes.Buffer and writing
// all n values through the codec's streaming encoder; decode times reading all
// n back from a fresh reader over the encoded bytes. ns/op is the minimum over
// timingRounds (GC/scheduler noise only adds time); B/op and allocs/op are
// MemStats deltas over one GC-settled round, divided by n. A warmup primes the
// pools first.
func benchStream(iters int, sc streamCase) streamStat {
	var st streamStat
	if sc.n == 0 {
		return st
	}

	// One reference encode to size the wire and to feed the decode loop.
	var ref bytes.Buffer
	if err := sc.encodeBatch(&ref); err != nil {
		fmt.Fprintf(os.Stderr, "stream %s/%s encode: %v\n", sc.codec, sc.repr, err)
		os.Exit(1)
	}
	encoded := append([]byte(nil), ref.Bytes()...)
	st.wirePerVal = len(encoded) / sc.n

	// ---- encode ----
	var buf bytes.Buffer
	buf.Grow(len(encoded))
	encOnce := func() {
		buf.Reset()
		_ = sc.encodeBatch(&buf)
	}
	for range min(iters, maxWarmupIters) {
		encOnce()
	}
	st.serNs, st.serB, st.serAllocs = streamRounds(iters, encOnce, sc.n)

	// ---- decode ----
	// Reuse one *bytes.Reader (Reset per call) so the harness's reader allocation
	// is not attributed to the codec's decode B/op — only the codec's own
	// per-batch decoder allocation counts.
	rd := bytes.NewReader(encoded)
	decOnce := func() {
		rd.Reset(encoded)
		_ = sc.decodeBatch(rd)
	}
	for range min(iters, maxWarmupIters) {
		decOnce()
	}
	st.deserNs, st.deserB, st.deserAllocs = streamRounds(iters, decOnce, sc.n)
	if sc.cleanup != nil {
		sc.cleanup()
	}
	return st
}

// streamRounds runs fn iters times per round over timingRounds rounds, returning
// the minimum ns/op (per value) and the per-value B/allocs from a GC-settled
// first round. Mirrors benchOp but divides by the batch size n.
func streamRounds(iters int, fn func(), n int) (nsPerVal int64, bPerVal, allocsPerVal uint64) {
	nsPerVal = -1
	for r := range timingRounds {
		var m0, m1 runtime.MemStats
		if r == 0 {
			runtime.GC()
			runtime.ReadMemStats(&m0)
		}
		t0 := time.Now()
		for range iters {
			fn()
		}
		ns := time.Since(t0).Nanoseconds() / int64(iters) / int64(n)
		if nsPerVal < 0 || ns < nsPerVal {
			nsPerVal = ns
		}
		if r == 0 {
			runtime.ReadMemStats(&m1)
			perRun := uint64(iters) * uint64(n)
			bPerVal = (m1.TotalAlloc - m0.TotalAlloc) / perRun
			allocsPerVal = (m1.Mallocs - m0.Mallocs) / perRun
		}
	}
	return
}

// streamCases builds the streaming matrix: qdf (typed + map), encoding/json
// (typed + map), and msgpack (typed + map). qdf streams with summaryBundle so
// it is comparable to the headline diff. json and msgpack use their standard
// streaming encoder/decoder over the same values.
func streamCases(typed []*Info, dyn []map[string]any) []streamCase {
	nt, nm := len(typed), len(dyn)
	opts := bundleOpts(summaryBundle)

	// qdf: one StreamEncoder/Decoder per case, Reset between batches. The heavy
	// per-stream construction (intern table) is paid once at NewStream*, outside
	// the timed loop — so the measured figures are the per-message cost, the same
	// basis as msgpack (whose encoder/decoder also reset over a reused buffer).
	qeT := qdf.NewStreamEncoderWith(io.Discard, opts)
	qdT := qdf.NewStreamDecoder(nil)
	qeM := qdf.NewStreamEncoderWith(io.Discard, opts)
	qdM := qdf.NewStreamDecoder(nil)
	meT := msgpack.NewEncoder(io.Discard)
	mdT := msgpack.NewDecoder(nil)
	meM := msgpack.NewEncoder(io.Discard)
	mdM := msgpack.NewDecoder(nil)

	return []streamCase{
		{
			codec: "qdf", repr: "typed", n: nt,
			encodeBatch: func(w io.Writer) error {
				qeT.Reset(w)
				for _, p := range typed {
					if err := qeT.Encode(*p); err != nil {
						return err
					}
				}
				return qeT.Flush()
			},
			decodeBatch: func(r io.Reader) error {
				qdT.Reset(r)
				for range nt {
					var out Info
					if err := qdT.Decode(&out); err != nil {
						return err
					}
					decInfo = out
				}
				return nil
			},
			cleanup: func() { qeT.Close(); qdT.Close() },
		},
		{
			codec: "qdf", repr: "map", n: nm,
			encodeBatch: func(w io.Writer) error {
				qeM.Reset(w)
				for _, v := range dyn {
					if err := qeM.Encode(v); err != nil {
						return err
					}
				}
				return qeM.Flush()
			},
			decodeBatch: func(r io.Reader) error {
				qdM.Reset(r)
				for range nm {
					var out map[string]any
					if err := qdM.Decode(&out); err != nil {
						return err
					}
					decMap = out
				}
				return nil
			},
			cleanup: func() { qeM.Close(); qdM.Close() },
		},
		{
			// json's Encoder/Decoder have no Reset; NewEncoder/NewDecoder are
			// cheap (no cross-message state), so constructing per batch adds no
			// measurable bias.
			codec: "json", repr: "typed", n: nt,
			encodeBatch: func(w io.Writer) error {
				e := json.NewEncoder(w)
				for _, p := range typed {
					if err := e.Encode(*p); err != nil {
						return err
					}
				}
				return nil
			},
			decodeBatch: func(r io.Reader) error {
				d := json.NewDecoder(r)
				for range nt {
					var out Info
					if err := d.Decode(&out); err != nil {
						return err
					}
					decInfo = out
				}
				return nil
			},
		},
		{
			codec: "json", repr: "map", n: nm,
			encodeBatch: func(w io.Writer) error {
				e := json.NewEncoder(w)
				for _, v := range dyn {
					if err := e.Encode(v); err != nil {
						return err
					}
				}
				return nil
			},
			decodeBatch: func(r io.Reader) error {
				d := json.NewDecoder(r)
				for range nm {
					var out map[string]any
					if err := d.Decode(&out); err != nil {
						return err
					}
					decMap = out
				}
				return nil
			},
		},
		{
			codec: "msgpack", repr: "typed", n: nt,
			encodeBatch: func(w io.Writer) error {
				meT.Reset(w)
				for _, p := range typed {
					if err := meT.Encode(*p); err != nil {
						return err
					}
				}
				return nil
			},
			decodeBatch: func(r io.Reader) error {
				mdT.Reset(r)
				for range nt {
					var out Info
					if err := mdT.Decode(&out); err != nil {
						return err
					}
					decInfo = out
				}
				return nil
			},
		},
		{
			codec: "msgpack", repr: "map", n: nm,
			encodeBatch: func(w io.Writer) error {
				meM.Reset(w)
				for _, v := range dyn {
					if err := meM.Encode(v); err != nil {
						return err
					}
				}
				return nil
			},
			decodeBatch: func(r io.Reader) error {
				mdM.Reset(r)
				for range nm {
					var out map[string]any
					if err := mdM.Decode(&out); err != nil {
						return err
					}
					decMap = out
				}
				return nil
			},
		},
	}
}

// printStreaming runs and prints the streaming section: per-value stream encode
// and decode cost for qdf vs json vs msgpack, over the same batches.
func printStreaming(iters int, typed []*Info, dyn []map[string]any) {
	fmt.Printf("\n=== STREAMING: encode then decode the whole batch through each codec's\n" +
		"    streaming Encoder/Decoder (qdf StreamEncoder/Decoder, json/msgpack\n" +
		"    NewEncoder/NewDecoder). Per-value figures. ===\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "codec\trepr\tser_ns\tser_B\tser_alloc\tdeser_ns\tdeser_B\tdeser_alloc\twire_B")
	for _, sc := range streamCases(typed, dyn) {
		s := benchStream(iters, sc)
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			sc.codec, sc.repr,
			s.serNs, s.serB, s.serAllocs,
			s.deserNs, s.deserB, s.deserAllocs, s.wirePerVal)
	}
	w.Flush()
}
