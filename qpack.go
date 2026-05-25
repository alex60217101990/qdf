package qdf

import (
	"math"
	"slices"
	"unsafe"
)

// QPack raw-LE codec kind byte. Low two bits encode the element width as
// log2(bytes); next two bits encode the type family (uint, int, float).
const (
	qpackRawW1 = 0
	qpackRawW2 = 1
	qpackRawW4 = 2
	qpackRawW8 = 3

	qpackRawFamUint  = 0 << 2
	qpackRawFamInt   = 1 << 2
	qpackRawFamFloat = 2 << 2

	qpackKindUint8   = qpackRawFamUint | qpackRawW1
	qpackKindUint16  = qpackRawFamUint | qpackRawW2
	qpackKindUint32  = qpackRawFamUint | qpackRawW4
	qpackKindUint64  = qpackRawFamUint | qpackRawW8
	qpackKindInt8    = qpackRawFamInt | qpackRawW1
	qpackKindInt16   = qpackRawFamInt | qpackRawW2
	qpackKindInt32   = qpackRawFamInt | qpackRawW4
	qpackKindInt64   = qpackRawFamInt | qpackRawW8
	qpackKindFloat32 = qpackRawFamFloat | qpackRawW4
	qpackKindFloat64 = qpackRawFamFloat | qpackRawW8
)

func qpackRawWidthBytes(kind byte) int {
	switch kind & 0x03 {
	case qpackRawW1:
		return 1
	case qpackRawW2:
		return 2
	case qpackRawW4:
		return 4
	case qpackRawW8:
		return 8
	}
	return 0
}

// pickU64Codec analyses a []uint64 and decides which QPack codec yields
// the smallest wire form. Cost of the scan is O(n) (min/max + delta
// stats), which is dominated by the encode itself.
func pickU64Codec(s []uint64) (codec qpackCodec, mn uint64, forBits int, first uint64, minDelta int64, deltaBits int) {
	n := len(s)
	codec = qpackRaw
	if n == 0 {
		return
	}
	// raw cost: tag + kind + nVarUint + 8n
	rawCost := 2 + uvarintLen(uint64(n)) + 8*n

	var mx uint64
	mn, mx = minMaxU64(s)
	forBits = bitsForDelta(mx - mn)
	bestCost := rawCost
	if forBits <= qpackForMaxBits {
		c := qpackForSizeUnsigned(n, forBits, mn)
		if c < bestCost {
			bestCost = c
			codec = qpackFor
		}
	}
	if n >= 4 {
		first, minDelta, deltaBits = computeDeltaStatsU64(s)
		if deltaBits <= qpackForMaxBits {
			hdr := 3 + uvarintLen(first) + uvarintLen(zigzagEncode64(minDelta)) + uvarintLen(uint64(n))
			body := 0
			if n >= 2 {
				body = ((n - 1) * deltaBits) >> 3
				if ((n-1)*deltaBits)&7 != 0 {
					body++
				}
			}
			c := hdr + body
			if c < bestCost {
				bestCost = c
				codec = qpackDeltaFor
			}
		}
	}
	return
}

func pickI64Codec(s []int64) (codec qpackCodec, mn int64, forBits int, first int64, minDelta int64, deltaBits int) {
	n := len(s)
	codec = qpackRaw
	if n == 0 {
		return
	}
	rawCost := 2 + uvarintLen(uint64(n)) + 8*n

	var mx int64
	mn, mx = minMaxI64(s)
	forBits = bitsForDelta(uint64(mx) - uint64(mn))
	bestCost := rawCost
	if forBits <= qpackForMaxBits {
		c := qpackForSizeSigned(n, forBits, mn)
		if c < bestCost {
			bestCost = c
			codec = qpackFor
		}
	}
	if n >= 4 {
		first, minDelta, deltaBits = computeDeltaStatsI64(s)
		if deltaBits <= qpackForMaxBits {
			hdr := 3 + uvarintLen(zigzagEncode64(first)) + uvarintLen(zigzagEncode64(minDelta)) + uvarintLen(uint64(n))
			body := 0
			if n >= 2 {
				body = ((n - 1) * deltaBits) >> 3
				if ((n-1)*deltaBits)&7 != 0 {
					body++
				}
			}
			c := hdr + body
			if c < bestCost {
				bestCost = c
				codec = qpackDeltaFor
			}
		}
	}
	return
}

