package qdf

import "testing"

// TestMarshaler_AlwaysFastFraming pins a best-UX contract: a type implementing
// Marshaler emits its own (Fast-format) body regardless of the requested
// Options, so its wire MUST be framed as Fast (header flag byte 0). Otherwise
// the header lies — claiming FlagDense/FlagQPack over a Fast body — which makes
// UnmarshalDirect take an unnecessary reflect fallback and a decoder allocate
// dense state it never uses. The body is identical across opts (the custom
// MarshalQDF ignores them); only the framing must stay honest.
func TestMarshaler_AlwaysFastFraming(t *testing.T) {
	in := directSample{ID: 7, Name: "x"}
	var fastBody []byte
	for _, opts := range []Options{OptSpeed, OptQPack, OptBalanced, OptCompression} {
		b, err := Marshal(&in, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		if b[4] != 0 {
			t.Fatalf("opts=%d: Marshaler wire flag=%08b, want 0 (Fast) — body is the custom Fast format", opts, b[4])
		}
		if fastBody == nil {
			fastBody = b
		} else if string(b) != string(fastBody) {
			t.Fatalf("opts=%d: Marshaler wire differs across opts (body must be opts-invariant)", opts)
		}
		// UnmarshalDirect decodes directly — no dense fallback needed.
		var out directSample
		if err := UnmarshalDirect(b, &out); err != nil {
			t.Fatalf("opts=%d UnmarshalDirect: %v", opts, err)
		}
		if out != in {
			t.Fatalf("opts=%d roundtrip mismatch: got %+v want %+v", opts, out, in)
		}
	}
}
