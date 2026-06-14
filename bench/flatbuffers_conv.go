package bench

import (
	"slices"

	benchfbs "github.com/alex60217101990/qdf/bench/fbs"
	flatbuffers "github.com/google/flatbuffers/go"
)

// ── builder helpers ───────────────────────────────────────────────────────

// fbKVs builds a vector of KeyValue tables from a map[string]string.
// Must be called before the enclosing table is started (flatbuffers bottom-up).
func fbKVs(b *flatbuffers.Builder, m map[string]string) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, 0, len(m))
	for k, v := range m {
		ko := b.CreateString(k)
		vo := b.CreateString(v)
		benchfbs.KeyValueStart(b)
		benchfbs.KeyValueAddKey(b, ko)
		benchfbs.KeyValueAddValue(b, vo)
		offsets = append(offsets, benchfbs.KeyValueEnd(b))
	}
	b.StartVector(4, len(offsets), 4)
	for _, offset := range slices.Backward(offsets) {
		b.PrependUOffsetT(offset)
	}
	return b.EndVector(len(offsets))
}

// fbStrings builds a vector of string offsets.
func fbStrings(b *flatbuffers.Builder, ss []string) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, len(ss))
	for i, s := range ss {
		offsets[i] = b.CreateString(s)
	}
	b.StartVector(4, len(offsets), 4)
	for _, offset := range slices.Backward(offsets) {
		b.PrependUOffsetT(offset)
	}
	return b.EndVector(len(offsets))
}

// ── RTB builders ─────────────────────────────────────────────────────────

func fbBuildGeo(b *flatbuffers.Builder, g Geo) flatbuffers.UOffsetT {
	country := b.CreateString(g.Country)
	benchfbs.GeoStart(b)
	benchfbs.GeoAddCountry(b, country)
	benchfbs.GeoAddLat(b, g.Lat)
	benchfbs.GeoAddLon(b, g.Lon)
	benchfbs.GeoAddGeoType(b, int32(g.Type))
	return benchfbs.GeoEnd(b)
}

func fbBuildDevice(b *flatbuffers.Builder, d Device) flatbuffers.UOffsetT {
	ua := b.CreateString(d.UA)
	ip := b.CreateString(d.IP)
	geo := fbBuildGeo(b, d.Geo)
	benchfbs.DeviceStart(b)
	benchfbs.DeviceAddUa(b, ua)
	benchfbs.DeviceAddIp(b, ip)
	benchfbs.DeviceAddOs(b, int32(d.OS))
	benchfbs.DeviceAddDeviceType(b, int32(d.Type))
	benchfbs.DeviceAddGeo(b, geo)
	return benchfbs.DeviceEnd(b)
}

func fbBuildImpression(b *flatbuffers.Builder, imp Impression) flatbuffers.UOffsetT {
	id := b.CreateString(imp.ID)
	deals := fbStrings(b, imp.DealIDs)
	ext := fbKVs(b, imp.Ext)
	benchfbs.ImpressionStart(b)
	benchfbs.ImpressionAddImpId(b, id)
	benchfbs.ImpressionAddBidFloor(b, imp.BidFloor)
	benchfbs.ImpressionAddW(b, int32(imp.W))
	benchfbs.ImpressionAddH(b, int32(imp.H))
	benchfbs.ImpressionAddDealIds(b, deals)
	benchfbs.ImpressionAddExt(b, ext)
	return benchfbs.ImpressionEnd(b)
}