// qpackCodec identifies which QPack codec to invoke for a numeric slice.
type qpackCodec uint8

const (
	qpackRaw qpackCodec = iota
	qpackFor
	qpackDeltaFor
)

// QPack codec helpers. Each codec emits a single self-described tagged
// payload that replaces the per-element tag stream for one slice. The
// codecs are opt-in (Encoder.SetQPack); decoders accept the new tags
// unconditionally so a payload written with QPack can always be read.

// writePackedBool encodes a []bool as one bit per element, LSB-first within
// each byte. Wire form: tagPackBool, varuint(n), ceil(n/8) bytes. Two
// fewer bytes than the per-element tag form once n >= 2, and 8× smaller
// past n >= 16.
func (e *Encoder) writePackedBool(s []bool) {
	e.writeHeader()
	n := len(s)
	nBytes := (n + 7) >> 3
	// Worst-case header is 1 (tag) + 10 (varuint) bytes. Reserve once so
	// the body append is a single memmove past the grow.
	out := slices.Grow(e.buf, 11+nBytes)
	out = append(out, tagPackBool)
	out = appendUvarint(out, uint64(n))
	start := len(out)
	out = out[:start+nBytes]
	body := out[start : start+nBytes]
	clear(body)
	// packBoolsBitsLSB dispatches to AVX2 (VPSLLW + VPMOVMSKB, 32
	// bools per iteration) under qdf_simd on amd64; everything else
	// falls back to a scalar bit-by-bit pack.
	packBoolsBitsLSB(body, s, n)
	e.buf = out
}

// writePackedRawBytes emits a tagPackRaw header followed by the supplied
// little-endian payload. n is the element count; payload length must
// equal n * widthFromKind(kind).
func (e *Encoder) writePackedRawBytes(kind byte, n int, payload []byte) {
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+len(payload))
	out = append(out, tagPackRaw, kind)
	out = appendUvarint(out, uint64(n))
	out = append(out, payload...)
	e.buf = out
}

// writePackedUint64Slice writes []uint64 as a raw-LE bulk payload. On a
// little-endian target this is a single memmove; on big-endian, an
// element-wise LE emit loop.
func (e *Encoder) writePackedUint64Slice(s []uint64) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*8)
		e.writePackedRawBytes(qpackKindUint64, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*8)
	out = append(out, tagPackRaw, qpackKindUint64)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU64(out, v)
	}
	e.buf = out
}

func (e *Encoder) writePackedInt64Slice(s []int64) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*8)
		e.writePackedRawBytes(qpackKindInt64, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*8)
	out = append(out, tagPackRaw, qpackKindInt64)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU64(out, uint64(v))
	}
	e.buf = out
}

func (e *Encoder) writePackedUint32Slice(s []uint32) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*4)
		e.writePackedRawBytes(qpackKindUint32, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*4)
	out = append(out, tagPackRaw, qpackKindUint32)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU32(out, v)
	}
	e.buf = out
}

func (e *Encoder) writePackedInt32Slice(s []int32) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*4)
		e.writePackedRawBytes(qpackKindInt32, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*4)
	out = append(out, tagPackRaw, qpackKindInt32)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU32(out, uint32(v))
	}
	e.buf = out
}

func (e *Encoder) writePackedFloat32Slice(s []float32) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*4)
		e.writePackedRawBytes(qpackKindFloat32, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*4)
	out = append(out, tagPackRaw, qpackKindFloat32)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU32(out, math.Float32bits(v))
	}
	e.buf = out
}

func (e *Encoder) writePackedFloat64Slice(s []float64) {
	n := len(s)
	if nativeLittleEndian && n > 0 {
		body := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*8)
		e.writePackedRawBytes(qpackKindFloat64, n, body)
		return
	}
	e.writeHeader()
	out := slices.Grow(e.buf, 2+10+n*8)
	out = append(out, tagPackRaw, qpackKindFloat64)
	out = appendUvarint(out, uint64(n))
	for _, v := range s {
		out = appendU64(out, math.Float64bits(v))
	}
	e.buf = out
}

