package bench

import (
	"testing"

	"google.golang.org/protobuf/proto"

	benchpb "github.com/alex60217101990/qdf/bench/pb"
)

// runProtobufArm benches proto.Marshal + proto.Unmarshal for a single
// protobuf message and reports wire-B so it sits next to the other codec
// columns. newOut must return a fresh proto.Message to decode into.
func runProtobufArm(b *testing.B, name string, msg proto.Message, newOut func() proto.Message) {
	b.Helper()

	wire, err := proto.Marshal(msg)
	if err != nil {
		b.Fatalf("proto.Marshal(%s): %v", name, err)
	}

	b.Run(name+"/encode/protobuf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		var buf []byte
		for i := 0; i < b.N; i++ {
			buf, err = proto.Marshal(msg)
			if err != nil {
				b.Fatal(err)
			}
		}
		_ = buf
		b.ReportMetric(float64(len(wire)), "wire-B")
	})

	b.Run(name+"/decode/protobuf", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(wire)))
		for i := 0; i < b.N; i++ {
			out := newOut()
			if err := proto.Unmarshal(wire, out); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkRTB_64_PB adds a protobuf arm to the 64-request RTB fixture,
// matching BenchmarkRTB_64 so wire sizes are directly comparable.
func BenchmarkRTB_64_PB(b *testing.B) {
	batch := mkRTBBatch(64)
	pb := toPBRTBBatch(batch)
	runProtobufArm(b, "RTB_64", pb, func() proto.Message { return new(benchpb.RTBBatch) })
}

// BenchmarkRTB_1024_PB adds a protobuf arm to the 1024-request RTB fixture.
func BenchmarkRTB_1024_PB(b *testing.B) {
	batch := mkRTBBatch(1024)
	pb := toPBRTBBatch(batch)
	runProtobufArm(b, "RTB_1024", pb, func() proto.Message { return new(benchpb.RTBBatch) })
}

// BenchmarkIoT_PB benches protobuf on the IoT fixture at the SAME size as
// BenchmarkIoT_32x256 so the wire-B is directly comparable to the qdf arms.
func BenchmarkIoT_PB(b *testing.B) {
	batch := mkIoTBatch(32, 256)
	pb := toPBIoTBatch(batch)
	runProtobufArm(b, "IoT_32x256", pb, func() proto.Message { return new(benchpb.IoTBatchPB) })
}

// BenchmarkOTLP_PB benches protobuf on the OTLP fixture at the SAME size as
// BenchmarkOTLP_4x512 so the wire-B is directly comparable to the qdf arms.
func BenchmarkOTLP_PB(b *testing.B) {
	batch := mkOTLPBatch(4, 512)
	pb := toPBTraceExport(batch)
	runProtobufArm(b, "OTLP_4x512", pb, func() proto.Message { return new(benchpb.TraceExportPB) })
}

// BenchmarkLogs_PB benches protobuf on the log batch fixture (1024 records).
func BenchmarkLogs_PB(b *testing.B) {
	batch := mkLogBatch(1024)
	pb := toPBLogBatch(batch)
	runProtobufArm(b, "Logs_1024", pb, func() proto.Message { return new(benchpb.LogBatchPB) })
}

// BenchmarkEvents_PB benches protobuf on the event batch fixture (1024
// records).
func BenchmarkEvents_PB(b *testing.B) {
	batch := mkEventBatch(1024)
	pb := toPBEventBatch(batch)
	runProtobufArm(b, "Events_1024", pb, func() proto.Message { return new(benchpb.EventBatchPB) })
}

// ── Roundtrip sanity tests ────────────────────────────────────────────────

// TestPB_RTB_Roundtrip verifies that the RTB conversion + proto
// marshal/unmarshal survives a round-trip and spot-checks key fields.
func TestPB_RTB_Roundtrip(t *testing.T) {
	batch := mkRTBBatch(32)
	pb := toPBRTBBatch(batch)

	wire, err := proto.Marshal(pb)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got benchpb.RTBBatch
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Requests) != len(batch) {
		t.Fatalf("len mismatch: want %d got %d", len(batch), len(got.Requests))
	}
	for i, req := range got.Requests {
		if req.Id != batch[i].ID {
			t.Errorf("[%d] ID: want %q got %q", i, batch[i].ID, req.Id)
		}
		if req.Dev.Geo.Country != batch[i].Dev.Geo.Country {
			t.Errorf("[%d] Country: want %q got %q", i, batch[i].Dev.Geo.Country, req.Dev.Geo.Country)
		}
	}
	t.Logf("RTB_32: wire=%d B, %d requests OK", len(wire), len(got.Requests))
}

