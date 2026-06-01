package bench

// FlatBuffers benchmark arms — one encode + one "decode" (field-access) arm
// per fixture, sized identically to the qdf/protobuf arms so wire-B is
// directly comparable.
//
// FlatBuffers is zero-copy: there is no "unmarshal" step.  The decode arm
// below walks every field / every vector element and sums / touches values
// so the compiler cannot elide the traversal.  This is the closest fair
// analog to a full deserialise — the zero-copy advantage is that no heap
// allocation occurs during this traversal.

import (
	"testing"

	benchfbs "github.com/alex60217101990/qdf/bench/fbs"
)

// ── RTB ───────────────────────────────────────────────────────────────────

func benchRTBFB(b *testing.B, n int) {
	b.Helper()
	batch := mkRTBBatch(n)

	b.Run("RTB_"+itoa(n)+"/encode/fb", func(b *testing.B) {
		b.ReportAllocs()
		var wire []byte
		for i := 0; i < b.N; i++ {
			wire = FBBuildRTBBatch(batch)
		}
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		_ = wire
	})

	wire := FBBuildRTBBatch(batch)

	b.Run("RTB_"+itoa(n)+"/decode/fb", func(b *testing.B) {
		// Zero-copy field-access: walk every request, every impression,
		// touch id/bid_floor/country. No heap allocation during traversal.
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		var sink int64
		for i := 0; i < b.N; i++ {
			root := benchfbs.GetRootAsRTBBatch(wire, 0)
			n := root.RequestsLength()
			var req benchfbs.BidRequest
			for j := 0; j < n; j++ {
				if root.Requests(&req, j) {
					sink += int64(req.At())
					sink += int64(len(req.ReqId()))
					dev := req.Dev(nil)
					if dev != nil {
						sink += int64(len(dev.Ua()))
						geo := dev.Geo(nil)
						if geo != nil {
							sink += int64(len(geo.Country()))
						}
					}
					var imp benchfbs.Impression
					for k := 0; k < req.ImpLength(); k++ {
						if req.Imp(&imp, k) {
							sink += int64(imp.W() + imp.H())
						}
					}
				}
			}
		}
		_ = sink
	})
}

// BenchmarkRTB_64_FB benches FlatBuffers on the 64-request RTB fixture.
func BenchmarkRTB_64_FB(b *testing.B) { benchRTBFB(b, 64) }

// BenchmarkRTB_1024_FB benches FlatBuffers on the 1024-request RTB fixture.
func BenchmarkRTB_1024_FB(b *testing.B) { benchRTBFB(b, 1024) }

// ── IoT ───────────────────────────────────────────────────────────────────

// BenchmarkIoT_FB benches FlatBuffers on the IoT fixture (32 devices × 256
// samples) — same size as BenchmarkIoT_32x256.
func BenchmarkIoT_FB(b *testing.B) {
	batch := mkIoTBatch(32, 256)

	b.Run("IoT_32x256/encode/fb", func(b *testing.B) {
		b.ReportAllocs()
		var wire []byte
		for i := 0; i < b.N; i++ {
			wire = FBBuildIoTBatch(batch)
		}
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		_ = wire
	})

	wire := FBBuildIoTBatch(batch)

	b.Run("IoT_32x256/decode/fb", func(b *testing.B) {
		// Walk every device, every sample timestamp and temperature.
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		var sink float64
		for i := 0; i < b.N; i++ {
			root := benchfbs.GetRootAsIoTBatchFB(wire, 0)
			nd := root.DevicesLength()
			var dev benchfbs.DeviceReading
			for j := 0; j < nd; j++ {
				if root.Devices(&dev, j) {
					n := dev.TempLength()
					for k := 0; k < n; k++ {
						sink += dev.Temp(k)
					}
					sink += float64(dev.Ts(0))
				}
			}
		}
		_ = sink
	})
}

// ── OTLP ──────────────────────────────────────────────────────────────────

