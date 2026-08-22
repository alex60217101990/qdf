package qdf

import (
	"math"
	"reflect"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/intern"
	"github.com/alex60217101990/qdf/internal/rans"
	"github.com/alex60217101990/qdf/internal/tans"
)

// Decoder reads QDF wire data from a single input buffer. Call SetInput
// to bind a buffer and the typed Read* methods to walk it. Behavior is
// undefined if the input is mutated while the decoder holds it.
// Field order groups the pointer-bearing and 8-byte fields first, then the
// int counters, then the 1-byte flags last so the interspersed bools do not
// each force their own padding word (176 bytes vs 208 for the source order).
type Decoder struct {
	state *decState

	// arena, when non-nil, receives copied inline string bodies (bump-packed)
	// instead of one heap allocation per string. Caller-owned (see Arena); the
	// decoder only borrows the pointer for one decode and clears it on return
	// to the pool so a pooled decoder never pins a caller's arena. Ignored when
	// noCopy is set (aliasing already avoids the copy).
	arena *Arena

	// query, when non-nil, makes a columnar decode filter rows by the plan's
	// predicates (AND) and project the plan's columns. Set by Unmarshal when
	// QueryOptions are passed; cleared on reset / SetInput so it never leaks.
	query *queryPlan

	// mapFreeList holds maps harvested from a reused []struct{map} (or []map)
	// decode target whose per-element maps decode-slice-reuse is about to zero.
	// Keyed by the map's reflect.Type; reuseOrMakeMap pops a recycled map
	// instead of allocating a fresh one. Lazily initialized; cleared (entries
	// dropped, backing kept) on SetInput so recycled maps never cross into a
	// different decode target.
	mapFreeList map[reflect.Type][]unsafe.Pointer

	// keyIdx is a reused base-key→index map for keyed slice apply. It is cleared
	// (entries dropped, backing kept) and rebuilt per keyed slice; the spike-sized
	// backing is dropped on the apply reset path so a one-off huge slice never
	// pins a large map across pooled reuse.
	keyIdx map[string]int

	// strDeltaBase points at the current struct field's previous value while a
	// value is being decoded, or is nil outside that. tagStrDelta codes against
	// it. Set by decodeStruct per wire field — including fields the target
	// struct does not declare, whose base still has to advance.
	strField *decFieldState

	buf []byte

	// keyCache dedupes map keys and other short repeated strings across
	// Unmarshal calls on the same pooled decoder.
	keyCache intern.Cache

	// selectFields, when non-nil, restricts the columnar map (any) decode to
	// the named columns: unrequested columns are skipped via the column-length
	// index when present, or simply not stored when it is absent. Set by
	// UnmarshalColumns for the duration of one decode; must be cleared on
	// reset / return-to-pool / SetInput so it never leaks across decodes.
	selectFields []string

	// selectKeys is the UnmarshalKeys projection: the keys to keep from the
	// ROOT map. It is deliberately separate from selectFields (column names)
	// — sharing one field made a column projection filter map entries and
	// vice versa — and is consumed one-shot by the root map's decode loop, so
	// nothing nested (values, Skip, nested maps, columnar containers) ever
	// sees a live filter.
	selectKeys []string

	// deltaScratch is a reused unpack buffer for the Delta+FOR readers: the
	// bit-unpacked deltas are a transient intermediate (the prefix sum writes
	// the retained out slice), so a per-call make is pure garbage. Grows to the
	// largest column seen; bounded on return to pool.
	deltaScratch []uint64

	i int

	// depth / maxDepth bound recursive decode nesting. The wire dictates
	// nesting for `any`, recursive pointer/slice/map types, so without a
	// guard a crafted deeply-nested payload overflows the goroutine stack —
	// an UNRECOVERABLE fatal error, i.e. a remote DoS. descend/ascend
	// (defer-balanced) cap it at maxDepth (lazily DefaultMaxDepth), returning
	// ErrCycleDetected, symmetric to the encoder's pointer-cycle guard.
	depth    int
	maxDepth int

	// colMaxLen bounds a slice codec's claimed element count while decoding
	// a columnar column, where every column must hold exactly the struct
	// count M. 0 means unbounded (standalone slice decode). It blocks a
	// hostile column whose constant/zero-width codec claims a huge n from a
	// tiny body before the per-element allocation.
	colMaxLen int

	// lastWireShapeID is the shape ID of the most recent struct header read
	// through decodeMapStringShapeHeader. The batch decoder reaches struct
	// headers via ReadStructHeader, whose public signature does not carry it.
	lastWireShapeID uint32

	mode Mode

	// noCopy returns aliased string / []byte values instead of copies.
	// Faster, but the caller may not retain the result past the input's
	// lifetime.
	noCopy bool

	// colStrNoPool forces colStrScratch to return a fresh owned []string instead
	// of the reused decode-state scratch. Set only around the nullable string
	// column read, which stores &strs[k] pointers into *string fields — those
	// must reference a stable backing array, not the buffer the next column reuses.
	colStrNoPool bool

	headerRead bool

	// colIndex records whether the header set FlagColIndex. When true a
	// columnar (tagColStruct) payload carries a column-length index right
	// after the shape declaration; decodeColumnar / decodeColumnarAny consume
	// and validate it. Set fresh by readHeader on every decode.
	colIndex bool

	// keyIdxBusy marks keyIdx as borrowed by an in-progress keyed-slice apply so a
	// nested keyed slice routes to a fresh local map instead of clobbering it.
	keyIdxBusy bool

	// lastReadOwned marks that the bytes readStringBytes just returned are
	// decoder-owned (a rebuilt tagStrDelta value) rather than an alias of the
	// input buffer, so ReadString can hand them out without a second copy.
	// Cleared at the top of every raw string read so it cannot leak forward.
	lastReadOwned bool
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

// ReadStructHeader reads a struct/map header for code-generated DecodeQDF,
// transparently handling both forms a struct takes on the wire:
//   - a shape-interned header (tagMapShape, see Encoder.StructShape) → returns the
//     field names in encoded order with shaped=true; the caller reads len(names)
//     values in that order.
//   - a plain map header (tagMap8/16/32) → returns the entry count as plainN with
//     shaped=false; the caller reads plainN (name, value) pairs inline.
//
// This lets a generated decoder consume both qdfgen shape output and a plain qdf
// map (e.g. from a non-shaped encoder, an older generated type, or the reflect
// path under OptSpeed) without a wire-format negotiation. Exported for
// cmd/qdfgen-generated code.
// ShapeID reports the wire shape ID of the struct header ReadStructHeader just
// read, or 0 if that header carried no shape.
//
// Generated decoders capture it in a local BEFORE decoding any field: a nested
// struct reads its own header and overwrites the decoder's record, so a parent
// that re-read it between fields would bind its remaining fields to the child's
// shape.
func (d *Decoder) ShapeID() uint32 { return d.lastWireShapeID }

// EnterField binds wire field i of the given shape as the context for the next
// value read, so a tagStrDelta value can rebuild against that field's previous
// value. Pair every call with LeaveField.
//
// A generated decoder must call this around EVERY field, not only string ones:
// the encoder advances a field's base on every value it writes for that field,
// and a decoder that binds only some of them leaves the base a row behind. The
// failure is silent — the types still line up.
func (d *Decoder) EnterField(shapeID uint32, nFields, i int) {
	if d.state == nil || shapeID == 0 || i < 0 || i >= nFields {
		return
	}
	bases := d.state.strFieldStates(shapeID, nFields)
	d.strField = &bases[i]
}

// LeaveField unbinds the field context set by EnterField.
func (d *Decoder) LeaveField() { d.strField = nil }

func (d *Decoder) ReadStructHeader() (names []string, plainN int, shaped bool, err error) {
	tag, err := d.peekTag()
	if err != nil {
		return nil, 0, false, err
	}
	if tag == tagMapShape {
		names, err = decodeMapStringShapeHeader(d)
		return names, 0, true, err
	}
	// A plain map header carries no shape, so ShapeID must not keep reporting the
	// last SHAPED one. EnterField already treats id 0 as "nothing to bind"; this
	// is what lets that guard fire. In-repo generated decoders read ShapeID only
	// inside the shaped branch, so nothing today can observe the stale value —
	// but it is a public API, and reporting another struct's shape id would bind
	// its per-field delta bases and rebuild strings against the wrong previous
	// value, silently.
	d.lastWireShapeID = 0
	n, err := d.ReadMapHeader()
	return nil, n, false, err
}

// DecodeValue decodes the next value as a dynamic any — the schemaless form
// (string, bool, int64/uint64/float64, []any, map[string]any, …) that decoding
// into an interface{} produces, mirroring encoding/json. It is the decode
// counterpart of Encoder.EncodeValue and is what qdfgen-generated code calls for
// an interface{} (any) struct field, so a code-generated type can still carry
// fully dynamic data on that field.
func (d *Decoder) DecodeValue(out *any) error {
	v, err := decodeAny(d)
	if err != nil {
		return err
	}
	*out = v
	return nil
}

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
// initialized so every Decoder construction path is covered.
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
	d.colMaxLen = 0
	d.selectFields = nil
	d.selectKeys = nil
	d.query = nil
	clear(d.mapFreeList) // drop recycled maps; keep the backing allocated
	if d.state != nil {
		d.state.reset()
	}
}

