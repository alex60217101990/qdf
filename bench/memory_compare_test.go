package bench

// TestMemoryReport measures per-encode+decode-cycle allocation pressure and
// peak heap usage for every codec across the five canonical LARGE fixtures.
//
// Gate: set EMIT_MEM=1 to run; otherwise the test is skipped.
//
// Usage:
//
//	cd bench && EMIT_MEM=1 go test -run=TestMemoryReport -count=1 -v .
//
// Metrics reported per codec:
//
//	bytes/cycle  — TotalAlloc delta / K iterations.
//	             This is the GC-pressure driver: bytes the allocator touched
//	             per encode+decode round-trip at steady state (after a warmup
//	             pass). Lower is better for container RSS.
//
//	peak-heap    — Maximum HeapInuse (bytes) observed across the K iterations,
//	             sampled via ReadMemStats after every iteration.  Reflects
//	             working-set peak inside this codec's window.
//
// maxRSS note: process-wide maxRSS is SHARED across all codecs in a single
// go test process (the OS high-watermark never resets within a process).  We
// therefore report it ONCE per fixture run — after all codecs — with an
// explicit caveat.  On macOS ru_maxrss is in BYTES; on Linux it is in
// KILOBYTES (see RUSAGE(2)).  We normalise to bytes via runtime.GOOS.
//
// FAIRNESS NOTE (encode-side buffer reuse):
//
// qdf's Marshal pools its encoder internally (reused scratch/intern state
// across calls). To give all codecs a fair comparison, TestMemoryReport
// emits TWO tables per fixture:
//
//   - "default API"   — each codec called via its simplest public API
//     (json.Marshal, proto.Marshal, etc.).  This is
//     the API a new user would write; it may or may not
//     reuse internal state.
//
//   - "reuse buffer"  — each codec called in its most allocation-friendly
//     mode: output buffer pre-allocated and reused across
//     iterations via buf[:0] / builder.Reset().
//     This is the apples-to-apples comparison.
//
// Decode side: all codecs allocate a fresh output target per iteration
// (var out T; _ = codec.Unmarshal(b, &out)). This is symmetric across
// codecs and is therefore left unchanged in both tables.
//
// Codec-specific notes:
//   - qdf:        default uses qdf.Marshal; reuse uses qdf.AppendMarshal(buf[:0], …).
//   - protobuf:   default uses proto.Marshal; reuse uses proto.MarshalOptions{}.MarshalAppend(buf[:0], …).
//   - msgpack:    msgpack.Marshal already pools its encoder internally; reuse
//     wires a GetEncoder()+Reset to a reused bytes.Buffer — same
//     pool, but the bytes.Buffer itself is also reused.
//   - json:       json.Marshal allocs a fresh buffer; reuse uses a reused
//     bytes.Buffer + json.NewEncoder(buf).Encode.  Note: json.Encoder
//     appends a trailing newline; wire length differs by 1 byte.
//   - flatbuffers: reuse calls builder.Reset() on a single pre-allocated
//     Builder instead of flatbuffers.NewBuilder each call.
//     FinishedBytes() returns a view into the builder's internal
//     Bytes slice; the view is valid until the next Reset().
//     The decode arm uses entry.encoded (a stable copy) so this
//     is safe.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
	benchfbs "github.com/alex60217101990/qdf/bench/fbs"
	benchpb "github.com/alex60217101990/qdf/bench/pb"
	flatbuffers "github.com/google/flatbuffers/go"
	msgpack "github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/proto"
)

// codecEntry describes a single codec arm in the memory matrix.
type codecEntry struct {
	name        string
	encode      func() []byte // default-API encode
	encodeReuse func() []byte // reuse-buffer encode (nil means same as encode)
	decode      func([]byte)
	encoded     []byte // pre-encoded wire bytes (set during setup)
}

// memResult holds measurements for one codec.
type memResult struct {
	codec         string
	bytesPerCycle uint64 // TotalAlloc delta / K
	peakHeap      uint64 // max HeapInuse observed
}