// TestPB_IoT_Roundtrip verifies the IoT conversion roundtrip and spot-checks
// a temperature value.
func TestPB_IoT_Roundtrip(t *testing.T) {
	batch := mkIoTBatch(4, 64)
	pb := toPBIoTBatch(batch)

	wire, err := proto.Marshal(pb)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got benchpb.IoTBatchPB
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Devices) != len(batch.Devices) {
		t.Fatalf("len mismatch: want %d got %d", len(batch.Devices), len(got.Devices))
	}
	for i, dev := range got.Devices {
		orig := batch.Devices[i]
		if dev.DeviceId != orig.DeviceID {
			t.Errorf("[%d] DeviceID: want %q got %q", i, orig.DeviceID, dev.DeviceId)
		}
		if len(dev.Temp) != len(orig.Temp) {
			t.Errorf("[%d] Temp len: want %d got %d", i, len(orig.Temp), len(dev.Temp))
			continue
		}
		if dev.Temp[0] != orig.Temp[0] {
			t.Errorf("[%d] Temp[0]: want %f got %f", i, orig.Temp[0], dev.Temp[0])
		}
	}
	t.Logf("IoT_4x64: wire=%d B, %d devices OK", len(wire), len(got.Devices))
}

// TestPB_OTLP_Roundtrip verifies the OTLP conversion roundtrip and
// spot-checks a span name.
func TestPB_OTLP_Roundtrip(t *testing.T) {
	batch := mkOTLPBatch(5, 10)
	pb := toPBTraceExport(batch)

	wire, err := proto.Marshal(pb)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got benchpb.TraceExportPB
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.ResourceSpans) != len(batch.ResourceSpans) {
		t.Fatalf("ResourceSpans len: want %d got %d", len(batch.ResourceSpans), len(got.ResourceSpans))
	}
	for i, rs := range got.ResourceSpans {
		if len(rs.Scopes) == 0 {
			t.Errorf("[%d] no scopes", i)
			continue
		}
		orig := batch.ResourceSpans[i]
		if rs.Scopes[0].Spans[0].Name != orig.Scopes[0].Spans[0].Name {
			t.Errorf("[%d] span name: want %q got %q",
				i, orig.Scopes[0].Spans[0].Name, rs.Scopes[0].Spans[0].Name)
		}
	}
	t.Logf("OTLP_5x10: wire=%d B, %d resource spans OK", len(wire), len(got.ResourceSpans))
}

// TestPB_Logs_Roundtrip verifies the log batch conversion roundtrip and
// spot-checks a service name.
func TestPB_Logs_Roundtrip(t *testing.T) {
	batch := mkLogBatch(64)
	pb := toPBLogBatch(batch)

	wire, err := proto.Marshal(pb)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got benchpb.LogBatchPB
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Records) != len(batch.Records) {
		t.Fatalf("len: want %d got %d", len(batch.Records), len(got.Records))
	}
	for i, rec := range got.Records {
		orig := batch.Records[i]
		if rec.Service != orig.Service {
			t.Errorf("[%d] Service: want %q got %q", i, orig.Service, rec.Service)
		}
		if rec.TraceId != orig.TraceID {
			t.Errorf("[%d] TraceID: want %q got %q", i, orig.TraceID, rec.TraceId)
		}
	}
	t.Logf("Logs_64: wire=%d B, %d records OK", len(wire), len(got.Records))
}

// TestPB_Events_Roundtrip verifies the event batch conversion roundtrip and
// spot-checks a payload byte.
func TestPB_Events_Roundtrip(t *testing.T) {
	batch := mkEventBatch(64)
	pb := toPBEventBatch(batch)

	wire, err := proto.Marshal(pb)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got benchpb.EventBatchPB
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Records) != len(batch.Records) {
		t.Fatalf("len: want %d got %d", len(batch.Records), len(got.Records))
	}
	for i, rec := range got.Records {
		orig := batch.Records[i]
		if rec.Source != orig.Source {
			t.Errorf("[%d] Source: want %q got %q", i, orig.Source, rec.Source)
		}
		if len(rec.Payload) != len(orig.Payload) {
			t.Errorf("[%d] Payload len: want %d got %d", i, len(orig.Payload), len(rec.Payload))
			continue
		}
		if len(orig.Payload) > 0 && rec.Payload[0] != orig.Payload[0] {
			t.Errorf("[%d] Payload[0]: want %d got %d", i, orig.Payload[0], rec.Payload[0])
		}
	}
	t.Logf("Events_64: wire=%d B, %d records OK", len(wire), len(got.Records))
}