func fbBuildBidRequest(b *flatbuffers.Builder, req BidRequest) flatbuffers.UOffsetT {
	id := b.CreateString(req.ID)
	cur := fbStrings(b, req.Cur)
	dev := fbBuildDevice(b, req.Dev)
	impOffsets := make([]flatbuffers.UOffsetT, len(req.Imp))
	for i, imp := range req.Imp {
		impOffsets[i] = fbBuildImpression(b, imp)
	}
	b.StartVector(4, len(impOffsets), 4)
	for _, impOffset := range slices.Backward(impOffsets) {
		b.PrependUOffsetT(impOffset)
	}
	imps := b.EndVector(len(impOffsets))
	benchfbs.BidRequestStart(b)
	benchfbs.BidRequestAddReqId(b, id)
	benchfbs.BidRequestAddAt(b, int32(req.At))
	benchfbs.BidRequestAddTmax(b, int32(req.Tmax))
	benchfbs.BidRequestAddImp(b, imps)
	benchfbs.BidRequestAddDev(b, dev)
	benchfbs.BidRequestAddCur(b, cur)
	return benchfbs.BidRequestEnd(b)
}

// FBBuildRTBBatch serialises a []BidRequest into a FlatBuffers buffer.
func FBBuildRTBBatch(batch []BidRequest) []byte {
	b := flatbuffers.NewBuilder(1024 * len(batch))
	reqOffsets := make([]flatbuffers.UOffsetT, len(batch))
	for i, req := range batch {
		reqOffsets[i] = fbBuildBidRequest(b, req)
	}
	b.StartVector(4, len(reqOffsets), 4)
	for _, reqOffset := range slices.Backward(reqOffsets) {
		b.PrependUOffsetT(reqOffset)
	}
	reqs := b.EndVector(len(reqOffsets))
	benchfbs.RTBBatchStart(b)
	benchfbs.RTBBatchAddRequests(b, reqs)
	root := benchfbs.RTBBatchEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

// ── IoT builders ──────────────────────────────────────────────────────────

func fbBuildDeviceReading(b *flatbuffers.Builder, d DeviceReading) flatbuffers.UOffsetT {
	id := b.CreateString(d.DeviceID)
	tags := fbKVs(b, d.Tags)

	// ts vector (int64, 8-byte elements)
	benchfbs.DeviceReadingStartTsVector(b, len(d.Ts))
	for _, v := range slices.Backward(d.Ts) {
		b.PrependInt64(v)
	}
	ts := b.EndVector(len(d.Ts))

	// temp vector (float64)
	benchfbs.DeviceReadingStartTempVector(b, len(d.Temp))
	for _, v := range slices.Backward(d.Temp) {
		b.PrependFloat64(v)
	}
	temp := b.EndVector(len(d.Temp))

	// humidity vector
	benchfbs.DeviceReadingStartHumidityVector(b, len(d.Humidity))
	for _, v := range slices.Backward(d.Humidity) {
		b.PrependFloat64(v)
	}
	humidity := b.EndVector(len(d.Humidity))

	benchfbs.DeviceReadingStart(b)
	benchfbs.DeviceReadingAddDeviceId(b, id)
	benchfbs.DeviceReadingAddTs(b, ts)
	benchfbs.DeviceReadingAddTemp(b, temp)
	benchfbs.DeviceReadingAddHumidity(b, humidity)
	benchfbs.DeviceReadingAddTags(b, tags)
	return benchfbs.DeviceReadingEnd(b)
}

// FBBuildIoTBatch serialises an IoTBatch into a FlatBuffers buffer.
func FBBuildIoTBatch(batch IoTBatch) []byte {
	b := flatbuffers.NewBuilder(1024 * len(batch.Devices))
	devOffsets := make([]flatbuffers.UOffsetT, len(batch.Devices))
	for i, d := range batch.Devices {
		devOffsets[i] = fbBuildDeviceReading(b, d)
	}
	benchfbs.IoTBatchFBStartDevicesVector(b, len(devOffsets))
	for _, devOffset := range slices.Backward(devOffsets) {
		b.PrependUOffsetT(devOffset)
	}
	devs := b.EndVector(len(devOffsets))
	benchfbs.IoTBatchFBStart(b)
	benchfbs.IoTBatchFBAddDevices(b, devs)
	root := benchfbs.IoTBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

// ── OTLP builders ─────────────────────────────────────────────────────────

func fbBuildKVPairs(b *flatbuffers.Builder, kvs []KV) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, len(kvs))
	for i, kv := range kvs {
		ko := b.CreateString(kv.Key)
		vo := b.CreateString(kv.Value)
		benchfbs.KeyValueStart(b)
		benchfbs.KeyValueAddKey(b, ko)
		benchfbs.KeyValueAddValue(b, vo)
		offsets[i] = benchfbs.KeyValueEnd(b)
	}
	b.StartVector(4, len(offsets), 4)
	for _, offset := range slices.Backward(offsets) {
		b.PrependUOffsetT(offset)
	}
	return b.EndVector(len(offsets))
}

