package qdf

import (
	"math"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

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
//
// When the tag at the cursor resolves through the intern table —
// tagInternStr, tagStateRef, tagStateMTF, tagStatePair, tagStateRepeat —
// readStringBytes leaves d.state.lastID set to the entry it just
// touched. We return the pre-materialised d.state.stringValues[id]
// from that path instead of allocating a fresh `string(b)` copy on
// every state-ref hit. Inline reads (fixstr, str8/16/32) set
// lastID = lruInvalidID, fall through, and pay the copy as before.
func (d *Decoder) ReadString() (string, error) {
	b, err := d.readStringBytes()
	if err != nil {
		return "", err
	}
	if d.noCopy {
		return unsafestr.String(b), nil
	}
	if d.state != nil && d.state.lastID != lruInvalidID {
		if s, ok := d.state.getString(d.state.lastID, d.arena); ok {
			return s, nil
		}
	}
	if d.arena != nil && len(b) > 0 {
		return d.arena.appendStr(b), nil
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
			d.state.lastID = lruInvalidID
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
			d.state.lastID = lruInvalidID
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
			d.state.lastID = lruInvalidID
		}
		return out, nil
	case tagStr32, tagBin32:
		if d.i+4 > len(d.buf) {
			return nil, ErrShortBuffer
		}
		n64 := uint64(readU32(d.buf[d.i:]))
		d.i += 4
		// Compare in uint64 before narrowing: on a 32-bit int a length near
		// 2^32 would become negative and slip past a `d.i+n > len` check, then
		// panic the slice with high < low. Mirrors the tagInternStr guard below.
		if n64 > uint64(len(d.buf)-d.i) {
			return nil, ErrShortBuffer
		}
		n := int(n64)
		out := d.buf[d.i : d.i+n]
		d.i += n
		if invalidateLast {
			d.state.lastID = lruInvalidID
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
		// A conforming encoder caps the intern table at maxInternEntries so ids
		// stay below the 0xFFFF MRU/LRU sentinel; a hostile stream claiming more
		// records would assign id 0xFFFF and corrupt the side-cache chains.
		// Reject it (mirrors the encoder-side cap).
		if len(d.state.values) >= maxInternEntries {
			return nil, ErrInvalidLength
		}
		// Register the bytes as-is; they alias the input. If the caller
		// later turns this into a copy, the table still references the alias
		// which is fine for the lifetime of the buffer.
		id := d.state.append(out)
		if d.state.lastID != lruInvalidID {
			d.state.pairRecord(d.state.lastID, id)
		}
		d.state.lastID = id
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
		// Reject before the uint32 narrowing: a hostile 10-byte varint above
		// 2^32 would truncate to a small, possibly-valid id and silently
		// resolve the WRONG interned string (and desync the LRU/pair chains).
		if id64 > math.MaxUint32 {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(uint32(id64))
		if !ok {
			return nil, ErrUnknownStateID
		}
		// Mirror encoder's LRU update so a subsequent tagStateMTF
		// resolves to the same ID position.
		d.state.lruMoveToFront(uint32(id64))
		if d.state.lastID != lruInvalidID {
			d.state.pairRecord(d.state.lastID, uint32(id64))
		}
		d.state.lastID = uint32(id64)
		return out, nil
	case tagStateMTF:
		rank64, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += n
		if d.state == nil {
			return nil, ErrUnknownStateID
		}
		// Reject before narrowing: a >2^32 varint would truncate to a small
		// rank and resolve a wrong entry instead of failing.
		if rank64 > math.MaxUint32 {
			return nil, ErrUnknownStateID
		}
		// Side-cache lookup: the encoder emits tagStateMTF only when
		// rank fits in fewer varuint bytes than the raw id, which
		// caps rank at mruRingSize-1 for the common 2-byte id case.
		// The MRU ring resolves that range in O(1) without walking
		// the LRU chain. Older/larger ranks fall back to the chain
		// walk for forward-compatibility.
		rank := uint32(rank64)
		id, ok := d.state.mruIDAtRank(rank)
		if !ok {
			id, ok = d.state.lruIDAtRank(rank)
		}
		if !ok {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(id)
		if !ok {
			return nil, ErrUnknownStateID
		}
		d.state.lruMoveToFront(id)
		if d.state.lastID != lruInvalidID {
			d.state.pairRecord(d.state.lastID, id)
		}
		d.state.lastID = id
		return out, nil
	case tagStatePair:
		if d.state == nil || d.state.lastID == lruInvalidID {
			return nil, ErrUnknownStateID
		}
		rank64, n := readUvarint(d.buf[d.i:])
		if n <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += n
		if rank64 >= pairPredK {
			return nil, ErrUnknownStateID
		}
		prev := d.state.lastID
		id, ok := d.state.pairAtRank(prev, uint8(rank64))
		if !ok {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(id)
		if !ok {
			return nil, ErrUnknownStateID
		}
		// Same post-emit bookkeeping as tagStateRef so the encoder
		// and decoder mirror chains diverge nowhere.
		d.state.lruMoveToFront(id)
		d.state.pairRecord(prev, id)
		d.state.lastID = id
		return out, nil
	case tagStateRepeat:
		if d.state == nil || d.state.lastID == lruInvalidID {
			return nil, ErrUnknownStateID
		}
		out, ok := d.state.get(d.state.lastID)
		if !ok {
			return nil, ErrUnknownStateID
		}
		// Pair predictor: mirror encoder's self-record.
		d.state.pairRecord(d.state.lastID, d.state.lastID)
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
		// uint64 before narrowing: on a 32-bit int a count >= 2^31 wraps negative,
		// breaking the non-negative API contract and slipping past caller bounds
		// checks. Each element is >= 1 byte, so a count over the remaining bytes is
		// impossible. Mirrors the readStringBytes / decoder_skip tagArr32 guard.
		n64 := uint64(readU32(d.buf[d.i:]))
		d.i += 4
		if n64 > uint64(len(d.buf)-d.i) {
			return 0, ErrShortBuffer
		}
		return int(n64), nil
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
		// Same 32-bit wrap guard as tagArr32; each entry is >= 2 bytes (key+value),
		// so a count over the remaining bytes is impossible.
		n64 := uint64(readU32(d.buf[d.i:]))
		d.i += 4
		if n64 > uint64(len(d.buf)-d.i) {
			return 0, ErrShortBuffer
		}
		return int(n64), nil
	}
	return 0, ErrTypeMismatch
}

// ReadTimestamp reads a full-range timestamp and returns (sec, nsec, err).
// sec is seconds since Unix epoch (may be negative for pre-1970 instants).
// nsec is nanoseconds in [0, 999_999_999].
// This replaces the old fixed-8-byte UnixNano encoding (clean break).
func (d *Decoder) ReadTimestamp() (sec int64, nsec uint32, err error) {
	t, err := d.next()
	if err != nil {
		return 0, 0, err
	}
	if t != tagTimestamp {
		return 0, 0, ErrTypeMismatch
	}
	// Read zigzag-encoded seconds.
	secZ, n := readUvarint(d.buf[d.i:])
	if n <= 0 {
		return 0, 0, ErrInvalidLength
	}
	d.i += n
	// Read nanoseconds.
	nsecU, n := readUvarint(d.buf[d.i:])
	if n <= 0 {
		return 0, 0, ErrInvalidLength
	}
	d.i += n
	if nsecU > 999_999_999 {
		return 0, 0, ErrInvalidLength
	}
	return zigzagDecode64(secZ), uint32(nsecU), nil
}
