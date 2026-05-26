package qdf

import (
	"encoding/binary"
	"math"
	"slices"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Encoder writes a single QDF value into a growing internal buffer. Reset
// drops the buffer contents and (in Dense mode) the intern table so the
// encoder can be reused.
type Encoder struct {
	buf       []byte
	mode      Mode
	state     *encState
	headerOut bool

	// minIntern is the minimum string length eligible for interning;
	// shorter values go in line.
	minIntern int

	// maxStateEntries caps the intern table. Past the cap, new strings go
	// in line; existing IDs still resolve.
	maxStateEntries int

	// qpack switches the slice fast paths to QPack codecs (bitpack, FOR,
	// Gorilla, raw-LE bulk). When set, the header's FlagQPack bit is
	// emitted as an early hint to legacy readers.
	qpack bool

	// depth tracks nested pointer/struct traversal. Pointer cycles do
	// not crash the process; encodePtr increments depth on entry and
	// returns ErrCycleDetected when it exceeds maxDepth. Lightweight
	// alternative to a per-pointer set (no allocation per call).
	depth    int
	maxDepth int
}

// DefaultMaxDepth caps reflect-path pointer/struct recursion. Set
// large enough for any legitimate payload (10 000) while still
// rejecting genuine cycles before the goroutine stack overflows.
const DefaultMaxDepth = 10_000

// Mode selects the wire dialect.
type Mode uint8

const (
	// Fast writes strings and []byte in line. No intern bookkeeping.
	Fast Mode = 0

	// Dense writes repeated strings/bytes once and references them by ID
	// thereafter. Smaller output on repetitive payloads; slightly slower
	// per call.
	Dense Mode = 1
)

// NewEncoder returns an Encoder. The internal buffer is allocated lazily
// on first write.
func NewEncoder(mode Mode) *Encoder {
	e := &Encoder{
		mode:            mode,
		minIntern:       4,
		maxStateEntries: 1 << 14,
		maxDepth:        DefaultMaxDepth,
	}
	if mode == Dense {
		e.state = newEncState()
	}
	return e
}

// NewEncoderOnBuf returns an Encoder that appends to buf at its current
// length. The buffer is NOT truncated; pass an empty slice for a fresh
// encoding. Call SetBuffer afterwards to truncate.
func NewEncoderOnBuf(buf []byte, mode Mode) *Encoder {
	e := NewEncoder(mode)
	e.buf = buf
	return e
}

// Reset truncates the buffer and resets the intern table. Capacities are
// preserved. Mode and tuning knobs are not touched.
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
	e.headerOut = false
	if e.state != nil {
		e.state.reset()
	}
	e.depth = 0
	if e.maxDepth == 0 {
		e.maxDepth = DefaultMaxDepth
	}
}

// SetMaxDepth caps reflect-path pointer/struct recursion. The default
// (DefaultMaxDepth = 10000) is sufficient for any normal payload and
// rejects pointer cycles before they stack-overflow the goroutine.
// Set to 0 to disable the check (legacy behaviour).
func (e *Encoder) SetMaxDepth(d int) { e.maxDepth = d }

// Bytes returns the encoded payload. It aliases the encoder's buffer and
// is only valid until the next write or Reset.
func (e *Encoder) Bytes() []byte { return e.buf }

// Take returns the encoded payload and detaches it from the encoder. The
// caller takes ownership.
func (e *Encoder) Take() []byte {
	out := e.buf
	e.buf = nil
	e.headerOut = false
	if e.state != nil {
		e.state.reset()
	}
	return out
}

// SetBuffer installs a caller-owned buffer (truncated to length 0).
func (e *Encoder) SetBuffer(b []byte) {
	e.buf = b[:0]
	e.headerOut = false
}

// AdoptBuffer installs b as the working buffer, preserving its current
// length. Used to continue writing into a buffer that already carries
// data — for example, after a nested type returned its extended slice.
// headerOut is left unchanged.
func (e *Encoder) AdoptBuffer(b []byte) {
	e.buf = b
}

