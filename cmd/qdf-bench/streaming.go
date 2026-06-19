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
	dec         string // decode mode label: copy / nocopy / arena (qdf), "-" otherwise
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

// streamCases builds the streaming matrix: qdf (typed + map, each in the three
// decode modes copy / nocopy / arena), encoding/json (typed + map), and msgpack
// (typed + map). Every codec constructs ONE encoder/decoder and the batch
// closures Reset it between batches, so the heavy per-stream construction (qdf's
// intern table) is paid once, outside the timed loop — the figures are the
// per-message cost, the same basis as msgpack (whose codec also resets over a
// reused buffer). json's Encoder/Decoder have no Reset but are cheap + stateless,
// so constructing per batch adds no measurable bias.
//
// The qdf decode modes exercise the StreamDecoder allocation levers: copy (one
// heap string per value), nocopy (SetNoCopy — values alias the window, zero
// copies), arena (SetArena — string bodies bump into a per-batch-Reset arena).
func streamCases(typed []*Info, dyn []map[string]any) []streamCase {
	nt, nm := len(typed), len(dyn)
	opts := bundleOpts(summaryBundle)

	// configDec applies a qdf decode mode to a StreamDecoder, returning the arena
	// (non-nil only for "arena") so the decode loop can Reset it per batch.
	configDec := func(d *qdf.StreamDecoder, mode string) *qdf.Arena {
		switch mode {
		case "nocopy":
			d.SetNoCopy(true)
		case "arena":
			a := qdf.NewArena()
			d.SetArena(a)
			return a
		}
		return nil
	}

	qdfTyped := func(mode string) streamCase {
		qe := qdf.NewStreamEncoderWith(io.Discard, opts)
		qd := qdf.NewStreamDecoder(nil)
		ar := configDec(qd, mode)
		return streamCase{
			codec: "qdf", repr: "typed", dec: mode, n: nt,
			encodeBatch: func(w io.Writer) error {
				qe.Reset(w)
				for _, p := range typed {
					if err := qe.Encode(*p); err != nil {
						return err
					}
				}
				return qe.Flush()
			},
			decodeBatch: func(r io.Reader) error {
				if ar != nil {
					ar.Reset() // reuse the arena's blocks for this batch (prior values are dead)
				}
				qd.Reset(r)
				for range nt {
					var out Info
					if err := qd.Decode(&out); err != nil {
						return err
					}
					decInfo = out
				}
				return nil
			},
			cleanup: func() { qe.Close(); qd.Close() },
		}
	}
	qdfMap := func(mode string) streamCase {
		qe := qdf.NewStreamEncoderWith(io.Discard, opts)
		qd := qdf.NewStreamDecoder(nil)
		ar := configDec(qd, mode)
		return streamCase{
			codec: "qdf", repr: "map", dec: mode, n: nm,
			encodeBatch: func(w io.Writer) error {
				qe.Reset(w)
				for _, v := range dyn {
					if err := qe.Encode(v); err != nil {
						return err
					}
				}
				return qe.Flush()
			},
			decodeBatch: func(r io.Reader) error {
				if ar != nil {
					ar.Reset()
				}
				qd.Reset(r)
				for range nm {
					var out map[string]any
					if err := qd.Decode(&out); err != nil {
						return err
					}
					decMap = out
				}
				return nil
			},
			cleanup: func() { qe.Close(); qd.Close() },
		}
	}

	meT := msgpack.NewEncoder(io.Discard)
	mdT := msgpack.NewDecoder(nil)
	meM := msgpack.NewEncoder(io.Discard)
	mdM := msgpack.NewDecoder(nil)

	return []streamCase{
		qdfTyped("copy"), qdfTyped("nocopy"), qdfTyped("arena"),
		qdfMap("copy"), qdfMap("nocopy"), qdfMap("arena"),
		{
			codec: "json", repr: "typed", dec: "-", n: nt,
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
			codec: "json", repr: "map", dec: "-", n: nm,
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
			codec: "msgpack", repr: "typed", dec: "-", n: nt,
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
			codec: "msgpack", repr: "map", dec: "-", n: nm,
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
		"    NewEncoder/NewDecoder). Per-value figures. The 'dec' column is qdf's\n" +
		"    stream decode mode — copy / nocopy (SetNoCopy) / arena (SetArena, the\n" +
		"    arena Reset per batch); json/msgpack have one mode ('-'). ===\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "codec\trepr\tdec\tser_ns\tser_B\tser_alloc\tdeser_ns\tdeser_B\tdeser_alloc\twire_B")
	for _, sc := range streamCases(typed, dyn) {
		s := benchStream(iters, sc)
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			sc.codec, sc.repr, sc.dec,
			s.serNs, s.serB, s.serAllocs,
			s.deserNs, s.deserB, s.deserAllocs, s.wirePerVal)
	}
	w.Flush()
}
