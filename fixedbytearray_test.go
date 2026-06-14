package qdf

import (
	"testing"
)

type idRec struct {
	Trace [16]byte `qdf:"trace"`
	Span  [8]byte  `qdf:"span"`
}

// TestFixedByteArrayCompact asserts a [N]byte array field encodes as one
// contiguous binary blob (like []byte) instead of N tagged elements — so real
// ID bytes (0..255, half of them >=128) do not bloat to ~2 bytes each — and
// decodes in place with zero allocation, bit-exact.
func TestFixedByteArrayCompact(t *testing.T) {
	var in idRec
	for i := range in.Trace {
		in.Trace[i] = byte(0x80 + i) // high bytes: per-element encoding would double them
	}
	for i := range in.Span {
		in.Span[i] = byte(0xF0 + i)
	}

	// Equivalent struct with []byte fields = the exact flat-bin target wire.
	type idRecSlice struct {
		Trace []byte `qdf:"trace"`
		Span  []byte `qdf:"span"`
	}
	ref := idRecSlice{Trace: in.Trace[:], Span: in.Span[:]}

	for _, opt := range []Options{OptSpeed, OptBalanced, OptCompression} {
		buf, err := Marshal(&in, opt)
		if err != nil {
			t.Fatalf("opt %d marshal: %v", opt, err)
		}
		refBuf, _ := Marshal(&ref, opt)
		// A [N]byte field must encode to the SAME flat blob as a []byte of length
		// N — not N tagged elements (which double every byte >= 128).
		if len(buf) != len(refBuf) {
			t.Fatalf("opt %d: [N]byte wire %d != []byte wire %d — not a flat blob", opt, len(buf), len(refBuf))
		}

		var out idRec
		if err := Unmarshal(buf, &out); err != nil {
			t.Fatalf("opt %d unmarshal: %v", opt, err)
		}
		if out != in {
			t.Fatalf("opt %d round-trip mismatch", opt)
		}
	}

	// Decode allocates nothing for the array bodies (they land in the inline
	// struct array), only whatever the decoder framing needs.
	buf, _ := Marshal(&in, OptSpeed)
	allocs := testing.AllocsPerRun(100, func() {
		var out idRec
		_ = Unmarshal(buf, &out)
	})
	if allocs > 1 {
		t.Fatalf("decode allocs = %.0f, want <=1 (array bodies must not allocate)", allocs)
	}
}
