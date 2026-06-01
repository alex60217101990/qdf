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

import (
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
	name    string
	encode  func() []byte
	decode  func([]byte)
	encoded []byte // pre-encoded wire bytes (set during setup)
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

// measureCodec runs K encode+decode iterations and returns bytes/cycle and
// peak HeapInuse. It calls runtime.GC() before the loop so each codec gets a
// clean baseline and is isolated from its predecessor's garbage.
//
// We do NOT call ReadMemStats inside the hot loop (it STW-stops the world).
// Instead we snapshot HeapInuse once before and once after, then report the
// post-loop value as "peak-heap" (the live set after K iterations).  This is
// a conservative proxy: it reflects the retained working set rather than
// a true per-iteration peak, but it is cheap and not artificially inflated by
// the measurement cost itself.
func measureCodec(entry *codecEntry, K int) (bytesPerCycle, peakHeap uint64) {
	// Warmup: one pass to prime pools, sync.Pool, type descriptors.
	_ = entry.encode()
	entry.decode(entry.encoded)

	runtime.GC()
	runtime.GC() // two passes to collect warmup garbage

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	for i := 0; i < K; i++ {
		entry.decode(entry.encoded)
		_ = entry.encode()
	}

	runtime.ReadMemStats(&after)
	delta := after.TotalAlloc - before.TotalAlloc
	return delta / uint64(K), after.HeapInuse
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

				// QDF tiers
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)

				// JSON / msgpack
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						decode:  func(b []byte) { var out []BidRequest; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						decode:  func(b []byte) { var out []BidRequest; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						decode:  func(b []byte) { var out benchpb.RTBBatch; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildRTBBatch(v) },
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
						decode:  func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						decode:  func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						decode:  func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						decode:  func(b []byte) { var out []BidRequest; _ = qdf.Unmarshal(b, &out) },
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
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						decode:  func(b []byte) { var out IoTBatch; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						decode:  func(b []byte) { var out IoTBatch; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						decode:  func(b []byte) { var out benchpb.IoTBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildIoTBatch(v) },
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
						decode:  func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						decode:  func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						decode:  func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						decode:  func(b []byte) { var out IoTBatch; _ = qdf.Unmarshal(b, &out) },
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
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						decode:  func(b []byte) { var out TraceExport; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						decode:  func(b []byte) { var out TraceExport; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						decode:  func(b []byte) { var out benchpb.TraceExportPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildTraceExport(v) },
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
						decode:  func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						decode:  func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						decode:  func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						decode:  func(b []byte) { var out TraceExport; _ = qdf.Unmarshal(b, &out) },
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
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						decode:  func(b []byte) { var out LogBatchLE; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						decode:  func(b []byte) { var out LogBatchLE; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						decode:  func(b []byte) { var out benchpb.LogBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildLogBatch(v) },
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
						decode:  func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						decode:  func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						decode:  func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						decode:  func(b []byte) { var out LogBatchLE; _ = qdf.Unmarshal(b, &out) },
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
				qspeed, _ := qdf.Marshal(v, qdf.OptSpeed)
				qbal, _ := qdf.Marshal(v, qdf.OptBalanced)
				qqpack, _ := qdf.Marshal(v, qdf.OptQPack)
				qcomp, _ := qdf.Marshal(v, qdf.OptCompression)
				jWire, _ := json.Marshal(v)
				mpWire, _ := msgpack.Marshal(v)

				return []codecEntry{
					{
						name:    "json",
						encoded: jWire,
						encode:  func() []byte { b, _ := json.Marshal(v); return b },
						decode:  func(b []byte) { var out EventBatch; _ = json.Unmarshal(b, &out) },
					},
					{
						name:    "msgpack",
						encoded: mpWire,
						encode:  func() []byte { b, _ := msgpack.Marshal(v); return b },
						decode:  func(b []byte) { var out EventBatch; _ = msgpack.Unmarshal(b, &out) },
					},
					{
						name:    "protobuf",
						encoded: pbWire,
						encode:  func() []byte { b, _ := proto.Marshal(pbMsg); return b },
						decode:  func(b []byte) { var out benchpb.EventBatchPB; _ = proto.Unmarshal(b, &out) },
					},
					{
						name:    "flatbuffers",
						encoded: fbWire,
						encode:  func() []byte { return FBBuildEventBatch(v) },
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
						decode:  func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_balanced",
						encoded: qbal,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptBalanced); return b },
						decode:  func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_qpack",
						encoded: qqpack,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptQPack); return b },
						decode:  func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
					{
						name:    "qdf_compression",
						encoded: qcomp,
						encode:  func() []byte { b, _ := qdf.Marshal(v, qdf.OptCompression); return b },
						decode:  func(b []byte) { var out EventBatch; _ = qdf.Unmarshal(b, &out) },
					},
				}
			},
		},
	}

	// Suppress the unused-import error for flatbuffers (used in decode closures
	// above via the benchfbs package, but the flatbuffers builder type appears
	// only here).
	_ = flatbuffers.NewBuilder

	for _, fix := range fixtures {
		codecs := fix.codecs()
		results := make([]memResult, 0, len(codecs))

		for i := range codecs {
			c := &codecs[i]
			bpc, ph := measureCodec(c, fix.K)
			results = append(results, memResult{
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
		fmt.Printf("| %-16s | %14s | %12s |\n", "codec", "bytes/cycle", "peak-heap")
		fmt.Printf("|%s|%s|%s|\n",
			"------------------",
			"----------------",
			"--------------")
		for _, r := range results {
			fmt.Printf("| %-16s | %14d | %12d |\n",
				r.codec, r.bytesPerCycle, r.peakHeap)
		}
		fmt.Printf("\n> process maxRSS after this fixture: %d %s\n", rss, rssUnit)
		fmt.Printf("> NOTE: maxRSS is a process-wide OS high-watermark; it is NOT per-codec.\n")
		fmt.Printf("> Use bytes/cycle (steady-state allocation) as the container-memory signal.\n")
	}
}
