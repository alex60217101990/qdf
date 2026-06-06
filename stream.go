package qdf

import (
	"errors"
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
	// broken is set when a mid-message encode fails. The body bytes are rolled
	// back, but the encoder's cross-message state (intern table, declared
	// struct/map shapes, LRU, predictors) may have advanced past what the
	// decoder saw — so every later frame's back-refs would desync. Once broken,
	// further Encode is refused; the already-buffered valid prefix can still be
	// Flushed.
	broken bool
}

// ErrStreamBroken is returned by StreamEncoder.Encode after a previous Encode
// failed mid-message: the cross-message encoder state can no longer be trusted,
// so the stream must be abandoned (the valid prefix may still be flushed).
var ErrStreamBroken = errors.New("qdf: stream encoder broken by a prior mid-message error")

// ErrStreamDecoderBroken is returned by StreamDecoder.Decode after a previous
// Decode failed mid-message: the read cursor and the shared dense state can no
// longer be trusted (subsequent frames would misparse), so the stream must be
// abandoned.
var ErrStreamDecoderBroken = errors.New("qdf: stream decoder broken by a prior mid-message error")

// NewStreamEncoder returns a stream encoder backed by w. Dense mode
// activates the full balanced codec set (OptBalanced); Fast mode
// emits raw tagged bytes (OptSpeed). For finer per-stream tuning,
// build the StreamEncoder via NewStreamEncoderWith.
func NewStreamEncoder(w io.Writer, mode Mode) *StreamEncoder {
	opts := OptSpeed
	if mode == Dense {
		opts = OptBalanced
	}
	return NewStreamEncoderWith(w, opts)
}

// NewStreamEncoderWith returns a stream encoder with the given
// Options bit-mask. The intern table (when OptDense is set) survives
// across Encode calls so back-references span the whole stream.
func NewStreamEncoderWith(w io.Writer, opts Options) *StreamEncoder {
	buf := bufpool.Get(4096)
	enc := &Encoder{buf: (*buf)[:0], minIntern: 4, maxStateEntries: maxInternEntries, maxDepth: DefaultMaxDepth}
	enc.applyOpts(opts)
	// The column index is a single-message feature: it backpatches the header
	// flag at a fixed offset, which a stream's shared/reused buffer invalidates
	// after the first Flush. Like the whole-body rANS pass, it is not emitted in
	// streaming mode.
	enc.colIndex = false
	if opts.Has(OptDense) {
		enc.state = newEncState()
	}
	return &StreamEncoder{w: w, enc: enc, buf: buf}
}

// Encode writes v as the next value in the stream. Each message is framed with
// a uvarint byte-length prefix so the decoder can buffer a whole message — of
// any size — before decoding it. The 5-byte stream header is written once,
// before the first frame, and is not itself framed. The encoder flushes
// internally when its buffer crosses 16 KiB; call Flush to push earlier.
func (s *StreamEncoder) Encode(v any) error {
	if s.enc == nil {
		return io.ErrClosedPipe
	}
	if s.broken {
		return ErrStreamBroken
	}
	// One-per-stream header preamble, outside the frames.
	if !s.enc.headerOut {
		s.enc.writeHeader()
	}
	// Reserve one byte for the length prefix, then encode the body after it.
	// The common case (a message under 128 bytes) needs exactly that one byte,
	// so it is written in place with no memmove; only larger messages shift.
	start := len(s.enc.buf)
	s.enc.buf = append(s.enc.buf, 0)
	bodyStart := len(s.enc.buf)
	if err := encodeReflect(s.enc, v); err != nil {
		s.enc.buf = s.enc.buf[:start] // drop the reservation + any partial body
		// The buffer is rolled back, but encoder state (intern IDs, shape
		// declarations, predictors) may have advanced — desyncing every later
		// frame. Poison the stream so no further frame is emitted against it.
		s.broken = true
		return err
	}
	n := len(s.enc.buf) - bodyStart
	if n < 0x80 {
		s.enc.buf[start] = byte(n) // single-byte uvarint, body already in place
	} else {
		// Need an m-byte prefix; one byte is reserved, so make room for m-1 more
		// and shift the body right by m-1 (overlap-safe copy).
		var lb [10]byte
		pre := appendUvarint(lb[:0], uint64(n))
		m := len(pre)
		s.enc.buf = append(s.enc.buf, make([]byte, m-1)...)
		copy(s.enc.buf[start+m:], s.enc.buf[bodyStart:bodyStart+n])
		copy(s.enc.buf[start:start+m], pre)
	}
	if len(s.enc.buf) >= 1<<14 {
		return s.Flush()
	}
	return nil
}

// Flush writes any buffered bytes to the underlying writer.
func (s *StreamEncoder) Flush() error {
	if s.enc == nil {
		return io.ErrClosedPipe
	}
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
	// broken mirrors StreamEncoder.broken: a mid-frame decode error (or a frame
	// whose body the value did not consume exactly) leaves the read cursor inside
	// the failed frame and the shared dense state (intern table, LRU, shapes)
	// partially advanced — every later frame would misparse. Once broken, further
	// Decode is refused rather than returning silently wrong values.
	broken bool
}

// NewStreamDecoder returns a stream decoder reading from r.
func NewStreamDecoder(r io.Reader) *StreamDecoder {
	buf := bufpool.Get(4096)
	return &StreamDecoder{r: r, dec: &Decoder{buf: (*buf)[:0]}, buf: buf}
}

