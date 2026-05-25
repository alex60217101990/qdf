package qdf

import (
	"math"

	"github.com/alex60217101990/qdf/internal/intern"
	"github.com/alex60217101990/qdf/internal/unsafestr"
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
	if flags&FlagDense != 0 {
		d.mode = Dense
		if d.state == nil {
			d.state = newDecState()
		}
	}
	d.i += 5
	d.headerRead = true
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

// ----- primitives -----

func (d *Decoder) ReadNil() error {
	t, err := d.next()
	if err != nil {
		return err
	}
	if t != tagNil {
		return ErrTypeMismatch
	}
	return nil
}

func (d *Decoder) ReadBool() (bool, error) {
	t, err := d.next()
	if err != nil {
		return false, err
	}
	switch t {
	case tagTrue:
		return true, nil
	case tagFalse:
		return false, nil
	default:
		return false, ErrTypeMismatch
	}
}

// ReadUint reads a value that was encoded with WriteUint or WriteInt with a
// non-negative argument. Returns an error if the next value is negative.
func (d *Decoder) ReadUint() (uint64, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	return d.decodeUint(t)
}

func (d *Decoder) decodeUint(t byte) (uint64, error) {
	if t <= tagFixintMax {
		return uint64(t), nil
	}
	switch t {
	case tagUint8:
		if d.i >= len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := uint64(d.buf[d.i])
		d.i++
		return v, nil
	case tagUint16:
		if d.i+2 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := uint64(readU16(d.buf[d.i:]))
		d.i += 2
		return v, nil
	case tagUint32:
		if d.i+4 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := uint64(readU32(d.buf[d.i:]))
		d.i += 4
		return v, nil
	case tagUint64:
		if d.i+8 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := readU64(d.buf[d.i:])
		d.i += 8
		return v, nil
	}
	return 0, ErrTypeMismatch
}

func (d *Decoder) ReadInt() (int64, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	return d.decodeInt(t)
}

func (d *Decoder) decodeInt(t byte) (int64, error) {
	// fixint positive
	if t <= tagFixintMax {
		return int64(t), nil
	}
	// negfixint range 0xD8..0xDF
	if t >= tagNegfixint && t <= tagNegfixint|tagNegfixintMask {
		return -int64(t&tagNegfixintMask) - 1, nil
	}
	switch t {
	case tagInt8:
		if d.i >= len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := int64(int8(d.buf[d.i]))
		d.i++
		return v, nil
	case tagInt16:
		if d.i+2 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := int64(int16(readU16(d.buf[d.i:])))
		d.i += 2
		return v, nil
	case tagInt32:
		if d.i+4 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := int64(int32(readU32(d.buf[d.i:])))
		d.i += 4
		return v, nil
	case tagInt64:
		if d.i+8 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := int64(readU64(d.buf[d.i:]))
		d.i += 8
		return v, nil
	}
	// fall back to the unsigned decoder for tagUintN
	u, err := d.decodeUint(t)
	if err != nil {
		return 0, err
	}
	if u > math.MaxInt64 {
		return 0, ErrTypeMismatch
	}
	return int64(u), nil
}

func (d *Decoder) ReadFloat32() (float32, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	if t != tagFloat32 {
		return 0, ErrTypeMismatch
	}
	if d.i+4 > len(d.buf) {
		return 0, ErrShortBuffer
	}
	v := math.Float32frombits(readU32(d.buf[d.i:]))
	d.i += 4
	return v, nil
}

func (d *Decoder) ReadFloat64() (float64, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	switch t {
	case tagFloat64:
		if d.i+8 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := math.Float64frombits(readU64(d.buf[d.i:]))
		d.i += 8
		return v, nil
	case tagFloat32:
		if d.i+4 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		v := float64(math.Float32frombits(readU32(d.buf[d.i:])))
		d.i += 4
		return v, nil
	}
	return 0, ErrTypeMismatch
}

// ReadString returns the next string. If the decoder is in noCopy mode the
// returned string aliases the input buffer; otherwise a copy is made.
func (d *Decoder) ReadString() (string, error) {
	b, err := d.readStringBytes()
	if err != nil {
		return "", err
	}
	if d.noCopy {
		return unsafestr.String(b), nil
	}
	return string(b), nil
}

// readStringBytes returns the raw bytes of a string/bin value without
// allocating; the returned slice aliases the input buffer or the decoder
// state table.
func (d *Decoder) readStringBytes() ([]byte, error) {
	t, err := d.next()
	if err != nil {
		return nil, err
	}
	// Inline (non-intern) string / bin reads break the Markov-0 chain so
	// a subsequent tagStateRepeat cannot resurrect a stale state-ref
	// from before the inline emission. State-ref and intern branches
	// below restore the invariant explicitly.
	invalidateLast := d.state != nil
	// fixstr
	if t >= tagFixstr && t <= tagFixstr|tagFixstrMask {
		n := int(t & tagFixstrMask)
		if d.i+n > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out := d.buf[d.i : d.i+n]
		d.i += n
		if invalidateLast {
			d.state.lastValid = false
		}
		return out, nil
	}
	switch t {
	case tagStr8, tagBin8:
		if d.i >= len(d.buf) {
			return nil, ErrShortBuffer
		}
		n := int(d.buf[d.i])
		d.i++
		if d.i+n > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out := d.buf[d.i : d.i+n]
		d.i += n
		if invalidateLast {
			d.state.lastValid = false
		}
		return out, nil
	case tagStr16, tagBin16:
		if d.i+2 > len(d.buf) {
			return nil, ErrShortBuffer
		}
		n := int(readU16(d.buf[d.i:]))
		d.i += 2
		if d.i+n > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out := d.buf[d.i : d.i+n]
		d.i += n
		if invalidateLast {
			d.state.lastValid = false
		}
		return out, nil
	case tagStr32, tagBin32:
		if d.i+4 > len(d.buf) {
			return nil, ErrShortBuffer
		}
		n := int(readU32(d.buf[d.i:]))
		d.i += 4
		if d.i+n > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out := d.buf[d.i : d.i+n]
		d.i += n
		if invalidateLast {
			d.state.lastValid = false
		}
		return out, nil
	case tagInternStr, tagInternBin:
		// Read length-prefixed payload, then register it in the state table.
		n64, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += n
		// Length validated in uint64 to avoid the int sign-bit pitfall on a
		// hostile varint. A 10-byte varint can encode values up to 2^64-1;
		// our buffer is always < 2^63, so the uint64 comparison is safe.
		if n64 > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
		nn := int(n64)
		out := d.buf[d.i : d.i+nn]
		d.i += nn
		if d.state == nil {
			d.state = newDecState()
		}
		// Register the bytes as-is; they alias the input. If the caller
		// later turns this into a copy, the table still references the alias
		// which is fine for the lifetime of the buffer.
		id := d.state.append(out)
		d.state.lastID = id
		d.state.lastValid = true
		return out, nil
	case tagStateRef:
		id64, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += n
		if d.state == nil {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(uint32(id64))
		if !ok {
			return nil, ErrUnknownStateID
		}
		d.state.lastID = uint32(id64)
		d.state.lastValid = true
		return out, nil
	case tagStateRepeat:
		if d.state == nil || !d.state.lastValid {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(d.state.lastID)
		if !ok {
			return nil, ErrUnknownStateID
		}
		return out, nil
	}
	return nil, ErrTypeMismatch
}

// ReadBytes returns a []byte value. With noCopy = true the result aliases
// the input.
func (d *Decoder) ReadBytes() ([]byte, error) {
	b, err := d.readStringBytes()
	if err != nil {
		return nil, err
	}
	if d.noCopy {
		return b, nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

// ReadArrayHeader returns the element count.
func (d *Decoder) ReadArrayHeader() (int, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	if t >= tagFixarr && t <= tagFixarr|tagFixarrMask {
		return int(t & tagFixarrMask), nil
	}
	switch t {
	case tagArr16:
		if d.i+2 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		n := int(readU16(d.buf[d.i:]))
		d.i += 2
		return n, nil
	case tagArr32:
		if d.i+4 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		n := int(readU32(d.buf[d.i:]))
		d.i += 4
		return n, nil
	}
	return 0, ErrTypeMismatch
}

// ReadMapHeader returns the key/value pair count.
func (d *Decoder) ReadMapHeader() (int, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	switch t {
	case tagMap8:
		if d.i >= len(d.buf) {
			return 0, ErrShortBuffer
		}
		n := int(d.buf[d.i])
		d.i++
		return n, nil
	case tagMap16:
		if d.i+2 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		n := int(readU16(d.buf[d.i:]))
		d.i += 2
		return n, nil
	case tagMap32:
		if d.i+4 > len(d.buf) {
			return 0, ErrShortBuffer
		}
		n := int(readU32(d.buf[d.i:]))
		d.i += 4
		return n, nil
	}
	return 0, ErrTypeMismatch
}

func (d *Decoder) ReadTimestampNano() (int64, error) {
	t, err := d.next()
	if err != nil {
		return 0, err
	}
	if t != tagTimestamp {
		return 0, ErrTypeMismatch
	}
	if d.i+8 > len(d.buf) {
		return 0, ErrShortBuffer
	}
	v := int64(readU64(d.buf[d.i:]))
	d.i += 8
	return v, nil
}

// Skip advances past one value without materializing it.
func (d *Decoder) Skip() error {
	t, err := d.peekTag()
	if err != nil {
		return err
	}
	switch {
	case t <= tagFixintMax:
		d.i++
		return nil
	case t >= tagFixstr && t <= tagFixstr|tagFixstrMask:
		n := int(t & tagFixstrMask)
		if d.i+1+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 1 + n
		return nil
	case t >= tagFixarr && t <= tagFixarr|tagFixarrMask:
		n := int(t & tagFixarrMask)
		d.i++
		for range n {
			if err := d.Skip(); err != nil {
				return err
			}
		}
		return nil
	case t >= tagNegfixint && t <= tagNegfixint|tagNegfixintMask:
		d.i++
		return nil
	}
	switch t {
	case tagNil, tagTrue, tagFalse:
		d.i++
		return nil
	case tagUint8, tagInt8:
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 2
		return nil
	case tagUint16, tagInt16:
		if d.i+3 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 3
		return nil
	case tagUint32, tagInt32, tagFloat32:
		if d.i+5 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 5
		return nil
	case tagUint64, tagInt64, tagFloat64, tagTimestamp:
		if d.i+9 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += 9
		return nil
	case tagStr8, tagBin8, tagMap8:
		// Map8 is a count, not a byte length; handle separately below.
		if t == tagMap8 {
			n, err := d.ReadMapHeader()
			if err != nil {
				return err
			}
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		d.i++ // tag
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		n := int(d.buf[d.i])
		d.i++
		if d.i+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += n
		return nil
	case tagStr16, tagBin16, tagArr16, tagMap16:
		d.i++
		if t == tagArr16 {
			if d.i+2 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU16(d.buf[d.i:]))
			d.i += 2
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if t == tagMap16 {
			if d.i+2 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU16(d.buf[d.i:]))
			d.i += 2
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		n := int(readU16(d.buf[d.i:]))
		d.i += 2
		if d.i+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += n
		return nil
	case tagStr32, tagBin32, tagArr32, tagMap32:
		d.i++
		if t == tagArr32 {
			if d.i+4 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU32(d.buf[d.i:]))
			d.i += 4
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if t == tagMap32 {
			if d.i+4 > len(d.buf) {
				return ErrShortBuffer
			}
			n := int(readU32(d.buf[d.i:]))
			d.i += 4
			for range n {
				if err := d.Skip(); err != nil {
					return err
				}
				if err := d.Skip(); err != nil {
					return err
				}
			}
			return nil
		}
		if d.i+4 > len(d.buf) {
			return ErrShortBuffer
		}
		n := int(readU32(d.buf[d.i:]))
		d.i += 4
		if d.i+n > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += n
		return nil
	case tagInternStr, tagInternBin:
		// Read+register; Skip semantics still need the state table to stay
		// in sync with the stream.
		_, err := d.readStringBytes()
		return err
	case tagStateRef, tagStateRepeat:
		_, err := d.readStringBytes()
		return err
	case tagPackBool:
		d.i++
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Validate the element count in uint64 BEFORE the signed cast so
		// a hostile varuint cannot drive nBytes negative and corrupt
		// d.i. n64 elements need ceil(n64/8) bytes.
		rem := uint64(len(d.buf) - d.i)
		if n64 > rem*8 {
			return ErrShortBuffer
		}
		d.i += int((n64 + 7) >> 3)
		return nil
	case tagPackRaw:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		w := qpackRawWidthBytes(k)
		if w == 0 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 > uint64(len(d.buf)-d.i)/uint64(w) {
			return ErrShortBuffer
		}
		d.i += int(n64) * w
		return nil
	case tagPackFor:
		d.i++
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		// kind, bits
		d.i++ // skip kind
		bitsPer := int(d.buf[d.i])
		d.i++
		if bitsPer > qpackForMaxBits {
			return ErrBadTag
		}
		// min varuint
		_, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// Same overflow guard as tagPackBool. n64 elements times
		// bitsPer bits => ceil/8 bytes; validate in uint64 to avoid the
		// sign-bit pitfall on a hostile varuint.
		rem := uint64(len(d.buf) - d.i)
		if bitsPer > 0 && n64 > rem*8/uint64(bitsPer) {
			return ErrShortBuffer
		}
		d.i += int((n64*uint64(bitsPer) + 7) / 8)
		return nil
	case tagPackGorilla:
		d.i++
		if d.i >= len(d.buf) {
			return ErrShortBuffer
		}
		k := d.buf[d.i]
		d.i++
		w := qpackRawWidthBytes(k)
		if w != 4 && w != 8 {
			return ErrBadTag
		}
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 == 0 {
			return nil
		}
		if d.i+w > len(d.buf) {
			return ErrShortBuffer
		}
		d.i += w
		// numBits varuint
		nb64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		rem := uint64(len(d.buf) - d.i)
		if nb64 > rem*8 {
			return ErrShortBuffer
		}
		d.i += int((nb64 + 7) >> 3)
		return nil
	case tagPackDeltaFor:
		d.i++
		if d.i+2 > len(d.buf) {
			return ErrShortBuffer
		}
		d.i++ // skip kind
		bitsPer := int(d.buf[d.i])
		d.i++
		if bitsPer > qpackForMaxBits {
			return ErrBadTag
		}
		// firstVal varuint
		_, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		// minDelta varuint
		_, nr = readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		n64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr
		if n64 >= 2 {
			// Body holds (n-1) elements at bitsPer each. Compute the
			// byte size in uint64 to keep the bounds check overflow-safe.
			bodyBits := (n64 - 1) * uint64(bitsPer)
			bodyBytes := (bodyBits + 7) >> 3
			if bodyBytes > uint64(len(d.buf)-d.i) {
				return ErrShortBuffer
			}
			d.i += int(bodyBytes)
		}
		return nil
	}
	return ErrBadTag
}