func fbBuildSpan(b *flatbuffers.Builder, s Span) flatbuffers.UOffsetT {
	traceID := b.CreateString(s.TraceID)
	spanID := b.CreateString(s.SpanID)
	parentID := b.CreateString(s.ParentID)
	name := b.CreateString(s.Name)
	attrs := fbBuildKVPairs(b, s.Attrs)
	benchfbs.SpanStart(b)
	benchfbs.SpanAddTraceId(b, traceID)
	benchfbs.SpanAddSpanId(b, spanID)
	benchfbs.SpanAddParentId(b, parentID)
	benchfbs.SpanAddName(b, name)
	benchfbs.SpanAddKind(b, int32(s.Kind))
	benchfbs.SpanAddStartNs(b, s.StartNs)
	benchfbs.SpanAddEndNs(b, s.EndNs)
	benchfbs.SpanAddAttrs(b, attrs)
	benchfbs.SpanAddStatus(b, int32(s.Status))
	return benchfbs.SpanEnd(b)
}

func fbBuildScopeSpans(b *flatbuffers.Builder, ss ScopeSpans) flatbuffers.UOffsetT {
	scope := b.CreateString(ss.Scope)
	spanOffsets := make([]flatbuffers.UOffsetT, len(ss.Spans))
	for i, s := range ss.Spans {
		spanOffsets[i] = fbBuildSpan(b, s)
	}
	b.StartVector(4, len(spanOffsets), 4)
	for _, spanOffset := range slices.Backward(spanOffsets) {
		b.PrependUOffsetT(spanOffset)
	}
	spans := b.EndVector(len(spanOffsets))
	benchfbs.ScopeSpansStart(b)
	benchfbs.ScopeSpansAddScope(b, scope)
	benchfbs.ScopeSpansAddSpans(b, spans)
	return benchfbs.ScopeSpansEnd(b)
}

func fbBuildResourceSpans(b *flatbuffers.Builder, rs ResourceSpans) flatbuffers.UOffsetT {
	resource := fbKVs(b, rs.Resource)
	scopeOffsets := make([]flatbuffers.UOffsetT, len(rs.Scopes))
	for i, sc := range rs.Scopes {
		scopeOffsets[i] = fbBuildScopeSpans(b, sc)
	}
	b.StartVector(4, len(scopeOffsets), 4)
	for _, scopeOffset := range slices.Backward(scopeOffsets) {
		b.PrependUOffsetT(scopeOffset)
	}
	scopes := b.EndVector(len(scopeOffsets))
	benchfbs.ResourceSpansStart(b)
	benchfbs.ResourceSpansAddResource(b, resource)
	benchfbs.ResourceSpansAddScopes(b, scopes)
	return benchfbs.ResourceSpansEnd(b)
}

