package qdf

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type bigFrameMsg struct {
	ID  int64  `qdf:"id"`
	Pay []byte `qdf:"pay"`
	Tag string `qdf:"tag"`
}

// TestStream_LargeMessageFraming pins the framing fix: a single message larger
// than the decoder's read window must round-trip. Before length-framing, any
// message bigger than the ~4 KiB refill window failed with ErrShortBuffer.
func TestStream_LargeMessageFraming(t *testing.T) {
	for _, mode := range []Mode{Fast, Dense} {
		for _, sz := range []int{0, 1, 100, 4095, 4096, 4097, 8192, 50000, 300000} {
			var w bytes.Buffer
			se := NewStreamEncoder(&w, mode)
			in := &bigFrameMsg{ID: int64(sz), Pay: bytes.Repeat([]byte{0xAB}, sz), Tag: "msg"}
			if err := se.Encode(in); err != nil {
				t.Fatalf("mode=%v sz=%d encode: %v", mode, sz, err)
			}
			if err := se.Close(); err != nil {
				t.Fatalf("mode=%v sz=%d close: %v", mode, sz, err)
			}
			sd := NewStreamDecoder(bytes.NewReader(w.Bytes()))
			var got bigFrameMsg
			if err := sd.Decode(&got); err != nil {
				t.Fatalf("mode=%v sz=%d decode: %v", mode, sz, err)
			}
			if got.ID != in.ID || got.Tag != in.Tag || len(got.Pay) != sz || !bytes.Equal(got.Pay, in.Pay) {
				t.Fatalf("mode=%v sz=%d: round-trip mismatch", mode, sz)
			}
			// Next Decode must report a clean end of stream.
			if err := sd.Decode(&got); !errors.Is(err, io.EOF) {
				t.Fatalf("mode=%v sz=%d: expected io.EOF after last message, got %v", mode, sz, err)
			}
			sd.Close()
		}
	}
}

// TestStream_ManyMixedMessages round-trips a sequence of small and large
// messages in one stream, checking frame boundaries hold and dense state
// (shared across frames) stays consistent.
func TestStream_ManyMixedMessages(t *testing.T) {
	sizes := []int{10, 9000, 3, 70000, 0, 256, 5000, 1}
	for _, mode := range []Mode{Fast, Dense} {
		var w bytes.Buffer
		se := NewStreamEncoder(&w, mode)
		for i, sz := range sizes {
			m := &bigFrameMsg{ID: int64(i), Pay: bytes.Repeat([]byte{byte(i)}, sz), Tag: "svc"}
			if err := se.Encode(m); err != nil {
				t.Fatalf("mode=%v msg%d encode: %v", mode, i, err)
			}
		}
		if err := se.Close(); err != nil {
			t.Fatal(err)
		}
		sd := NewStreamDecoder(bytes.NewReader(w.Bytes()))
		for i, sz := range sizes {
			var got bigFrameMsg
			if err := sd.Decode(&got); err != nil {
				t.Fatalf("mode=%v msg%d decode: %v", mode, i, err)
			}
			if got.ID != int64(i) || len(got.Pay) != sz || got.Tag != "svc" {
				t.Fatalf("mode=%v msg%d: mismatch (id=%d len=%d tag=%q)", mode, i, got.ID, len(got.Pay), got.Tag)
			}
		}
		var sink bigFrameMsg
		if err := sd.Decode(&sink); !errors.Is(err, io.EOF) {
			t.Fatalf("mode=%v: expected io.EOF at end, got %v", mode, err)
		}
		sd.Close()
	}
}

// TestStream_ChunkedReaderLargeMessage feeds a large message through a reader
// that returns one byte per Read, exercising the incremental fill across frame
// length and body.
func TestStream_ChunkedReaderLargeMessage(t *testing.T) {
	var w bytes.Buffer
	se := NewStreamEncoder(&w, Fast)
	in := &bigFrameMsg{ID: 7, Pay: bytes.Repeat([]byte("z"), 20000), Tag: "chunk"}
	if err := se.Encode(in); err != nil {
		t.Fatal(err)
	}
	if err := se.Close(); err != nil {
		t.Fatal(err)
	}
	sd := NewStreamDecoder(iotest1Byte(w.Bytes()))
	var got bigFrameMsg
	if err := sd.Decode(&got); err != nil {
		t.Fatalf("chunked decode: %v", err)
	}
	if len(got.Pay) != 20000 || got.ID != 7 || got.Tag != "chunk" {
		t.Fatal("chunked: round-trip mismatch")
	}
	sd.Close()
}

// iotest1Byte returns a reader that yields the data one byte per Read.
func iotest1Byte(b []byte) io.Reader { return &bfOneByteReader{b: b} }

type bfOneByteReader struct {
	b []byte
	i int
}

func (r *bfOneByteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = r.b[r.i]
	r.i++
	return 1, nil
}

// TestStream_DecodeAfterCloseNoPanic: Decode/SetNoCopy after Close must error,
// not panic.
func TestStream_DecodeAfterCloseNoPanic(t *testing.T) {
	var w bytes.Buffer
	se := NewStreamEncoder(&w, Fast)
	_ = se.Encode(&bigFrameMsg{ID: 1, Tag: "x"})
	_ = se.Close()
	// Encode after Close → error, not panic.
	if err := se.Encode(&bigFrameMsg{ID: 2}); err == nil {
		t.Fatal("Encode after Close: expected error")
	}

	sd := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	var got bigFrameMsg
	_ = sd.Decode(&got)
	sd.Close()
	sd.SetNoCopy(true) // must not panic
	if err := sd.Decode(&got); err == nil {
		t.Fatal("Decode after Close: expected error")
	}
}