// readPackedRawHeader consumes the kind byte and varuint length that
// follow a tagPackRaw tag (the tag itself must already be consumed). It
// returns the element count and a slice aliasing the LE payload. The
// caller's expected kind must match the on-wire kind.
func (d *Decoder) readPackedRawHeader(expectKind byte) (int, []byte, error) {
	if d.i >= len(d.buf) {
		return 0, nil, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, nil, ErrTypeMismatch
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, nil, ErrInvalidLength
	}
	d.i += nr
	w := qpackRawWidthBytes(k)
	if w == 0 {
		return 0, nil, ErrBadTag
	}
	if n64 > uint64(len(d.buf)-d.i)/uint64(w) {
		return 0, nil, ErrShortBuffer
	}
	n := int(n64)
	nBytes := n * w
	body := d.buf[d.i : d.i+nBytes]
	d.i += nBytes
	return n, body, nil
}

func (d *Decoder) readPackedUint64Slice() ([]uint64, error) {
	n, body, err := d.readPackedRawHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = readU64(body[i*8:])
	}
	return out, nil
}

func (d *Decoder) readPackedInt64Slice() ([]int64, error) {
	n, body, err := d.readPackedRawHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = int64(readU64(body[i*8:]))
	}
	return out, nil
}

func (d *Decoder) readPackedUint32Slice() ([]uint32, error) {
	n, body, err := d.readPackedRawHeader(qpackKindUint32)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*4)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = readU32(body[i*4:])
	}
	return out, nil
}

func (d *Decoder) readPackedInt32Slice() ([]int32, error) {
	n, body, err := d.readPackedRawHeader(qpackKindInt32)
	if err != nil {
		return nil, err
	}
	out := make([]int32, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*4)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = int32(readU32(body[i*4:]))
	}
	return out, nil
}

func (d *Decoder) readPackedFloat32Slice() ([]float32, error) {
	n, body, err := d.readPackedRawHeader(qpackKindFloat32)
	if err != nil {
		return nil, err
	}
	out := make([]float32, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*4)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = math.Float32frombits(readU32(body[i*4:]))
	}
	return out, nil
}

func (d *Decoder) readPackedFloat64Slice() ([]float64, error) {
	n, body, err := d.readPackedRawHeader(qpackKindFloat64)
	if err != nil {
		return nil, err
	}
	out := make([]float64, n)
	if n == 0 {
		return out, nil
	}
	if nativeLittleEndian {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), n*8)
		copy(dst, body)
		return out, nil
	}
	for i := range n {
		out[i] = math.Float64frombits(readU64(body[i*8:]))
	}
	return out, nil
}

// readPackedBool decodes a bool slice written by writePackedBool. The tag
// byte must already have been consumed.
func (d *Decoder) readPackedBool() ([]bool, error) {
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if n64 > uint64(len(d.buf)-d.i)*8 {
		// Length is in elements, payload is in bytes. Reject claims that
		// cannot possibly fit even if every byte carried 8 valid bits.
		return nil, ErrShortBuffer
	}
	n := int(n64)
	nBytes := (n + 7) >> 3
	if d.i+nBytes > len(d.buf) {
		return nil, ErrShortBuffer
	}
	out := make([]bool, n)
	base := d.i
	i := 0
	for ; i+8 <= n; i += 8 {
		b := d.buf[base+(i>>3)]
		out[i] = b&(1<<0) != 0
		out[i+1] = b&(1<<1) != 0
		out[i+2] = b&(1<<2) != 0
		out[i+3] = b&(1<<3) != 0
		out[i+4] = b&(1<<4) != 0
		out[i+5] = b&(1<<5) != 0
		out[i+6] = b&(1<<6) != 0
		out[i+7] = b&(1<<7) != 0
	}
	for ; i < n; i++ {
		out[i] = d.buf[base+(i>>3)]&(1<<uint(i&7)) != 0
	}
	d.i += nBytes
	return out, nil
}
