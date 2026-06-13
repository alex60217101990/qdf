package qdf

import (
	"bytes"
	"testing"
)

// bigFastMarshaler emits a large, highly compressible Fast-format body — large
// enough that the OptCompression rANS pass would fire on it if it were allowed
// to. It is the regression guard for "rANS must not reframe a Marshaler body".
type bigFastMarshaler struct{ n int }

func (m *bigFastMarshaler) MarshalQDF(dst []byte) ([]byte, error) {
	for i := 0; i < m.n; i++ {
		dst = append(dst, 'A')
	}
	return dst, nil
}
func (m *bigFastMarshaler) UnmarshalQDF(src []byte) (int, error) {
	m.n = len(src)
	return len(src), nil
}

// TestMarshaler_AlwaysFastFraming_LargeBody extends the contract to a body big
// enough to trip the entropy pass: even under OptCompression the wire must stay
// Fast-framed (flag 0) and byte-identical across opts. (The small-body sibling
// passes trivially because rANS never fires; this one would catch a reframe.)
func TestMarshaler_AlwaysFastFraming_LargeBody(t *testing.T) {
	in := &bigFastMarshaler{n: 5000}
	var fast []byte
	for _, opts := range []Options{OptSpeed, OptBalanced, OptCompression} {
		b, err := Marshal(in, opts)
		if err != nil {
			t.Fatalf("opts=%d marshal: %v", opts, err)
		}
		if b[4] != 0 {
			t.Fatalf("opts=%d: large Marshaler wire flag=%08b, want 0 — the entropy pass reframed it", opts, b[4])
		}
		if fast == nil {
			fast = b
		} else if !bytes.Equal(b, fast) {
			t.Fatalf("opts=%d: large Marshaler wire not opts-invariant (rANS reframe)", opts)
		}
	}
	var out bigFastMarshaler
	if err := Unmarshal(fast, &out); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if out.n != 5000 {
		t.Fatalf("round-trip body length %d != 5000", out.n)
	}
}

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
