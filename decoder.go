package qdf

import (
	"reflect"
	"unsafe"

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

	// depth / maxDepth bound recursive decode nesting. The wire dictates
	// nesting for `any`, recursive pointer/slice/map types, so without a
	// guard a crafted deeply-nested payload overflows the goroutine stack —
	// an UNRECOVERABLE fatal error, i.e. a remote DoS. descend/ascend
	// (defer-balanced) cap it at maxDepth (lazily DefaultMaxDepth), returning
	// ErrCycleDetected, symmetric to the encoder's pointer-cycle guard.
	depth    int
	maxDepth int

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

	// query, when non-nil, makes a columnar decode filter rows by the plan's
	// predicates (AND) and project the plan's columns. Set by Unmarshal when
	// QueryOptions are passed; cleared on reset / SetInput so it never leaks.
	query *queryPlan

	// mapFreeList holds maps harvested from a reused []struct{map} (or []map)
	// decode target whose per-element maps decode-slice-reuse is about to zero.
	// Keyed by the map's reflect.Type; reuseOrMakeMap pops a recycled map
	// instead of allocating a fresh one. Lazily initialised; cleared (entries
	// dropped, backing kept) on SetInput so recycled maps never cross into a
	// different decode target.
	mapFreeList map[reflect.Type][]unsafe.Pointer

	// deltaScratch is a reused unpack buffer for the Delta+FOR readers: the
	// bit-unpacked deltas are a transient intermediate (the prefix sum writes
	// the retained out slice), so a per-call make is pure garbage. Grows to the
	// largest column seen; bounded on return to pool.
	deltaScratch []uint64
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

// descend enters one level of recursive decode, bounding nesting depth so a
// hostile deeply-nested payload cannot overflow the goroutine stack (an
// unrecoverable fatal error). Pair with a deferred ascend. maxDepth is lazily
// initialised so every Decoder construction path is covered.
func (d *Decoder) descend() error {
	if d.maxDepth == 0 {
		d.maxDepth = DefaultMaxDepth
	}
	d.depth++
	if d.depth > d.maxDepth {
		return ErrCycleDetected
	}
	return nil
}

func (d *Decoder) ascend() { d.depth-- }

// SetInput rebinds the decoder to buf, dropping any prior state table.
func (d *Decoder) SetInput(buf []byte) {
	d.buf = buf
	d.i = 0
	d.depth = 0
	d.headerRead = false
	d.mode = Fast
	d.colIndex = false
	d.selectFields = nil
	d.query = nil
	clear(d.mapFreeList) // drop recycled maps; keep the backing allocated
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

// readHeader consumes the 5-byte header once per buffer. The hot path is the
// already-read check; the actual parse is outlined into readHeaderSlow so this
// (and its callers, peekTag/next) stay within the inliner budget.
func (d *Decoder) readHeader() error {
	if d.headerRead {
		return nil
	}
	return d.readHeaderSlow()
}

// readHeaderSlow parses the 5-byte header (and an optional rANS body). Kept out
// of line: it runs once per decode, so inlining its cost into the per-tag hot
// path would only bloat callers.
//
//go:noinline
func (d *Decoder) readHeaderSlow() error {
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

// peekTag returns the next tag without consuming it. The header check is
// inlined here so the common (header-already-read) path has no call; the parse
// is taken only on the first tag of a decode.
func (d *Decoder) peekTag() (byte, error) {
	if !d.headerRead {
		if err := d.readHeaderSlow(); err != nil {
			return 0, err
		}
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