// maxRSSBytes returns the current process maxRSS normalised to bytes.
// On macOS ru_maxrss is already bytes; on Linux it is kilobytes.
func maxRSSBytes() uint64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	rss := uint64(ru.Maxrss) //nolint:unconvert
	if runtime.GOOS == "linux" {
		rss *= 1024 // Linux reports kB
	}
	return rss
}

// measureCodecWith runs K encode+decode iterations using the provided encodeFn
// and returns bytes/cycle and peak HeapInuse. It calls runtime.GC() before the
// loop so each measurement gets a clean baseline isolated from its predecessor.
//
// We do NOT call ReadMemStats inside the hot loop (it STW-stops the world).
// Instead we snapshot HeapInuse once before and once after, then report the
// post-loop value as "peak-heap" (the live set after K iterations).
func measureCodecWith(entry *codecEntry, encodeFn func() []byte, K int) (bytesPerCycle, peakHeap uint64) {
	// Warmup: one pass to prime pools, sync.Pool, type descriptors.
	_ = encodeFn()
	entry.decode(entry.encoded)

	runtime.GC()
	runtime.GC() // two passes to collect warmup garbage

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	for i := 0; i < K; i++ {
		entry.decode(entry.encoded)
		_ = encodeFn()
	}

	runtime.ReadMemStats(&after)
	delta := after.TotalAlloc - before.TotalAlloc
	return delta / uint64(K), after.HeapInuse
}

// measureCodec measures the default-API encode path.
func measureCodec(entry *codecEntry, K int) (bytesPerCycle, peakHeap uint64) {
	return measureCodecWith(entry, entry.encode, K)
}

// measureCodecReuse measures the reuse-buffer encode path. Falls back to the
// default-API path when no reuse encode function is registered.
func measureCodecReuse(entry *codecEntry, K int) (bytesPerCycle, peakHeap uint64) {
	fn := entry.encodeReuse
	if fn == nil {
		fn = entry.encode
	}
	return measureCodecWith(entry, fn, K)
}