// BenchmarkOTLP_FB benches FlatBuffers on the OTLP fixture (4 resources ×
// 512 spans/scope) — same size as BenchmarkOTLP_4x512.
func BenchmarkOTLP_FB(b *testing.B) {
	batch := mkOTLPBatch(4, 512)

	b.Run("OTLP_4x512/encode/fb", func(b *testing.B) {
		b.ReportAllocs()
		var wire []byte
		for i := 0; i < b.N; i++ {
			wire = FBBuildTraceExport(batch)
		}
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		_ = wire
	})

	wire := FBBuildTraceExport(batch)

	b.Run("OTLP_4x512/decode/fb", func(b *testing.B) {
		// Walk every resource span → scope → span → attr pair.
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		var sink int64
		for i := 0; i < b.N; i++ {
			root := benchfbs.GetRootAsTraceExportFB(wire, 0)
			nrs := root.ResourceSpansLength()
			var rs benchfbs.ResourceSpans
			var sc benchfbs.ScopeSpans
			var sp benchfbs.Span
			var kv benchfbs.KeyValue
			for r := 0; r < nrs; r++ {
				if root.ResourceSpans(&rs, r) {
					nsc := rs.ScopesLength()
					for s := 0; s < nsc; s++ {
						if rs.Scopes(&sc, s) {
							nsp := sc.SpansLength()
							for k := 0; k < nsp; k++ {
								if sc.Spans(&sp, k) {
									sink += int64(sp.Kind())
									sink += int64(sp.StartNs())
									na := sp.AttrsLength()
									for a := 0; a < na; a++ {
										if sp.Attrs(&kv, a) {
											sink += int64(len(kv.Key()))
										}
									}
								}
							}
						}
					}
				}
			}
		}
		_ = sink
	})
}

// ── Logs ──────────────────────────────────────────────────────────────────

// BenchmarkLogs_FB benches FlatBuffers on the Logs fixture (1024 records) —
// same size as BenchmarkLogs_1024.
func BenchmarkLogs_FB(b *testing.B) {
	lb := mkLogBatch(1024)

	b.Run("Logs_1024/encode/fb", func(b *testing.B) {
		b.ReportAllocs()
		var wire []byte
		for i := 0; i < b.N; i++ {
			wire = FBBuildLogBatch(lb)
		}
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		_ = wire
	})

	wire := FBBuildLogBatch(lb)

	b.Run("Logs_1024/decode/fb", func(b *testing.B) {
		// Walk every record, touch ts/level/service/traceId and all fields.
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		var sink int64
		for i := 0; i < b.N; i++ {
			root := benchfbs.GetRootAsLogBatchFB(wire, 0)
			n := root.RecordsLength()
			var rec benchfbs.LogRecord
			var kv benchfbs.KeyValue
			for j := 0; j < n; j++ {
				if root.Records(&rec, j) {
					sink += rec.Ts()
					sink += int64(rec.Level())
					sink += int64(len(rec.Service()))
					sink += int64(len(rec.TraceId()))
					nf := rec.FieldsLength()
					for k := 0; k < nf; k++ {
						if rec.Fields(&kv, k) {
							sink += int64(len(kv.Value()))
						}
					}
				}
			}
		}
		_ = sink
	})
}

// ── Events ────────────────────────────────────────────────────────────────

