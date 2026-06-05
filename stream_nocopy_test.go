package qdf

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// TestStream_NoCopyValidUntilClose pins the documented zero-copy stream
// contract: with SetNoCopy, a value decoded from an early message stays valid
// while later messages are decoded — even across a window growth — because the
// stream owns its buffer and never compacts it mid-stream. (A retained value is
// only undefined after Close.)
func TestStream_NoCopyValidUntilClose(t *testing.T) {
	type Msg struct {
		Tag  string `qdf:"tag"`
		Blob string `qdf:"blob"`
		N    int    `qdf:"n"`
	}
	// Distinct, sizeable payloads so the window must grow as we decode on.
	const count = 64
	mk := func(i int) Msg {
		return Msg{
			Tag:  "tag-" + strings.Repeat("x", 8) + "-" + strconv.Itoa(i),
			Blob: strings.Repeat("blob-payload-", 40) + strconv.Itoa(i),
			N:    i,
		}
	}
	var w bytes.Buffer
	enc := NewStreamEncoder(&w, Dense)
	for i := 0; i < count; i++ {
		if err := enc.Encode(mk(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	dec := NewStreamDecoder(bytes.NewReader(w.Bytes()))
	dec.SetNoCopy(true)

	// Decode message 0 and RETAIN its (aliased) strings.
	var first Msg
	if err := dec.Decode(&first); err != nil {
		t.Fatal(err)
	}
	keptTag, keptBlob := first.Tag, first.Blob
	want0 := mk(0)
	if keptTag != want0.Tag || keptBlob != want0.Blob {
		t.Fatalf("message 0 mismatch: tag=%q blob=%q", keptTag, keptBlob)
	}

	// Decode the rest (forces the window to grow well past message 0's bytes).
	for i := 1; i < count; i++ {
		var m Msg
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
		want := mk(i)
		if m.Tag != want.Tag || m.Blob != want.Blob || m.N != want.N {
			t.Fatalf("message %d mismatch: %+v", i, m)
		}
		// The retained message-0 aliases must NOT have been corrupted by the
		// growth / later decodes.
		if keptTag != want0.Tag || keptBlob != want0.Blob {
			t.Fatalf("retained message-0 value corrupted after decoding message %d: tag=%q blob=%q", i, keptTag, keptBlob)
		}
	}
	_ = dec.Close()
}
