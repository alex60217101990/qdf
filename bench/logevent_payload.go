package bench

import (
	"math/rand/v2"
)

// LogRecord is a high-cardinality structured log entry.
// Level is an int enum (low-card → dict/RLE friendly).
// TraceID/SpanID are hex strings (high-card → raw).
// Service/Host are repeated from small sets (dict wins).
// Fields is a mixed-key map.
type LogRecord struct {
	Ts      int64             `json:"ts" msgpack:"ts" qdf:"ts"`
	Level   int               `json:"level" msgpack:"level" qdf:"level"`
	Service string            `json:"service" msgpack:"service" qdf:"service"`
	Host    string            `json:"host" msgpack:"host" qdf:"host"`
	Message string            `json:"message" msgpack:"message" qdf:"message"`
	TraceID string            `json:"trace_id" msgpack:"trace_id" qdf:"trace_id"`
	SpanID  string            `json:"span_id" msgpack:"span_id" qdf:"span_id"`
	Fields  map[string]string `json:"fields" msgpack:"fields" qdf:"fields"`
}

// EventRecord is a compact event/metric entry with a binary payload.
// Type is a low-card int enum (RLE friendly).
// Source repeats from a small set (dict wins).
type EventRecord struct {
	Ts      int64  `json:"ts" msgpack:"ts" qdf:"ts"`
	Type    int    `json:"type" msgpack:"type" qdf:"type"`
	Source  string `json:"source" msgpack:"source" qdf:"source"`
	Payload []byte `json:"payload" msgpack:"payload" qdf:"payload"`
}

// LogBatchLE is a batch of LogRecord values.
// The name avoids collision with the existing LogBatch (which uses LogEntry).
type LogBatchLE struct {
	Records []LogRecord `json:"records" msgpack:"records" qdf:"records"`
}

// EventBatch is a batch of EventRecord values.
type EventBatch struct {
	Records []EventRecord `json:"records" msgpack:"records" qdf:"records"`
}

// mkLogBatch generates a deterministic batch of n LogRecord values.
//
// The fixture exercises:
//   - Level int enum (0–4, low-card) → dict/RLE wins
//   - Service/Host from small sets → dict wins
//   - TraceID (32-char hex) / SpanID (16-char hex) unique per entry → high-card
//   - Message from a small template set → dict wins
//   - Fields map with repeated keys and mixed values
func mkLogBatch(n int) LogBatchLE {
	rng := rand.New(rand.NewPCG(0xabcdef01, 0x23456789))

	services := []string{"api", "worker", "scheduler", "ingest", "edge", "auth", "payment"}
	hosts := []string{"host-a", "host-b", "host-c", "host-d", "host-e", "host-f"}
	messages := []string{
		"request received", "request handled", "cache miss",
		"db timeout", "panic recovered", "rate limit hit",
		"connection refused", "authentication failed",
	}
	fieldKeys := []string{
		"http.method", "http.status", "db.name", "error.type",
		"user.id", "region", "env",
	}
	fieldVals := []string{
		"GET", "POST", "PUT", "DELETE",
		"200", "201", "400", "401", "403", "500", "503",
		"users_db", "orders_db", "timeout", "panic",
		"eu-west", "us-east", "production", "staging",
	}

	baseTs := int64(1_700_000_000_000_000_000)
	records := make([]LogRecord, n)
	for i := range records {
		nFields := 2 + rng.IntN(4) // 2–5 fields
		fields := make(map[string]string, nFields)
		for range nFields {
			fields[fieldKeys[rng.IntN(len(fieldKeys))]] = fieldVals[rng.IntN(len(fieldVals))]
		}
		records[i] = LogRecord{
			Ts:      baseTs + int64(i)*int64(1e6), // 1ms apart
			Level:   rng.IntN(5),                  // 0=TRACE,1=DEBUG,2=INFO,3=WARN,4=ERROR
			Service: services[rng.IntN(len(services))],
			Host:    hosts[rng.IntN(len(hosts))],
			Message: messages[rng.IntN(len(messages))],
			TraceID: logeventHex(rng, 32),
			SpanID:  logeventHex(rng, 16),
			Fields:  fields,
		}
	}
	return LogBatchLE{Records: records}
}

// mkEventBatch generates a deterministic batch of n EventRecord values.
//
// The fixture exercises:
//   - Type int enum (0–7, low-card) → dict/RLE wins
//   - Source from a small set → dict wins
//   - Payload as raw bytes (8–64 bytes, random) → no compression benefit
//   - Monotonic timestamps → Delta+FOR friendly
func mkEventBatch(n int) EventBatch {
	rng := rand.New(rand.NewPCG(0x99aabbcc, 0xddeeff00))

	sources := []string{
		"prometheus", "statsd", "opentelemetry", "datadog",
		"cloudwatch", "grafana-agent",
	}

	baseTs := int64(1_700_000_000_000_000_000)
	records := make([]EventRecord, n)
	for i := range records {
		payloadLen := 8 + rng.IntN(57) // 8–64 bytes
		payload := make([]byte, payloadLen)
		for j := range payload {
			payload[j] = byte(rng.IntN(256))
		}
		records[i] = EventRecord{
			Ts:      baseTs + int64(i)*int64(5e8), // 500ms apart
			Type:    rng.IntN(8),                  // 0–7
			Source:  sources[rng.IntN(len(sources))],
			Payload: payload,
		}
	}
	return EventBatch{Records: records}
}

// logeventHex returns a deterministic n-char lowercase hex string.
func logeventHex(rng *rand.Rand, n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexChars[rng.IntN(16)]
	}
	return string(b)
}
