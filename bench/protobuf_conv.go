package bench

import (
	benchpb "github.com/alex60217101990/qdf/bench/pb"
)

// toPBGeo converts a Geo to its protobuf representation.
func toPBGeo(g Geo) *benchpb.Geo {
	return &benchpb.Geo{
		Country: g.Country,
		Lat:     g.Lat,
		Lon:     g.Lon,
		Type:    int32(g.Type),
	}
}

// toPBDevice converts a Device to its protobuf representation.
func toPBDevice(d Device) *benchpb.Device {
	return &benchpb.Device{
		Ua:   d.UA,
		Ip:   d.IP,
		Os:   int32(d.OS),
		Type: int32(d.Type),
		Geo:  toPBGeo(d.Geo),
	}
}

// toPBImpression converts an Impression to its protobuf representation.
func toPBImpression(imp Impression) *benchpb.Impression {
	return &benchpb.Impression{
		Id:       imp.ID,
		BidFloor: imp.BidFloor,
		W:        int32(imp.W),
		H:        int32(imp.H),
		DealIds:  imp.DealIDs,
		Ext:      imp.Ext,
	}
}

// toPBBidRequest converts a BidRequest to its protobuf representation.
func toPBBidRequest(req BidRequest) *benchpb.BidRequest {
	imps := make([]*benchpb.Impression, len(req.Imp))
	for i, imp := range req.Imp {
		imps[i] = toPBImpression(imp)
	}
	return &benchpb.BidRequest{
		Id:   req.ID,
		At:   int32(req.At),
		Tmax: int32(req.Tmax),
		Imp:  imps,
		Dev:  toPBDevice(req.Dev),
		Cur:  req.Cur,
	}
}

// toPBRTBBatch converts a []BidRequest batch to its protobuf representation.
func toPBRTBBatch(batch []BidRequest) *benchpb.RTBBatch {
	reqs := make([]*benchpb.BidRequest, len(batch))
	for i, req := range batch {
		reqs[i] = toPBBidRequest(req)
	}
	return &benchpb.RTBBatch{Requests: reqs}
}

// toPBDeviceReading converts a DeviceReading to its protobuf representation.
func toPBDeviceReading(d DeviceReading) *benchpb.DeviceReading {
	return &benchpb.DeviceReading{
		DeviceId: d.DeviceID,
		Ts:       d.Ts,
		Temp:     d.Temp,
		Humidity: d.Humidity,
		Tags:     d.Tags,
	}
}

// toPBIoTBatch converts an IoTBatch to its protobuf representation.
func toPBIoTBatch(b IoTBatch) *benchpb.IoTBatchPB {
	devs := make([]*benchpb.DeviceReading, len(b.Devices))
	for i, d := range b.Devices {
		devs[i] = toPBDeviceReading(d)
	}
	return &benchpb.IoTBatchPB{Devices: devs}
}

// toPBKV converts a KV to its protobuf representation.
func toPBKV(kv KV) *benchpb.KV {
	return &benchpb.KV{Key: kv.Key, Value: kv.Value}
}

// toPBSpan converts a Span to its protobuf representation.
func toPBSpan(s Span) *benchpb.Span {
	attrs := make([]*benchpb.KV, len(s.Attrs))
	for i, a := range s.Attrs {
		attrs[i] = toPBKV(a)
	}
	return &benchpb.Span{
		TraceId:  s.TraceID,
		SpanId:   s.SpanID,
		ParentId: s.ParentID,
		Name:     s.Name,
		Kind:     int32(s.Kind),
		StartNs:  s.StartNs,
		EndNs:    s.EndNs,
		Attrs:    attrs,
		Status:   int32(s.Status),
	}
}

// toPBScopeSpans converts a ScopeSpans to its protobuf representation.
func toPBScopeSpans(ss ScopeSpans) *benchpb.ScopeSpans {
	spans := make([]*benchpb.Span, len(ss.Spans))
	for i, s := range ss.Spans {
		spans[i] = toPBSpan(s)
	}
	return &benchpb.ScopeSpans{Scope: ss.Scope, Spans: spans}
}

// toPBResourceSpans converts a ResourceSpans to its protobuf representation.
func toPBResourceSpans(rs ResourceSpans) *benchpb.ResourceSpans {
	scopes := make([]*benchpb.ScopeSpans, len(rs.Scopes))
	for i, sc := range rs.Scopes {
		scopes[i] = toPBScopeSpans(sc)
	}
	return &benchpb.ResourceSpans{Resource: rs.Resource, Scopes: scopes}
}

// toPBTraceExport converts a TraceExport to its protobuf representation.
func toPBTraceExport(te TraceExport) *benchpb.TraceExportPB {
	rs := make([]*benchpb.ResourceSpans, len(te.ResourceSpans))
	for i, r := range te.ResourceSpans {
		rs[i] = toPBResourceSpans(r)
	}
	return &benchpb.TraceExportPB{ResourceSpans: rs}
}

// toPBLogRecord converts a LogRecord to its protobuf representation.
func toPBLogRecord(r LogRecord) *benchpb.LogRecord {
	return &benchpb.LogRecord{
		Ts:      r.Ts,
		Level:   int32(r.Level),
		Service: r.Service,
		Host:    r.Host,
		Message: r.Message,
		TraceId: r.TraceID,
		SpanId:  r.SpanID,
		Fields:  r.Fields,
	}
}

// toPBLogBatch converts a LogBatchLE to its protobuf representation.
func toPBLogBatch(lb LogBatchLE) *benchpb.LogBatchPB {
	recs := make([]*benchpb.LogRecord, len(lb.Records))
	for i, r := range lb.Records {
		recs[i] = toPBLogRecord(r)
	}
	return &benchpb.LogBatchPB{Records: recs}
}

// toPBEventRecord converts an EventRecord to its protobuf representation.
func toPBEventRecord(e EventRecord) *benchpb.EventRecord {
	return &benchpb.EventRecord{
		Ts:      e.Ts,
		Type:    int32(e.Type),
		Source:  e.Source,
		Payload: e.Payload,
	}
}

// toPBEventBatch converts an EventBatch to its protobuf representation.
func toPBEventBatch(eb EventBatch) *benchpb.EventBatchPB {
	recs := make([]*benchpb.EventRecord, len(eb.Records))
	for i, e := range eb.Records {
		recs[i] = toPBEventRecord(e)
	}
	return &benchpb.EventBatchPB{Records: recs}
}
