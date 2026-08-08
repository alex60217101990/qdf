package bench

import (
	"fmt"
	"math/rand/v2"
	"strconv"
)

// AccessRecord is an HTTP access-log entry, the shape the rest of this suite
// does not cover.
//
// The existing log payload models templated logs: Message comes from a small
// template set (dict wins) and TraceID/SpanID are hex (a 16-symbol alphabet, so
// the alphabet packer wins). Neither exercises the case where a text column is
// high-cardinality AND has no restricted alphabet, but its values still share
// long substrings with each other — request lines that differ only in an id,
// user agents that differ only in a version, error strings that differ only in
// a parameter. That is what the substring codec (FSST) exists for, and with no
// column of this shape the suite cannot show what it is worth.
//
// Every text field below is deliberately of that kind: thousands of distinct
// values, no small dictionary, no restricted alphabet, heavy substring sharing.
type AccessRecord struct {
	Ts        int64  `json:"ts" msgpack:"ts" qdf:"ts"`
	Status    int    `json:"status" msgpack:"status" qdf:"status"`
	Bytes     int64  `json:"bytes" msgpack:"bytes" qdf:"bytes"`
	Request   string `json:"request" msgpack:"request" qdf:"request"`
	Referer   string `json:"referer" msgpack:"referer" qdf:"referer"`
	UserAgent string `json:"user_agent" msgpack:"user_agent" qdf:"user_agent"`
	Error     string `json:"error" msgpack:"error" qdf:"error"`
}

// AccessBatch wraps the slice so the columnar path sees a struct root, matching
// the other payloads in this package.
type AccessBatch struct {
	Records []AccessRecord `json:"records" msgpack:"records" qdf:"records"`
}

func mkAccessBatch(n int) AccessBatch {
	rng := rand.New(rand.NewPCG(0x5eed, 0xa11ce))

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	resources := []string{"users", "orders", "invoices", "sessions", "products", "carts"}
	subs := []string{"", "/items", "/history", "/settings", "/audit-log"}
	agents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.86 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:%d.0) Gecko/20100101 Firefox/%d.0",
		"Mozilla/5.0 (iPhone; CPU iPhone OS %d_%d like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
	}
	// Each error template takes (string, int) in that order, so one call site
	// serves them all.
	errs := []string{
		"",
		"upstream connect error or disconnect/reset before headers on %s, retried %d times",
		"context deadline exceeded while calling %s after %d ms",
		"permission denied on resource %s for service-account-%d",
		"validation failed: field %s must be a positive integer, got %d",
	}

	// A uuid-shaped id, distinct per row, so the request lines cannot collapse
	// into a dictionary.
	uuid := func() string {
		const hex = "0123456789abcdef"
		b := make([]byte, 36)
		for i := range b {
			switch i {
			case 8, 13, 18, 23:
				b[i] = '-'
			default:
				b[i] = hex[rng.IntN(16)]
			}
		}
		return string(b)
	}

	recs := make([]AccessRecord, n)
	ts := int64(1700000000)
	for i := range recs {
		res := resources[rng.IntN(len(resources))]
		req := methods[rng.IntN(len(methods))] + " /api/v2/" + res + "/" + uuid() +
			subs[rng.IntN(len(subs))] + "?page=" + strconv.Itoa(rng.IntN(50)) +
			"&limit=" + strconv.Itoa(25*(1+rng.IntN(4))) + " HTTP/1.1"

		ref := "https://app.example.com/dashboard/" + res + "/" + strconv.Itoa(rng.IntN(9999))

		ua := fmt.Sprintf(agents[rng.IntN(len(agents))], 110+rng.IntN(20), 5000+rng.IntN(999))

		var e string
		if tpl := errs[rng.IntN(len(errs))]; tpl != "" {
			e = fmt.Sprintf(tpl, res, rng.IntN(30000))
		}

		// Status is low-cardinality and skewed, the way it is in production.
		st := 200
		switch x := rng.IntN(100); {
		case x >= 97:
			st = 500
		case x >= 92:
			st = 404
		case x >= 88:
			st = 401
		}

		ts += int64(rng.IntN(40))
		recs[i] = AccessRecord{
			Ts:        ts,
			Status:    st,
			Bytes:     int64(200 + rng.IntN(60000)),
			Request:   req,
			Referer:   ref,
			UserAgent: ua,
			Error:     e,
		}
	}
	return AccessBatch{Records: recs}
}