// FBBuildTraceExport serialises a TraceExport into a FlatBuffers buffer.
func FBBuildTraceExport(te TraceExport) []byte {
	b := flatbuffers.NewBuilder(4096)
	rsOffsets := make([]flatbuffers.UOffsetT, len(te.ResourceSpans))
	for i, rs := range te.ResourceSpans {
		rsOffsets[i] = fbBuildResourceSpans(b, rs)
	}
	benchfbs.TraceExportFBStartResourceSpansVector(b, len(rsOffsets))
	for _, rsOffset := range slices.Backward(rsOffsets) {
		b.PrependUOffsetT(rsOffset)
	}
	rsVec := b.EndVector(len(rsOffsets))
	benchfbs.TraceExportFBStart(b)
	benchfbs.TraceExportFBAddResourceSpans(b, rsVec)
	root := benchfbs.TraceExportFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

// ── Log builders ──────────────────────────────────────────────────────────

func fbBuildLogRecord(b *flatbuffers.Builder, r LogRecord) flatbuffers.UOffsetT {
	service := b.CreateString(r.Service)
	host := b.CreateString(r.Host)
	message := b.CreateString(r.Message)
	traceID := b.CreateString(r.TraceID)
	spanID := b.CreateString(r.SpanID)
	fields := fbKVs(b, r.Fields)
	benchfbs.LogRecordStart(b)
	benchfbs.LogRecordAddTs(b, r.Ts)
	benchfbs.LogRecordAddLevel(b, int32(r.Level))
	benchfbs.LogRecordAddService(b, service)
	benchfbs.LogRecordAddHost(b, host)
	benchfbs.LogRecordAddMessage(b, message)
	benchfbs.LogRecordAddTraceId(b, traceID)
	benchfbs.LogRecordAddSpanId(b, spanID)
	benchfbs.LogRecordAddFields(b, fields)
	return benchfbs.LogRecordEnd(b)
}

// FBBuildLogBatch serialises a LogBatchLE into a FlatBuffers buffer.
func FBBuildLogBatch(lb LogBatchLE) []byte {
	b := flatbuffers.NewBuilder(256 * len(lb.Records))
	recOffsets := make([]flatbuffers.UOffsetT, len(lb.Records))
	for i, r := range lb.Records {
		recOffsets[i] = fbBuildLogRecord(b, r)
	}
	benchfbs.LogBatchFBStartRecordsVector(b, len(recOffsets))
	for _, recOffset := range slices.Backward(recOffsets) {
		b.PrependUOffsetT(recOffset)
	}
	recs := b.EndVector(len(recOffsets))
	benchfbs.LogBatchFBStart(b)
	benchfbs.LogBatchFBAddRecords(b, recs)
	root := benchfbs.LogBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}

// ── Event builders ────────────────────────────────────────────────────────

func fbBuildEventRecord(b *flatbuffers.Builder, e EventRecord) flatbuffers.UOffsetT {
	source := b.CreateString(e.Source)
	// [ubyte] payload — use CreateByteVector for efficient inline bytes
	payload := b.CreateByteVector(e.Payload)
	benchfbs.EventRecordStart(b)
	benchfbs.EventRecordAddTs(b, e.Ts)
	benchfbs.EventRecordAddEvType(b, int32(e.Type))
	benchfbs.EventRecordAddSource(b, source)
	benchfbs.EventRecordAddPayload(b, payload)
	return benchfbs.EventRecordEnd(b)
}

// FBBuildEventBatch serialises an EventBatch into a FlatBuffers buffer.
func FBBuildEventBatch(eb EventBatch) []byte {
	b := flatbuffers.NewBuilder(128 * len(eb.Records))
	recOffsets := make([]flatbuffers.UOffsetT, len(eb.Records))
	for i, e := range eb.Records {
		recOffsets[i] = fbBuildEventRecord(b, e)
	}
	benchfbs.EventBatchFBStartRecordsVector(b, len(recOffsets))
	for _, recOffset := range slices.Backward(recOffsets) {
		b.PrependUOffsetT(recOffset)
	}
	recs := b.EndVector(len(recOffsets))
	benchfbs.EventBatchFBStart(b)
	benchfbs.EventBatchFBAddRecords(b, recs)
	root := benchfbs.EventBatchFBEnd(b)
	b.Finish(root)
	return b.FinishedBytes()
}