// SetNoCopy makes Decode return string / []byte values that alias the
// decoder's internal window buffer instead of copying them — eliminating the
// per-value copy that dominates decode allocation.
//
// Unlike the one-shot Decoder.SetNoCopy (which aliases the caller's input
// buffer and is unsafe the moment that buffer is reused — the typical server
// recycles its read buffer right after the handler returns), the stream owns
// its window buffer, so aliasing is safe for the lifetime of the stream:
//
//   - A decoded value stays valid until Close. Close returns the window to a
//     pool, after which any retained aliased value is undefined — copy
//     (strings.Clone / append) anything you need past Close.
//   - The window grows but is never compacted mid-stream (the Dense intern
//     table already aliases it for cross-message back-references), so a growth
//     does not invalidate earlier values, and noCopy adds no extra memory: the
//     bytes are retained either way.
//
// Because the window is retained for the whole stream regardless, this is the
// safe home for zero-copy decode — process or copy each value before Close and
// you pay zero per-value allocation across every message. No-op after Close.
func (s *StreamDecoder) SetNoCopy(v bool) {
	if s.dec != nil {
		s.dec.noCopy = v
	}
}

// maxStreamMsg bounds a single framed message so a hostile length prefix can't
// drive an unbounded buffer read. 2 GiB is far above any realistic message.
const maxStreamMsg = 1 << 31

// Decode reads the next value into out. out must be a pointer. Returns io.EOF
// at a clean end of stream. Each message is length-framed, so a message of any
// size is buffered in full before decoding — no 4 KiB window limit.
func (s *StreamDecoder) Decode(out any) error {
	if s.dec == nil {
		return io.ErrClosedPipe
	}
	if s.broken {
		return ErrStreamDecoderBroken
	}
	// Consume the one-per-stream header before the first frame.
	if !s.dec.headerRead {
		if err := s.fill(5, true); err != nil {
			return err // io.EOF here means an empty stream
		}
		if err := s.dec.readHeader(); err != nil {
			return err
		}
	}
	// Read the frame length (io.EOF at a clean frame boundary = end of stream).
	framelen, err := s.readFrameLen()
	if err != nil {
		return err
	}
	// Buffer the whole message body, then decode it in a single pass so a
	// partial decode never mutates the shared dense state. Past the length
	// prefix a short read is a truncated frame, never a clean end.
	if err := s.fill(framelen, false); err != nil {
		return err // ErrShortBuffer on a truncated final frame
	}
	start := s.dec.i
	if err := decodeReflect(s.dec, out); err != nil {
		// The cursor is left partway through the frame and the dense state is
		// half-advanced; poison the stream so the next Decode fails cleanly
		// instead of misparsing the rest of this frame as new frames.
		s.broken = true
		return err
	}
	// The value must consume exactly the framed length; a mismatch means a
	// corrupt or hostile frame, so reject it instead of desyncing the stream.
	if s.dec.i-start != framelen {
		s.broken = true
		return ErrInvalidLength
	}
	return nil
}

// readFrameLen parses the uvarint length prefix of the next message, advancing
// the cursor past it. Returns io.EOF when no more frames remain.
func (s *StreamDecoder) readFrameLen() (int, error) {
	for {
		if v, k := readUvarint(s.dec.buf[s.dec.i:]); k > 0 {
			if v > maxStreamMsg {
				return 0, ErrInvalidLength
			}
			s.dec.i += k
			return int(v), nil
		}
		had := len(s.dec.buf) - s.dec.i
		if had >= 10 { // a uvarint is at most 10 bytes; longer is malformed
			return 0, ErrInvalidLength
		}
		if err := s.fill(had+1, true); err != nil {
			return 0, err // io.EOF (boundary) = clean end; ErrShortBuffer = partial length
		}
	}
}

// fill reads from the underlying reader until at least need unread bytes are
// buffered. Returns io.EOF if the stream ends with nothing unread, or
// ErrShortBuffer if it ends mid-message (fewer than need bytes available).
// fill reads from the underlying reader until at least need unread bytes are
// buffered. When boundary is true (reading at a message boundary — the header
// or a frame length) an immediate clean end of stream returns io.EOF; in every
// other case a stream that ends before need bytes returns ErrShortBuffer.
func (s *StreamDecoder) fill(need int, boundary bool) error {
	for len(s.dec.buf)-s.dec.i < need {
		if cap(*s.buf)-len(s.dec.buf) == 0 {
			grown := make([]byte, len(s.dec.buf), cap(*s.buf)*2+4096)
			copy(grown, s.dec.buf)
			*s.buf = grown
			s.dec.buf = grown
		}
		end := len(s.dec.buf)
		(*s.buf) = (*s.buf)[:cap(*s.buf)]
		s.dec.buf = *s.buf
		n, rerr := s.r.Read(s.dec.buf[end:])
		s.dec.buf = s.dec.buf[:end+n]
		(*s.buf) = s.dec.buf
		if n == 0 {
			if rerr == nil {
				continue // no progress but no error: retry
			}
			if errors.Is(rerr, io.EOF) {
				if boundary && len(s.dec.buf)-s.dec.i == 0 {
					return io.EOF // clean end of stream at a message boundary
				}
				return ErrShortBuffer // truncated mid-message
			}
			return rerr
		}
		if rerr != nil && !errors.Is(rerr, io.EOF) {
			return rerr
		}
	}
	return nil
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