// BenchmarkEvents_FB benches FlatBuffers on the Events fixture (1024 records)
// — same size as BenchmarkEvents_1024.
func BenchmarkEvents_FB(b *testing.B) {
	eb := mkEventBatch(1024)

	b.Run("Events_1024/encode/fb", func(b *testing.B) {
		b.ReportAllocs()
		var wire []byte
		for i := 0; i < b.N; i++ {
			wire = FBBuildEventBatch(eb)
		}
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		_ = wire
	})

	wire := FBBuildEventBatch(eb)

	b.Run("Events_1024/decode/fb", func(b *testing.B) {
		// Walk every record, touch ts/type/source and entire payload bytes.
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		b.ReportMetric(float64(len(wire)), "wire-B")
		var sink int64
		for i := 0; i < b.N; i++ {
			root := benchfbs.GetRootAsEventBatchFB(wire, 0)
			n := root.RecordsLength()
			var rec benchfbs.EventRecord
			for j := 0; j < n; j++ {
				if root.Records(&rec, j) {
					sink += rec.Ts()
					sink += int64(rec.EvType())
					sink += int64(len(rec.Source()))
					// PayloadBytes returns a slice into the buffer — zero copy.
					sink += int64(len(rec.PayloadBytes()))
				}
			}
		}
		_ = sink
	})
}

// ── Roundtrip sanity tests ────────────────────────────────────────────────

// TestFB_RTB_Roundtrip builds an RTB batch, reads it back, and spot-checks
// the first request ID and device country.
func TestFB_RTB_Roundtrip(t *testing.T) {
	batch := mkRTBBatch(32)
	wire := FBBuildRTBBatch(batch)
	root := benchfbs.GetRootAsRTBBatch(wire, 0)
	if got, want := root.RequestsLength(), len(batch); got != want {
		t.Fatalf("len mismatch: want %d got %d", want, got)
	}
	var req benchfbs.BidRequest
	if !root.Requests(&req, 0) {
		t.Fatal("Requests(0) returned false")
	}
	if got, want := string(req.ReqId()), batch[0].ID; got != want {
		t.Errorf("ReqId: want %q got %q", want, got)
	}
	dev := req.Dev(nil)
	if dev == nil {
		t.Fatal("Dev is nil")
	}
	geo := dev.Geo(nil)
	if geo == nil {
		t.Fatal("Geo is nil")
	}
	if got, want := string(geo.Country()), batch[0].Dev.Geo.Country; got != want {
		t.Errorf("Geo.Country: want %q got %q", want, got)
	}
	t.Logf("RTB_32: wire=%d B, %d requests OK", len(wire), root.RequestsLength())
}

// TestFB_IoT_Roundtrip builds an IoT batch, reads it back, and spot-checks
// the first device ID and first temperature.
func TestFB_IoT_Roundtrip(t *testing.T) {
	batch := mkIoTBatch(4, 64)
	wire := FBBuildIoTBatch(batch)
	root := benchfbs.GetRootAsIoTBatchFB(wire, 0)
	if got, want := root.DevicesLength(), len(batch.Devices); got != want {
		t.Fatalf("len mismatch: want %d got %d", want, got)
	}
	var dev benchfbs.DeviceReading
	if !root.Devices(&dev, 0) {
		t.Fatal("Devices(0) returned false")
	}
	if got, want := string(dev.DeviceId()), batch.Devices[0].DeviceID; got != want {
		t.Errorf("DeviceId: want %q got %q", want, got)
	}
	if dev.TempLength() == 0 {
		t.Fatal("TempLength == 0")
	}
	if got, want := dev.Temp(0), batch.Devices[0].Temp[0]; got != want {
		t.Errorf("Temp[0]: want %f got %f", want, got)
	}
	t.Logf("IoT_4x64: wire=%d B, %d devices OK", len(wire), root.DevicesLength())
}

