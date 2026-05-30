package qdf

import (
	"github.com/alex60217101990/qdf/internal/intern"
	"github.com/alex60217101990/qdf/internal/rans"
)

// Decoder reads QDF wire data from a single input buffer. Call SetInput
// to bind a buffer and the typed Read* methods to walk it. Behaviour is
// undefined if the input is mutated while the decoder holds it.
type Decoder struct {
	buf   []byte
	i     int
	mode  Mode
	state *decState

	// noCopy returns aliased string / []byte values instead of copies.
	// Faster, but the caller may not retain the result past the input's
	// lifetime.
	noCopy bool

	// keyCache dedupes map keys and other short repeated strings across
	// Unmarshal calls on the same pooled decoder.
	keyCache intern.Cache

	headerRead bool

	// colMaxLen bounds a slice codec's claimed element count while decoding
	// a columnar column, where every column must hold exactly the struct
	// count M. 0 means unbounded (standalone slice decode). It blocks a
	// hostile column whose constant/zero-width codec claims a huge n from a
	// tiny body before the per-element allocation.
	colMaxLen int

	// colIndex records whether the header set FlagColIndex. When true a
	// columnar (tagColStruct) payload carries a column-length index right
	// after the shape declaration; decodeColumnar / decodeColumnarAny consume
	// and validate it. Set fresh by readHeader on every decode.
	colIndex bool

	// selectFields, when non-nil, restricts the columnar map (any) decode to
	// the named columns: unrequested columns are skipped via the column-length
	// index when present, or simply not stored when it is absent. Set by
	// UnmarshalColumns for the duration of one decode; must be cleared on
	// reset / return-to-pool / SetInput so it never leaks across decodes.
	selectFields []string
}

// colLenOK reports whether a slice length is acceptable in the current
// decode context. Outside a columnar column (colMaxLen == 0) any length is
// allowed; the row-major and standalone paths bound length by the buffer.
func (d *Decoder) colLenOK(n uint64) bool {
	return d.colMaxLen == 0 || n <= uint64(d.colMaxLen)
}

// InternKey returns a string equal to b, sharing storage with prior
// identical keys when the cache has them.
//
//go:nosplit
func (d *Decoder) InternKey(b []byte) string { return d.keyCache.Make(b) }

// NewDecoder constructs a Decoder. Bind input via SetInput.
func NewDecoder() *Decoder { return &Decoder{} }

// NewDecoderOnBuf binds a Decoder to buf. Equivalent to NewDecoder followed
// by SetInput(buf), without the SetInput branch checks.
func NewDecoderOnBuf(buf []byte) *Decoder {
	return &Decoder{buf: buf}
}

// ReadStringBytes returns the next string or []byte value without copying.
// The returned slice aliases either the input buffer or the decoder's
// state-table storage. Callers that retain it beyond the input's lifetime
// must copy.
func (d *Decoder) ReadStringBytes() ([]byte, error) { return d.readStringBytes() }

// PeekTag returns the next tag byte without advancing the cursor.
func (d *Decoder) PeekTag() (byte, error) { return d.peekTag() }

// IsNil reports whether the next value is the nil tag, consuming it on
// true. Returns (false, nil) for any other tag.
func (d *Decoder) IsNil() (bool, error) {
	t, err := d.peekTag()
	if err != nil {
		return false, err
	}
	if t == tagNil {
		d.i++
		return true, nil
	}
	return false, nil
}

// TagNil exposes the nil-value tag for comparison with PeekTag.
const TagNil = tagNil

// RemainingBytes returns the unread portion of the input buffer. The
// result aliases the buffer and is invalidated by the next SetInput.
func (d *Decoder) RemainingBytes() []byte { return d.buf[d.i:] }

// Advance moves the read cursor forward by n bytes.
func (d *Decoder) Advance(n int) { d.i += n }

// MarkHeaderRead tells the decoder the magic+version header is already
// consumed (e.g. the buffer is a tail slice from a parent decoder). The
// next read will skip the header check.
func (d *Decoder) MarkHeaderRead() { d.headerRead = true }

// SetInput rebinds the decoder to buf, dropping any prior state table.
func (d *Decoder) SetInput(buf []byte) {
	d.buf = buf
	d.i = 0
	d.headerRead = false
	d.mode = Fast
	d.colIndex = false
	d.selectFields = nil
	if d.state != nil {
		d.state.reset()
	}
}

// SetNoCopy switches the decoder into aliasing mode: string and []byte
// reads return slices that share storage with the input buffer. Faster,
// but the caller must not retain the result past the lifetime of the
// input.
func (d *Decoder) SetNoCopy(v bool) { d.noCopy = v }

// Pos returns the current read offset.
func (d *Decoder) Pos() int { return d.i }

// Remaining returns the number of unread bytes.
func (d *Decoder) Remaining() int { return len(d.buf) - d.i }

// CheckLength returns ErrShortBuffer when n claimed elements cannot fit
// in the remaining input. Use before allocating per-element storage.
//
//go:nosplit
func (d *Decoder) CheckLength(n int, perElem int) error {
	if n < 0 {
		return ErrInvalidLength
	}
	if uint64(n)*uint64(perElem) > uint64(len(d.buf)-d.i) {
		return ErrShortBuffer
	}
	return nil
}

// readHeader parses the 5-byte header. Called once per buffer.
func (d *Decoder) readHeader() error {
	if d.headerRead {
		return nil
	}
	if len(d.buf)-d.i < 5 {
		return ErrShortBuffer
	}
	if d.buf[d.i] != Magic0 || d.buf[d.i+1] != Magic1 || d.buf[d.i+2] != Magic2 {
		return ErrBadMagic
	}
	if d.buf[d.i+3] != Version1 {
		return ErrBadVersion
	}
	flags := d.buf[d.i+4]
	d.colIndex = flags&FlagColIndex != 0
	if flags&FlagDense != 0 {
		d.mode = Dense
		if d.state == nil {
			d.state = newDecState()
		}
	}
	d.i += 5
	d.headerRead = true
	if flags&FlagRANS != 0 {
		rest := d.buf[d.i:]
		origLen, k := readUvarint(rest)
		if k <= 0 {
			return ErrInvalidLength
		}
		// Bound the allocation: reject a hostile origLen that dwarfs the
		// input. rANS shrinks at best modestly, so a real origLen stays
		// within a small factor of the compressed size.
		if origLen > uint64(len(d.buf))*64+(1<<20) {
			return ErrInvalidLength
		}
		body, err := rans.Decode(rest[k:], int(origLen))
		if err != nil {
			return err
		}
		// The reconstructed body is a plain (post-decompression) tag stream;
		// read it from offset 0, exactly as a non-rANS buffer's body.
		d.buf = body
		d.i = 0
	}
	return nil
}

// peekTag returns the next tag without consuming it.
func (d *Decoder) peekTag() (byte, error) {
	if err := d.readHeader(); err != nil {
		return 0, err
	}
	if d.i >= len(d.buf) {
		return 0, ErrShortBuffer
	}
	return d.buf[d.i], nil
}

// next returns the next tag and advances past it.
func (d *Decoder) next() (byte, error) {
	t, err := d.peekTag()
	if err != nil {
		return 0, err
	}
	d.i++
	return t, nil
}