// SetIntern overrides the Dense-mode tuning knobs. Zero values keep the
// current setting.
func (e *Encoder) SetIntern(min int, cap int) {
	if min > 0 {
		e.minIntern = min
	}
	if cap > 0 {
		e.maxStateEntries = cap
	}
}

func (e *Encoder) writeHeader() {
	if e.headerOut {
		return
	}
	flag := byte(0)
	if e.mode == Dense {
		flag |= FlagDense
	}
	if e.qpack {
		flag |= FlagQPack
	}
	e.buf = append(e.buf, Magic0, Magic1, Magic2, Version1, flag)
	e.headerOut = true
}

// SetQPack toggles QPack codec emission. When true, slice fast paths
// produce packed/encoded forms (bitpacked bools, FOR-packed integers,
// Gorilla-encoded floats, raw-LE bulk) instead of per-element tag streams.
// Setting must happen before the first write of the value (the header is
// emitted lazily and carries the FlagQPack hint when this is on).
func (e *Encoder) SetQPack(v bool) { e.qpack = v }

// QPack reports whether QPack codec emission is enabled.
func (e *Encoder) QPack() bool { return e.qpack }

// EnsureHeader forces a header write if one has not been emitted yet.
func (e *Encoder) EnsureHeader() { e.writeHeader() }

// MarkHeaderWritten tells the encoder the QDF header is already present
// in its buffer (e.g. left there by a parent encoder). The next write
// will skip the header emission.
func (e *Encoder) MarkHeaderWritten() { e.headerOut = true }

// AppendBytes appends raw, already-valid wire bytes. Bypasses tag
// dispatch; used by generated code to emit pre-encoded field-name
// prefixes.
func (e *Encoder) AppendBytes(p []byte) {
	e.writeHeader()
	e.buf = append(e.buf, p...)
}

// ----- primitives -----

func (e *Encoder) WriteNil() {
	e.writeHeader()
	e.buf = append(e.buf, tagNil)
}

func (e *Encoder) WriteBool(v bool) {
	e.writeHeader()
	if v {
		e.buf = append(e.buf, tagTrue)
	} else {
		e.buf = append(e.buf, tagFalse)
	}
}

func (e *Encoder) WriteUint(v uint64) {
	e.writeHeader()
	switch {
	case v <= tagFixintMax:
		e.buf = append(e.buf, byte(v))
	case v <= math.MaxUint8:
		e.buf = append(e.buf, tagUint8, byte(v))
	case v <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagUint16), uint16(v))
	case v <= math.MaxUint32:
		e.buf = appendU32(append(e.buf, tagUint32), uint32(v))
	default:
		e.buf = appendU64(append(e.buf, tagUint64), v)
	}
}

func (e *Encoder) WriteInt(v int64) {
	e.writeHeader()
	if v >= 0 {
		e.WriteUint(uint64(v))
		return
	}
	switch {
	case v >= -negfixintMaxAbs:
		// negfixint packs -1..-8 as 0xD8..0xDF; decoder mirrors as
		// -(int8(tag & 0x07) + 1).
		e.buf = append(e.buf, tagNegfixint|byte(-v-1))
	case v >= math.MinInt8:
		e.buf = append(e.buf, tagInt8, byte(int8(v)))
	case v >= math.MinInt16:
		e.buf = appendU16(append(e.buf, tagInt16), uint16(int16(v)))
	case v >= math.MinInt32:
		e.buf = appendU32(append(e.buf, tagInt32), uint32(int32(v)))
	default:
		e.buf = appendU64(append(e.buf, tagInt64), uint64(v))
	}
}

func (e *Encoder) WriteFloat32(v float32) {
	e.writeHeader()
	e.buf = appendU32(append(e.buf, tagFloat32), math.Float32bits(v))
}

func (e *Encoder) WriteFloat64(v float64) {
	e.writeHeader()
	e.buf = appendU64(append(e.buf, tagFloat64), math.Float64bits(v))
}