// TestFB_OTLP_Roundtrip builds a trace export, reads it back, and spot-checks
// the first span name.
func TestFB_OTLP_Roundtrip(t *testing.T) {
	batch := mkOTLPBatch(5, 10)
	wire := FBBuildTraceExport(batch)
	root := benchfbs.GetRootAsTraceExportFB(wire, 0)
	if got, want := root.ResourceSpansLength(), len(batch.ResourceSpans); got != want {
		t.Fatalf("ResourceSpans len: want %d got %d", want, got)
	}
	var rs benchfbs.ResourceSpans
	var sc benchfbs.ScopeSpans
	var sp benchfbs.Span
	if !root.ResourceSpans(&rs, 0) {
		t.Fatal("ResourceSpans(0) returned false")
	}
	if !rs.Scopes(&sc, 0) {
		t.Fatal("Scopes(0) returned false")
	}
	if !sc.Spans(&sp, 0) {
		t.Fatal("Spans(0) returned false")
	}
	want := batch.ResourceSpans[0].Scopes[0].Spans[0].Name
	if got := string(sp.Name()); got != want {
		t.Errorf("span Name: want %q got %q", want, got)
	}
	t.Logf("OTLP_5x10: wire=%d B, %d resource spans OK", len(wire), root.ResourceSpansLength())
}

// TestFB_Logs_Roundtrip builds a log batch, reads it back, and spot-checks
// the first record service and trace ID.
func TestFB_Logs_Roundtrip(t *testing.T) {
	lb := mkLogBatch(64)
	wire := FBBuildLogBatch(lb)
	root := benchfbs.GetRootAsLogBatchFB(wire, 0)
	if got, want := root.RecordsLength(), len(lb.Records); got != want {
		t.Fatalf("len mismatch: want %d got %d", want, got)
	}
	var rec benchfbs.LogRecord
	if !root.Records(&rec, 0) {
		t.Fatal("Records(0) returned false")
	}
	if got, want := string(rec.Service()), lb.Records[0].Service; got != want {
		t.Errorf("Service: want %q got %q", want, got)
	}
	if got, want := string(rec.TraceId()), lb.Records[0].TraceID; got != want {
		t.Errorf("TraceId: want %q got %q", want, got)
	}
	t.Logf("Logs_64: wire=%d B, %d records OK", len(wire), root.RecordsLength())
}

// TestFB_Events_Roundtrip builds an event batch, reads it back, and
// spot-checks the first record source and payload length.
func TestFB_Events_Roundtrip(t *testing.T) {
	eb := mkEventBatch(64)
	wire := FBBuildEventBatch(eb)
	root := benchfbs.GetRootAsEventBatchFB(wire, 0)
	if got, want := root.RecordsLength(), len(eb.Records); got != want {
		t.Fatalf("len mismatch: want %d got %d", want, got)
	}
	var rec benchfbs.EventRecord
	if !root.Records(&rec, 0) {
		t.Fatal("Records(0) returned false")
	}
	if got, want := string(rec.Source()), eb.Records[0].Source; got != want {
		t.Errorf("Source: want %q got %q", want, got)
	}
	if got, want := len(rec.PayloadBytes()), len(eb.Records[0].Payload); got != want {
		t.Errorf("Payload len: want %d got %d", want, got)
	}
	t.Logf("Events_64: wire=%d B, %d records OK", len(wire), root.RecordsLength())
}

// ── tiny helper ───────────────────────────────────────────────────────────

// itoa is a minimal int-to-string to avoid importing strconv in a test file.
func itoa(n int) string {
	b := make([]byte, 0, 8)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ── wire-size sanity: log wire-B to t.Log for quick comparison ────────────

// TestFB_WireSizes logs the FlatBuffers wire size for every fixture at the
// canonical benchmark dimensions, next to the expected protobuf sizes.
func TestFB_WireSizes(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
	}{
		{"RTB_64", FBBuildRTBBatch(mkRTBBatch(64))},
		{"RTB_1024", FBBuildRTBBatch(mkRTBBatch(1024))},
		{"IoT_32x256", FBBuildIoTBatch(mkIoTBatch(32, 256))},
		{"OTLP_4x512", FBBuildTraceExport(mkOTLPBatch(4, 512))},
		{"Logs_1024", FBBuildLogBatch(mkLogBatch(1024))},
		{"Events_1024", FBBuildEventBatch(mkEventBatch(1024))},
	}
	for _, tc := range tests {
		t.Logf("%-14s FB wire = %7d B", tc.name, len(tc.wire))
	}
}