func TestMemoryReport(t *testing.T) {
	if os.Getenv("EMIT_MEM") == "" {
		t.Skip("set EMIT_MEM=1 to run memory report")
	}

	// K is chosen so each codec finishes in a few seconds on a laptop.
	// Large, deeply-nested fixtures (OTLP, RTB) need fewer iterations than
	// simple ones (Events) to stay well under the test timeout.
	type fixture struct {
		name   string
		K      int // encode+decode iterations per codec
		codecs func() []codecEntry
	}

	fixtures := []fixture{
		{
			name: "RTB_1024",
			K:    200,
			codecs: func() []codecEntry {
				v := mkRTBBatch(1024)

				// Protobuf
				pbMsg := toPBRTBBatch(v)
				pbWire, _ := proto.Marshal(pbMsg)

				// FlatBuffers
				fbWire := FBBuildRTBBatch(v)
				fbBuilder := flatbuffers.NewBuilder(1024 * len(v))

				// QDF tiers
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)

				// JSON / msgpack
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				// Reuse buffers
				var qBuf, pbBuf []byte
				var mpBuf bytes.Buffer
				var jBuf bytes.Buffer
				mpEnc := msgpack.GetEncoder()
				mpEnc.Reset(&mpBuf)

				// Sanity: verify reuse-buffer codecs produce same wire length.
				qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
				if len(qBuf) != len(qspeed) {
					t.Errorf("RTB_1024 qdf_speed reuse len mismatch: got %d want %d", len(qBuf), len(qspeed))
				}
				pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
				if len(pbBuf) != len(pbWire) {
					t.Errorf("RTB_1024 protobuf reuse len mismatch: got %d want %d", len(pbBuf), len(pbWire))
				}

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						encodeReuse: func() []byte {
							jBuf.Reset()
							enc := json.NewEncoder(&jBuf)
							_ = enc.Encode(v) // appends trailing newline; len = len(jWire)+1
							return jBuf.Bytes()
						},
						decode: func(b []byte) { var out []BidRequest; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						encodeReuse: func() []byte {
							mpBuf.Reset()
							mpEnc.Reset(&mpBuf)
							_ = mpEnc.Encode(v)
							return mpBuf.Bytes()
						},
						decode: func(b []byte) { var out []BidRequest; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						encodeReuse: func() []byte {
							pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
							return pbBuf
						},
						decode: func(b []byte) { var out benchpb.RTBBatch; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildRTBBatch(v) },
						encodeReuse: func() []byte {
							fbBuilder.Reset()
							return fbBuildRTBBatchWithBuilder(fbBuilder, v)
						},
						decode: func(b []byte) {
							root := benchfbs.GetRootAsRTBBatch(b, 0)
							var req benchfbs.BidRequest
							for j := 0; j < root.RequestsLength(); j++ {
								root.Requests(&req, j)
							}
						},
					},
					{
						name:    "qdf_speed",
						encoded: qspeed,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptSpeed); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
							return qBuf
						},
						decode: func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptBalanced)
							return qBuf
						},
						decode: func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptQPack)
							return qBuf
						},
						decode: func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptCompression)
							return qBuf
						},
						decode: func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
		{
			name: "IoT_32x256",
			K:    500,
			codecs: func() []codecEntry {
				v := mkIoTBatch(32, 256)

				pbMsg := toPBIoTBatch(v)
				pbWire, _ := proto.Marshal(pbMsg)
				fbWire := FBBuildIoTBatch(v)
				fbBuilder := flatbuffers.NewBuilder(1024 * len(v.Devices))
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				var qBuf, pbBuf []byte
				var mpBuf bytes.Buffer
				var jBuf bytes.Buffer
				mpEnc := msgpack.GetEncoder()
				mpEnc.Reset(&mpBuf)

				pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
				if len(pbBuf) != len(pbWire) {
					t.Errorf("IoT_32x256 protobuf reuse len mismatch: got %d want %d", len(pbBuf), len(pbWire))
				}

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						encodeReuse: func() []byte {
							jBuf.Reset()
							enc := json.NewEncoder(&jBuf)
							_ = enc.Encode(v)
							return jBuf.Bytes()
						},
						decode: func(b []byte) { var out IoTBatch; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						encodeReuse: func() []byte {
							mpBuf.Reset()
							mpEnc.Reset(&mpBuf)
							_ = mpEnc.Encode(v)
							return mpBuf.Bytes()
						},
						decode: func(b []byte) { var out IoTBatch; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						encodeReuse: func() []byte {
							pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
							return pbBuf
						},
						decode: func(b []byte) { var out benchpb.IoTBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildIoTBatch(v) },
						encodeReuse: func() []byte {
							fbBuilder.Reset()
							return fbBuildIoTBatchWithBuilder(fbBuilder, v)
						},
						decode: func(b []byte) {
							root := benchfbs.GetRootAsIoTBatchFB(b, 0)
							var dev benchfbs.DeviceReading
							for j := 0; j < root.DevicesLength(); j++ {
								root.Devices(&dev, j)
							}
						},
					},
					{
						name:    "qdf_speed",
						encoded: qspeed,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptSpeed); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
							return qBuf
						},
						decode: func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptBalanced)
							return qBuf
						},
						decode: func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptQPack)
							return qBuf
						},
						decode: func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptCompression)
							return qBuf
						},
						decode: func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
		{
			name: "OTLP_4x512",
			K:    100,
			codecs: func() []codecEntry {
				v := mkOTLPBatch(4, 512)

				pbMsg := toPBTraceExport(v)
				pbWire, _ := proto.Marshal(pbMsg)
				fbWire := FBBuildTraceExport(v)
				fbBuilder := flatbuffers.NewBuilder(4096)
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				var qBuf, pbBuf []byte
				var mpBuf bytes.Buffer
				var jBuf bytes.Buffer
				mpEnc := msgpack.GetEncoder()
				mpEnc.Reset(&mpBuf)

				pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
				if len(pbBuf) != len(pbWire) {
					t.Errorf("OTLP_4x512 protobuf reuse len mismatch: got %d want %d", len(pbBuf), len(pbWire))
				}

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						encodeReuse: func() []byte {
							jBuf.Reset()
							enc := json.NewEncoder(&jBuf)
							_ = enc.Encode(v)
							return jBuf.Bytes()
						},
						decode: func(b []byte) { var out TraceExport; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						encodeReuse: func() []byte {
							mpBuf.Reset()
							mpEnc.Reset(&mpBuf)
							_ = mpEnc.Encode(v)
							return mpBuf.Bytes()
						},
						decode: func(b []byte) { var out TraceExport; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						encodeReuse: func() []byte {
							pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
							return pbBuf
						},
						decode: func(b []byte) { var out benchpb.TraceExportPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildTraceExport(v) },
						encodeReuse: func() []byte {
							fbBuilder.Reset()
							return fbBuildTraceExportWithBuilder(fbBuilder, v)
						},
						decode: func(b []byte) {
							root := benchfbs.GetRootAsTraceExportFB(b, 0)
							var rs benchfbs.ResourceSpans
							for j := 0; j < root.ResourceSpansLength(); j++ {
								root.ResourceSpans(&rs, j)
							}
						},
					},
					{
						name:    "qdf_speed",
						encoded: qspeed,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptSpeed); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
							return qBuf
						},
						decode: func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptBalanced)
							return qBuf
						},
						decode: func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptQPack)
							return qBuf
						},
						decode: func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptCompression)
							return qBuf
						},
						decode: func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
		{
			name: "Logs_1024",
			K:    500,
			codecs: func() []codecEntry {
				v := mkLogBatch(1024)

				pbMsg := toPBLogBatch(v)
				pbWire, _ := proto.Marshal(pbMsg)
				fbWire := FBBuildLogBatch(v)
				fbBuilder := flatbuffers.NewBuilder(256 * len(v.Records))
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				var qBuf, pbBuf []byte
				var mpBuf bytes.Buffer
				var jBuf bytes.Buffer
				mpEnc := msgpack.GetEncoder()
				mpEnc.Reset(&mpBuf)

				pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
				if len(pbBuf) != len(pbWire) {
					t.Errorf("Logs_1024 protobuf reuse len mismatch: got %d want %d", len(pbBuf), len(pbWire))
				}

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						encodeReuse: func() []byte {
							jBuf.Reset()
							enc := json.NewEncoder(&jBuf)
							_ = enc.Encode(v)
							return jBuf.Bytes()
						},
						decode: func(b []byte) { var out LogBatchLE; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						encodeReuse: func() []byte {
							mpBuf.Reset()
							mpEnc.Reset(&mpBuf)
							_ = mpEnc.Encode(v)
							return mpBuf.Bytes()
						},
						decode: func(b []byte) { var out LogBatchLE; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						encodeReuse: func() []byte {
							pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
							return pbBuf
						},
						decode: func(b []byte) { var out benchpb.LogBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildLogBatch(v) },
						encodeReuse: func() []byte {
							fbBuilder.Reset()
							return fbBuildLogBatchWithBuilder(fbBuilder, v)
						},
						decode: func(b []byte) {
							root := benchfbs.GetRootAsLogBatchFB(b, 0)
							var rec benchfbs.LogRecord
							for j := 0; j < root.RecordsLength(); j++ {
								root.Records(&rec, j)
							}
						},
					},
					{
						name:    "qdf_speed",
						encoded: qspeed,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptSpeed); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
							return qBuf
						},
						decode: func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptBalanced)
							return qBuf
						},
						decode: func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptQPack)
							return qBuf
						},
						decode: func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptCompression)
							return qBuf
						},
						decode: func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
		{
			name: "Events_1024",
			K:    500,
			codecs: func() []codecEntry {
				v := mkEventBatch(1024)

				pbMsg := toPBEventBatch(v)
				pbWire, _ := proto.Marshal(pbMsg)
				fbWire := FBBuildEventBatch(v)
				fbBuilder := flatbuffers.NewBuilder(128 * len(v.Records))
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				var qBuf, pbBuf []byte
				var mpBuf bytes.Buffer
				var jBuf bytes.Buffer
				mpEnc := msgpack.GetEncoder()
				mpEnc.Reset(&mpBuf)

				pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
				if len(pbBuf) != len(pbWire) {
					t.Errorf("Events_1024 protobuf reuse len mismatch: got %d want %d", len(pbBuf), len(pbWire))
				}

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						encodeReuse: func() []byte {
							jBuf.Reset()
							enc := json.NewEncoder(&jBuf)
							_ = enc.Encode(v)
							return jBuf.Bytes()
						},
						decode: func(b []byte) { var out EventBatch; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						encodeReuse: func() []byte {
							mpBuf.Reset()
							mpEnc.Reset(&mpBuf)
							_ = mpEnc.Encode(v)
							return mpBuf.Bytes()
						},
						decode: func(b []byte) { var out EventBatch; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						encodeReuse: func() []byte {
							pbBuf, _ = proto.MarshalOptions{}.MarshalAppend(pbBuf[:0], pbMsg)
							return pbBuf
						},
						decode: func(b []byte) { var out benchpb.EventBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildEventBatch(v) },
						encodeReuse: func() []byte {
							fbBuilder.Reset()
							return fbBuildEventBatchWithBuilder(fbBuilder, v)
						},
						decode: func(b []byte) {
							root := benchfbs.GetRootAsEventBatchFB(b, 0)
							var rec benchfbs.EventRecord
							for j := 0; j < root.RecordsLength(); j++ {
								root.Records(&rec, j)
							}
						},
					},
					{
						name:    "qdf_speed",
						encoded: qspeed,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptSpeed); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptSpeed)
							return qBuf
						},
						decode: func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptBalanced)
							return qBuf
						},
						decode: func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptQPack)
							return qBuf
						},
						decode: func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						encodeReuse: func() []byte {
							qBuf, _ = qdf.AppendMarshal(qBuf[:0], v, qdf.OptCompression)
							return qBuf
						},
						decode: func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
	}

	for _, fix := range fixtures {
		codecs := fix.codecs()
		defaultResults := make([]memResult, 0, len(codecs))
		reuseResults := make([]memResult, 0, len(codecs))

		for i := range codecs {
			c := &codecs[i]
			bpc, ph := measureCodec(c, fix.K)
			defaultResults = append(defaultResults, memResult{
				codec:         c.name,
				bytesPerCycle: bpc,
				peakHeap:      ph,
			})
		}

		for i := range codecs {
			c := &codecs[i]
			bpc, ph := measureCodecReuse(c, fix.K)
			reuseResults = append(reuseResults, memResult{
				codec:         c.name,
				bytesPerCycle: bpc,
				peakHeap:      ph,
			})
		}

		// Print whole-run maxRSS once (process-global high-watermark).
		rss := maxRSSBytes()
		rssUnit := "bytes"
		if runtime.GOOS == "linux" {
			rssUnit = "bytes (normalised from kB)"
		}

		now := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("\n## Fixture: %s  (K=%d, %s)\n\n", fix.name, fix.K, now)

		fmt.Printf("### default API (no explicit buffer reuse)\n\n")
		fmt.Printf("| %-16s | %14s | %12s |\n", "codec", "bytes/cycle", "peak-heap")
		fmt.Printf("|%s|%s|%s|\n",
			"------------------",
			"----------------",
			"--------------")
		for _, r := range defaultResults {
			fmt.Printf("| %-16s | %14d | %12d |\n",
				r.codec, r.bytesPerCycle, r.peakHeap)
		}

		fmt.Printf("\n### reuse buffer (fair, apples-to-apples)\n\n")
		fmt.Printf("  Encode: qdf=AppendMarshal(buf[:0]), proto=MarshalAppend(buf[:0]),\n")
		fmt.Printf("          msgpack=GetEncoder+reused bytes.Buffer, json=NewEncoder(reused bytes.Buffer),\n")
		fmt.Printf("          flatbuffers=builder.Reset().\n")
		fmt.Printf("  Decode: all codecs allocate a fresh output target — symmetric, unchanged.\n\n")
		fmt.Printf("| %-16s | %14s | %12s |\n", "codec", "bytes/cycle", "peak-heap")
		fmt.Printf("|%s|%s|%s|\n",
			"------------------",
			"----------------",
			"--------------")
		for _, r := range reuseResults {
			fmt.Printf("| %-16s | %14d | %12d |\n",
				r.codec, r.bytesPerCycle, r.peakHeap)
		}

		fmt.Printf("\n> process maxRSS after this fixture: %d %s\n", rss, rssUnit)
		fmt.Printf("> NOTE: maxRSS is a process-wide OS high-watermark; it is NOT per-codec.\n")
		fmt.Printf("> Use bytes/cycle (steady-state allocation) as the container-memory signal.\n")
	}
}