// WriteString writes s. In Dense mode, eligible strings are intern-encoded.
func (e *Encoder) WriteString(s string) {
	e.writeHeader()
	st := e.state
	if st != nil && len(s) >= e.minIntern && len(st.ids) < e.maxStateEntries {
		id, ok := st.lookupOrAssign(s)
		if ok {
			// Repeat hot path: hand-inlined out of emitStateRef so the
			// most common Dense hit avoids the non-inlinable call.
			if st.lastID == id {
				e.buf = append(e.buf, tagStateRepeat)
				st.pairRecord(id, id)
				return
			}
			e.emitStateRef(id)
			return
		}
		e.buf = append(e.buf, tagInternStr)
		e.buf = appendUvarint(e.buf, uint64(len(s)))
		e.buf = appendString(e.buf, s)
		if st.lastID != lruInvalidID {
			st.pairRecord(st.lastID, id)
		}
		st.lastID = id
		return
	}
	if st != nil {
		st.lastID = lruInvalidID
	}
	e.writeStringInline(s)
}

// emitStateRef writes a state-ref to id. Four forms are possible, the
// encoder picks the smallest:
//
//	tagStateRepeat                   1 byte total, when id == lastID
//	tagStatePair  + varuint(pairR)   1 + uvarintLen(pairR) bytes,
//	                                 when id is in lastID's predictor ring
//	tagStateMTF   + varuint(mtfR)    1 + uvarintLen(mtfR) bytes
//	tagStateRef   + varuint(id)      1 + uvarintLen(id) bytes
//
// MTF rank comes from the encState LRU. The pair rank comes from the
// per-prev successor ring (Markov-1 predictor). The wire never grows
// over the plain tagStateRef encoding because we only pick the
// alternative when its varuint is strictly shorter than the raw id
// varuint.
//
// Every successful emit moves id to the LRU head AND records the
// (prev, id) transition in the pair predictor so the decoder's mirror
// chain stays in sync.
func (e *Encoder) emitStateRef(id uint32) {
	st := e.state
	if st.lastID == id {
		e.buf = append(e.buf, tagStateRepeat)
		st.pairRecord(id, id)
		return
	}
	prev := st.lastID
	prevValid := prev != lruInvalidID
	// Small-id fast path: id<128 means the raw state-ref is already a
	// 2-byte literal (tag + 1-byte varuint). Neither MTF (rank varuint
	// ≥1 byte) nor Pair (rank varuint =1 byte) can be strictly shorter,
	// so we write the raw form directly and skip the LRU walk entirely.
	if id < 0x80 {
		st.lruMoveFront(id)
		e.buf = append(e.buf, tagStateRef, byte(id))
		if prevValid {
			st.pairRecord(prev, id)
		}
		st.lastID = id
		return
	}
	// Pair predictor: when (prev, id) is in the predictor ring the
	// payload is always a single byte (ranks 0..3), so the pair form
	// is strictly shorter than the raw state-ref whenever id needs a
	// multi-byte varuint — which the small-id branch above just
	// excluded.
	if prevValid {
		if r, ok := st.pairLookup(prev, id); ok {
			st.lruMoveFront(id)
			e.buf = append(e.buf, tagStatePair, byte(r))
			st.pairRecord(prev, id)
			st.lastID = id
			return
		}
	}
	// MTF — fall back on raw if the rank varuint is not shorter than
	// the id varuint.
	idLen := uvarintLen(uint64(id))
	rank := st.lruMoveToFront(id)
	if rankLen := uvarintLen(uint64(rank)); rankLen < idLen {
		e.buf = append(e.buf, tagStateMTF)
		e.buf = appendUvarint(e.buf, uint64(rank))
	} else {
		e.buf = append(e.buf, tagStateRef)
		e.buf = appendUvarint(e.buf, uint64(id))
	}
	if prevValid {
		st.pairRecord(prev, id)
	}
	st.lastID = id
}

// WriteStringInline forces an in-line encoding even when Dense intern would
// be eligible. Use for fields known to be unique per message.
func (e *Encoder) WriteStringInline(s string) {
	e.writeHeader()
	e.writeStringInline(s)
}

