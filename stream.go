package qdf

import (
	"io"

	"github.com/alex60217101990/qdf/internal/bufpool"
)

// StreamEncoder writes a sequence of values into an io.Writer. The header
// is emitted once before the first value; in Dense mode the intern table
// survives across Encode calls so back-references span the whole stream.
type StreamEncoder struct {
	w   io.Writer
	enc *Encoder
	buf *[]byte
}

// NewStreamEncoder returns a stream encoder backed by w.
func NewStreamEncoder(w io.Writer, mode Mode) *StreamEncoder {
	buf := bufpool.Get(4096)
	enc := &Encoder{mode: mode, buf: (*buf)[:0]}
	if mode == Dense {
		enc.state = newEncState()
		enc.minIntern = 4
		enc.maxStateEntries = 1 << 16
	}
	return &StreamEncoder{w: w, enc: enc, buf: buf}
}

// Encode writes v as the next value in the stream. The encoder flushes
// internally when its buffer crosses 16 KiB; call Flush to push earlier.
func (s *StreamEncoder) Encode(v any) error {
	if err := encodeReflect(s.enc, v); err != nil {
		return err
	}
	if len(s.enc.buf) >= 1<<14 {
		return s.Flush()
	}
	return nil
}

// Flush writes any buffered bytes to the underlying writer.
func (s *StreamEncoder) Flush() error {
	if len(s.enc.buf) == 0 {
		return nil
	}
	if _, err := s.w.Write(s.enc.buf); err != nil {
		return err
	}
	s.enc.buf = s.enc.buf[:0]
	return nil
}

// Close flushes pending data and releases the scratch buffer to the
// pool. The underlying writer is not closed. Idempotent: subsequent
// calls are a safe no-op.
func (s *StreamEncoder) Close() error {
	if s.enc == nil {
		return nil
	}
	if err := s.Flush(); err != nil {
		return err
	}
	*s.buf = s.enc.buf
	bufpool.Put(s.buf)
	s.buf = nil
	s.enc = nil
	return nil
}

// StreamDecoder reads a sequence of values from an io.Reader. The intern
// table is preserved across Decode calls to match StreamEncoder.
//
// The decoder grows its window buffer as needed. State-table entries
// alias the buffer, so the window is never compacted during a stream;
// total memory tracks the stream length. For unbounded streams partition
// into envelopes and create a fresh StreamDecoder per envelope.
type StreamDecoder struct {
	r   io.Reader
	dec *Decoder
	buf *[]byte
}

// NewStreamDecoder returns a stream decoder reading from r.
func NewStreamDecoder(r io.Reader) *StreamDecoder {
	buf := bufpool.Get(4096)
	return &StreamDecoder{r: r, dec: &Decoder{buf: (*buf)[:0]}, buf: buf}
}

// SetNoCopy mirrors Decoder.SetNoCopy.
func (s *StreamDecoder) SetNoCopy(v bool) { s.dec.noCopy = v }

// Decode reads the next value into out. out must be a pointer.
func (s *StreamDecoder) Decode(out any) error {
	if err := s.refillIfNeeded(); err != nil {
		return err
	}
	return decodeReflect(s.dec, out)
}

func (s *StreamDecoder) refillIfNeeded() error {
	for {
		if len(s.dec.buf)-s.dec.i >= 4096 {
			return nil
		}
		if cap(*s.buf) < len(s.dec.buf)+4096 {
			grown := make([]byte, len(s.dec.buf), cap(*s.buf)*2+4096)
			copy(grown, s.dec.buf)
			*s.buf = grown
			s.dec.buf = grown
		}
		end := len(s.dec.buf)
		extra := cap(*s.buf) - end
		(*s.buf) = (*s.buf)[:end+extra]
		s.dec.buf = *s.buf
		n, err := s.r.Read(s.dec.buf[end:])
		s.dec.buf = s.dec.buf[:end+n]
		(*s.buf) = s.dec.buf
		if err == io.EOF {
			if len(s.dec.buf)-s.dec.i == 0 {
				return io.EOF
			}
			return nil
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
}

// Close releases the scratch buffer to the pool. The underlying reader
// is not closed. Idempotent: subsequent calls are a safe no-op.
func (s *StreamDecoder) Close() error {
	if s.dec == nil {
		return nil
	}
	*s.buf = s.dec.buf
	bufpool.Put(s.buf)
	s.buf = nil
	s.dec = nil
	return nil
}
