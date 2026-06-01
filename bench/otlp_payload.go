package bench

import (
	"fmt"
	"math/rand/v2"
)

// KV is a key-value attribute pair used in OTLP spans.
type KV struct {
	Key   string `json:"key" msgpack:"key" qdf:"key"`
	Value string `json:"value" msgpack:"value" qdf:"value"`
}

// Span represents a single OTLP trace span.
type Span struct {
	TraceID  string `json:"trace_id" msgpack:"trace_id" qdf:"trace_id"`
	SpanID   string `json:"span_id" msgpack:"span_id" qdf:"span_id"`
	ParentID string `json:"parent_id" msgpack:"parent_id" qdf:"parent_id"`
	Name     string `json:"name" msgpack:"name" qdf:"name"`
	Kind     int    `json:"kind" msgpack:"kind" qdf:"kind"`
	StartNs  int64  `json:"start_ns" msgpack:"start_ns" qdf:"start_ns"`
	EndNs    int64  `json:"end_ns" msgpack:"end_ns" qdf:"end_ns"`
	Attrs    []KV   `json:"attrs" msgpack:"attrs" qdf:"attrs"`
	Status   int    `json:"status" msgpack:"status" qdf:"status"`
}

// ScopeSpans groups spans under a single instrumentation scope.
type ScopeSpans struct {
	Scope string `json:"scope" msgpack:"scope" qdf:"scope"`
	Spans []Span `json:"spans" msgpack:"spans" qdf:"spans"`
}

// ResourceSpans groups scopes under a single resource (service instance).
type ResourceSpans struct {
	Resource map[string]string `json:"resource" msgpack:"resource" qdf:"resource"`
	Scopes   []ScopeSpans      `json:"scopes" msgpack:"scopes" qdf:"scopes"`
}

// TraceExport is the top-level OTLP-style export batch.
type TraceExport struct {
	ResourceSpans []ResourceSpans `json:"resource_spans" msgpack:"resource_spans" qdf:"resource_spans"`
}

// mkOTLPBatch generates a deterministic OTLP-style trace batch.
//
// The fixture exercises:
//   - High-cardinality hex trace/span/parent IDs → raw storage
//   - Repeated span Name from a small set → dict wins
//   - Repeated attr keys with mixed values → dict on keys
//   - Kind/Status int enums (low-card) → RLE/dict friendly
//   - Monotonic-ish StartNs/EndNs per resource → Delta+FOR friendly
func mkOTLPBatch(resources, spansPerScope int) TraceExport {
	rng := rand.New(rand.NewPCG(0x1234abcd, 0x5678ef01))

	serviceNames := []string{
		"frontend", "api-gateway", "auth-service", "payment-svc",
		"inventory-svc", "notification-svc",
	}
	scopeNames := []string{
		"go.opentelemetry.io/otel/sdk/trace",
		"github.com/grpc-ecosystem/go-grpc-middleware",
		"net/http",
	}
	opNames := []string{
		"HTTP GET /api/v1/users",
		"HTTP POST /api/v1/orders",
		"gRPC /auth.AuthService/Validate",
		"db.query",
		"cache.get",
		"kafka.publish",
		"redis.set",
		"HTTP PUT /api/v1/inventory",
	}
	attrKeys := []string{
		"http.method", "http.status_code", "db.system", "net.peer.name",
		"rpc.service", "messaging.system", "cache.hit", "error",
	}
	attrVals := []string{
		"GET", "POST", "PUT", "200", "201", "400", "500",
		"postgresql", "redis", "kafka", "true", "false",
		"db.example.com", "cache.example.com",
	}
	envVals := []string{"production", "staging", "canary"}
	versionVals := []string{"v1.2.3", "v1.3.0", "v2.0.0-rc1"}

	baseNs := int64(1_700_000_000_000_000_000)

	rs := make([]ResourceSpans, resources)
	for r := range rs {
		svcName := serviceNames[rng.IntN(len(serviceNames))]
		resource := map[string]string{
			"service.name":    svcName,
			"service.version": versionVals[rng.IntN(len(versionVals))],
			"deployment.env":  envVals[rng.IntN(len(envVals))],
		}

		numScopes := 1 + rng.IntN(2) // 1 or 2 scopes per resource
		scopes := make([]ScopeSpans, numScopes)
		for sc := range scopes {
			spans := make([]Span, spansPerScope)
			// Each scope shares a common trace ID for realism
			traceID := otlpHex(rng, 32)
			tNs := baseNs + int64(r)*int64(1e12) + int64(sc)*int64(1e10)

			for i := range spans {
				nAttrs := 3 + rng.IntN(4) // 3–6 attrs
				attrs := make([]KV, nAttrs)
				for a := range attrs {
					attrs[a] = KV{
						Key:   attrKeys[rng.IntN(len(attrKeys))],
						Value: attrVals[rng.IntN(len(attrVals))],
					}
				}

				dur := int64(1_000_000 + rng.IntN(50_000_000)) // 1ms–50ms in ns
				startNs := tNs + int64(i)*int64(2e8)           // ~200ms apart
				spans[i] = Span{
					TraceID:  traceID,
					SpanID:   otlpHex(rng, 16),
					ParentID: fmt.Sprintf("%016x", rng.Uint64()),
					Name:     opNames[rng.IntN(len(opNames))],
					Kind:     rng.IntN(6), // 0–5
					StartNs:  startNs,
					EndNs:    startNs + dur,
					Attrs:    attrs,
					Status:   rng.IntN(3), // 0–2
				}
			}

			scopes[sc] = ScopeSpans{
				Scope: scopeNames[rng.IntN(len(scopeNames))],
				Spans: spans,
			}
		}

		rs[r] = ResourceSpans{
			Resource: resource,
			Scopes:   scopes,
		}
	}
	return TraceExport{ResourceSpans: rs}
}

// otlpHex returns a deterministic n-char lowercase hex string.
func otlpHex(rng *rand.Rand, n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexChars[rng.IntN(16)]
	}
	return string(b)
}