func (e *Encoder) writeStringInline(s string) {
	n := len(s)
	// One Grow up front for worst-case header (5 B) + body beats append's
	// amortized growth on the hot path.
	b := slices.Grow(e.buf, 5+n)
	switch {
	case n <= int(tagFixstrMask):
		b = append(b, tagFixstr|byte(n))
	case n <= math.MaxUint8:
		b = append(b, tagStr8, byte(n))
	case n <= math.MaxUint16:
		b = appendU16(append(b, tagStr16), uint16(n))
	default:
		b = appendU32(append(b, tagStr32), uint32(n))
	}
	b = append(b, s...)
	e.buf = b
}

// WriteBytes writes a []byte. In Dense mode, eligible payloads are
// intern-encoded. The intern table is keyed on the byte sequence, so a
// string and a []byte with identical content share an ID.
func (e *Encoder) WriteBytes(b []byte) {
	e.writeHeader()
	st := e.state
	if st != nil && len(b) >= e.minIntern && len(st.ids) < e.maxStateEntries {
		key := unsafestr.String(b)
		id, ok := st.lookupOrAssign(key)
		if ok {
			if st.lastID == id {
				e.buf = append(e.buf, tagStateRepeat)
				st.pairRecord(id, id)
				return
			}
			e.emitStateRef(id)
			return
		}
		e.buf = append(e.buf, tagInternBin)
		e.buf = appendUvarint(e.buf, uint64(len(b)))
		e.buf = append(e.buf, b...)
		if st.lastID != lruInvalidID {
			st.pairRecord(st.lastID, id)
		}
		st.lastID = id
		return
	}
	if st != nil {
		st.lastID = lruInvalidID
	}
	e.writeBytesInline(b)
}

func (e *Encoder) writeBytesInline(p []byte) {
	n := len(p)
	out := slices.Grow(e.buf, 5+n)
	switch {
	case n <= math.MaxUint8:
		out = append(out, tagBin8, byte(n))
	case n <= math.MaxUint16:
		out = appendU16(append(out, tagBin16), uint16(n))
	default:
		out = appendU32(append(out, tagBin32), uint32(n))
	}
	out = append(out, p...)
	e.buf = out
}

// WriteArrayHeader writes the header for an array of n elements. The
// caller must follow with exactly n element writes.
func (e *Encoder) WriteArrayHeader(n int) {
	e.writeHeader()
	switch {
	case n <= int(tagFixarrMask):
		e.buf = append(e.buf, tagFixarr|byte(n))
	case n <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagArr16), uint16(n))
	default:
		e.buf = appendU32(append(e.buf, tagArr32), uint32(n))
	}
}

// WriteMapHeader writes the header for a map of n entries. The caller
// must follow with exactly n key/value pairs.
func (e *Encoder) WriteMapHeader(n int) {
	e.writeHeader()
	switch {
	case n <= math.MaxUint8:
		e.buf = append(e.buf, tagMap8, byte(n))
	case n <= math.MaxUint16:
		e.buf = appendU16(append(e.buf, tagMap16), uint16(n))
	default:
		e.buf = appendU32(append(e.buf, tagMap32), uint32(n))
	}
}

// WriteTimestampNano writes a Unix-nanoseconds timestamp.
func (e *Encoder) WriteTimestampNano(ns int64) {
	e.writeHeader()
	e.buf = appendU64(append(e.buf, tagTimestamp), uint64(ns))
}

// ----- helpers -----

func appendU16(b []byte, v uint16) []byte { return binary.LittleEndian.AppendUint16(b, v) }
func appendU32(b []byte, v uint32) []byte { return binary.LittleEndian.AppendUint32(b, v) }
func appendU64(b []byte, v uint64) []byte { return binary.LittleEndian.AppendUint64(b, v) }

// appendString uses the runtime's append-string fast path to avoid the
// implicit []byte(s) copy.
func appendString(b []byte, s string) []byte { return append(b, s...) }

func readU16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func readU32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func readU64(b []byte) uint64 { return binary.LittleEndian.Uint64(b) }