// ── builder-reuse helpers (accept an existing *Builder instead of allocating) ──
//
// These mirror the public FBBuild* functions in flatbuffers_conv.go but take
// a pre-allocated Builder so the benchmark can call builder.Reset() between
// iterations instead of flatbuffers.NewBuilder.

func fbBuildRTBBatchWithBuilder(b *flatbuffers.Builder, batch []BidRequest) []byte {
	reqOffsets := make([]flatbuffers.UOffsetT, len(batch))
	for i, req := range batch {
		reqOffsets[i] = fbBuildBidRequest(b, req)
	}
	b.StartVector(4, len(reqOffsets), 4)
	for i := len(reqOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(reqOffsets[i])
	}
	reqs := b.EndVector(len(reqOffsets))
	benchfbs.RTBBatchStart(b)
	benchfbs.RTBBatchAddRequests(b, reqs)
	root := benchfbs.RTBBatchEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

func fbBuildIoTBatchWithBuilder(b *flatbuffers.Builder, batch IoTBatch) []byte {
	devOffsets := make([]flatbuffers.UOffsetT, len(batch.Devices))
	for i, d := range batch.Devices {
		devOffsets[i] = fbBuildDeviceReading(b, d)
	}
	benchfbs.IoTBatchFBStartDevicesVector(b, len(devOffsets))
	for i := len(devOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(devOffsets[i])
	}
	devs := b.EndVector(len(devOffsets))
	benchfbs.IoTBatchFBStart(b)
	benchfbs.IoTBatchFBAddDevices(b, devs)
	root := benchfbs.IoTBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

func fbBuildTraceExportWithBuilder(b *flatbuffers.Builder, te TraceExport) []byte {
	rsOffsets := make([]flatbuffers.UOffsetT, len(te.ResourceSpans))
	for i, rs := range te.ResourceSpans {
		rsOffsets[i] = fbBuildResourceSpans(b, rs)
	}
	benchfbs.TraceExportFBStartResourceSpansVector(b, len(rsOffsets))
	for i := len(rsOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(rsOffsets[i])
	}
	rsVec := b.EndVector(len(rsOffsets))
	benchfbs.TraceExportFBStart(b)
	benchfbs.TraceExportFBAddResourceSpans(b, rsVec)
	root := benchfbs.TraceExportFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

func fbBuildLogBatchWithBuilder(b *flatbuffers.Builder, lb LogBatchLE) []byte {
	recOffsets := make([]flatbuffers.UOffsetT, len(lb.Records))
	for i, r := range lb.Records {
		recOffsets[i] = fbBuildLogRecord(b, r)
	}
	benchfbs.LogBatchFBStartRecordsVector(b, len(recOffsets))
	for i := len(recOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(recOffsets[i])
	}
	recs := b.EndVector(len(recOffsets))
	benchfbs.LogBatchFBStart(b)
	benchfbs.LogBatchFBAddRecords(b, recs)
	root := benchfbs.LogBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

func fbBuildEventBatchWithBuilder(b *flatbuffers.Builder, eb EventBatch) []byte {
	recOffsets := make([]flatbuffers.UOffsetT, len(eb.Records))
	for i, e := range eb.Records {
		recOffsets[i] = fbBuildEventRecord(b, e)
	}
	benchfbs.EventBatchFBStartRecordsVector(b, len(recOffsets))
	for i := len(recOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(recOffsets[i])
	}
	recs := b.EndVector(len(recOffsets))
	benchfbs.EventBatchFBStart(b)
	benchfbs.EventBatchFBAddRecords(b, recs)
	root := benchfbs.EventBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}
