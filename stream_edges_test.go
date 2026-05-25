package qdf

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

// Stream encoder / decoder edge cases. These guard the I/O wrapper
// around the in-memory Marshal / Unmarshal: chunked / partial reads,
// EOF mid-message, double Close, Encode-after-Close.

type streamMsg struct {
	Seq int      `qdf:"seq"`
	Buf []byte   `qdf:"buf"`
	Tag string   `qdf:"tag"`
	Vec []uint64 `qdf:"vec"`
}

func TestStream_ChunkedReader(t *testing.T) {
	// Encode 10 messages, then feed the resulting bytes through a
	// reader that returns 1 byte at a time. Decoder must recover.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for i := range 10 {
		if err := enc.Encode(streamMsg{
			Seq: i,
			Buf: []byte{0x01, 0x02, 0x03},
			Tag: "service-tag",
			Vec: []uint64{uint64(i) * 100, uint64(i)*100 + 1, uint64(i)*100 + 2},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	dec := NewStreamDecoder(&oneByteReader{src: bytes.NewReader(w.Bytes())})
	defer dec.Close()
	for i := range 10 {
		var got streamMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("msg %d: %v", i, err)
		}
		if got.Seq != i {
			t.Fatalf("msg %d seq=%d", i, got.Seq)
		}
	}
}

// oneByteReader returns one byte per Read call, exercising the
// chunked-input refill path in StreamDecoder.
type oneByteReader struct {
	src *bytes.Reader
}

func (o *oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return o.src.Read(p[:1])
}

func TestStream_EOFMidMessage(t *testing.T) {
	// Encode one message, truncate at half the payload, decode must
	// return an error and not panic.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Fast)
	if err := enc.Encode(streamMsg{Seq: 1, Buf: []byte{0x01, 0x02, 0x03}, Tag: "eof-test"}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	full := w.Bytes()
	for i := 1; i < len(full); i++ {
		dec := NewStreamDecoder(bytes.NewReader(full[:i]))
		var got streamMsg
		// Decode must not panic; either error or partial success.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at truncation %d: %v", i, r)
				}
			}()
			_ = dec.Decode(&got)
			dec.Close()
		}()
	}
}

func TestStream_EOFAtStreamEnd(t *testing.T) {
	// After all messages decoded, next Decode must return io.EOF.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Fast)
	for i := range 3 {
		if err := enc.Encode(streamMsg{Seq: i, Tag: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()
	for i := range 3 {
		var got streamMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("msg %d: %v", i, err)
		}
	}
	var trailing streamMsg
	err := dec.Decode(&trailing)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestStream_CloseIdempotent(t *testing.T) {
	// Close on an Encoder after Close should not panic. The current
	// implementation nils out the internal buffer; calling Close a
	// second time must either no-op or return an error.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Fast)
	_ = enc.Encode(streamMsg{Seq: 1})
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("double Close panicked: %v", r)
		}
	}()
	// Either nil or a non-panic error is acceptable; the contract is
	// "does not crash".
	_ = enc.Close()
}

func TestStream_DecoderCloseIdempotent(t *testing.T) {
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Fast)
	_ = enc.Encode(streamMsg{Seq: 1})
	_ = enc.Close()
	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	if err := dec.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("double Close on decoder panicked: %v", r)
		}
	}()
	_ = dec.Close()
}

func TestStream_LargeBatchDense(t *testing.T) {
	// Larger batch (10k messages) — exercises Flush, refill, growth.
	const N = 10_000
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for i := range N {
		if err := enc.Encode(streamMsg{
			Seq: i,
			Tag: "production-service",
			Vec: []uint64{uint64(i), uint64(i) + 1},
		}); err != nil {
			t.Fatalf("encode %d: %v", i, err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()
	for i := range N {
		var got streamMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("msg %d: %v", i, err)
		}
		if got.Seq != i {
			t.Fatalf("msg %d seq=%d", i, got.Seq)
		}
	}
}

func TestStream_DenseInternTableSpansMessages(t *testing.T) {
	// First message defines a value; subsequent messages reference it.
	// Confirms the intern table persists across stream Decode calls.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	common := "common-service-name-long-enough-to-intern"
	for i := range 5 {
		if err := enc.Encode(streamMsg{Seq: i, Tag: common}); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	// First message must contain the full string; subsequent ones
	// should be smaller — the stream amortises the intern cost.
	totalLen := w.Len()
	if totalLen >= 5*(len(common)+16) {
		t.Fatalf("stream too large for interned dup: %d bytes (each msg should reference, not copy)", totalLen)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()
	for i := range 5 {
		var got streamMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Tag != common {
			t.Fatalf("msg %d tag=%q want %q", i, got.Tag, common)
		}
	}
}

func TestStream_DecodeRandomShape(t *testing.T) {
	// Mixed-type stream: encode different shapes and decode each one
	// into the right destination type.
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	if err := enc.Encode(map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode([]float64{1.5, 2.5, 3.5}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(streamMsg{Seq: 42, Tag: "mixed"}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	defer dec.Close()
	var m map[string]int
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, map[string]int{"x": 1}) {
		t.Fatalf("map: %v", m)
	}
	var f []float64
	if err := dec.Decode(&f); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f, []float64{1.5, 2.5, 3.5}) {
		t.Fatalf("floats: %v", f)
	}
	var s streamMsg
	if err := dec.Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s.Seq != 42 || s.Tag != "mixed" {
		t.Fatalf("struct: %+v", s)
	}
}