// SetNoCopy switches the decoder into aliasing mode: string and []byte
// reads return slices that share storage with the input buffer. Faster
// (~2x on string-heavy payloads) with near-zero allocations.
//
// The aliases are valid only while the input is alive and unmodified. This is
// safe — and free — when the input is caller-owned and not reused: an mmap, a
// file fully read into memory, or a buffer allocated fresh per message and
// never pooled (the aliasing headers keep it alive via the GC, so you need not
// track its lifetime). The hazard is buffer REUSE or mutation (a sync.Pool
// buffer, an overwritten scratch slice), which silently corrupts already-decoded
// values — a manual use-after-free the race detector will not catch. See
// WithNoCopy for the full contract.
func (d *Decoder) SetNoCopy(v bool) { d.noCopy = v }

// SetArena directs the decoder to pack copied inline string bodies into a,
// bump-packed, instead of allocating one string per field. Pass nil to disable.
// The decoded strings alias a's memory; see Arena for the lifetime contract.
// Ignored while noCopy is set. Not concurrent-safe: one Arena per goroutine.
func (d *Decoder) SetArena(a *Arena) { d.arena = a }

// Pos returns the current read offset.
func (d *Decoder) Pos() int { return d.i }

// Remaining returns the number of unread bytes.
func (d *Decoder) Remaining() int { return len(d.buf) - d.i }

// CheckLength returns ErrShortBuffer when n claimed elements cannot fit
// in the remaining input. Use before allocating per-element storage.
//
//go:nosplit
func (d *Decoder) CheckLength(n, perElem int) error {
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
		// within a small factor of the compressed size. The second clause
		// rejects anything past the int range so the int(origLen) narrowing
		// below cannot truncate a multi-GiB length on a 32-bit build and decode
		// a silently short/wrong body.
		if origLen > uint64(len(d.buf))*64+(1<<20) || origLen > uint64(math.MaxInt) {
			return ErrInvalidLength
		}
		blob := rest[k:]
		var body []byte
		var err error
		if len(blob) > 0 && tans.IsTag(blob[0]) {
			body, err = tans.Decode(blob, int(origLen))
		} else {
			body, err = rans.Decode(blob, int(origLen))
		}
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
