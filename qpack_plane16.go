package qdf

import (
	"math"
	"slices"

	"github.com/alex60217101990/qdf/internal/tans"
)

// Byte-plane codec for []uint16 columns whose high byte carries far less
// entropy than the low one — the shape of bf16/fp16 weight tensors, quantized
// activations, and any 16-bit column clustered around a few magnitudes.
//
// The values are de-interleaved into a high-byte plane and a low-byte plane,
// and ONLY the high plane is entropy-coded (its own tANS frequency table).
// The split matters because order-0 entropy is order-independent: running one
// shared table over the interleaved stream, or over the concatenated planes,
// scores the same (measured 1.271x on bf16 weights either way). A table fitted
// to the high plane alone is what pays — measured 1.49x on bf16, 1.18x on
// fp16, against an order-0 floor of 1.53x / 1.21x.
//
// The low plane is stored raw on purpose: at ~8 bits of entropy it is
// incompressible, and a tANS pass over it costs a 256-entry table for nothing.
//
// Wire (kind qpackKindPlane16 under tagPackRaw):
//
//	tag, kind, varuint(n), varuint(len(hiBody)), hiBody, n raw low bytes
//
// hiBody is always a tANS blob strictly shorter than the n-byte high plane:
// when entropy coding does not shrink it, plane16Estimate declines and the
// caller emits the native 2 B/elem column instead. The decoder therefore
// rejects hiLen >= n rather than treating it as a raw plane — no flag byte is
// spent, and a hostile stream cannot smuggle in arbitrary high bytes that
// would decode silently. The caller only emits this codec when the total
// beats the native column, so it is never larger.

// plane16MinElems is the smallest column worth the codec: below it the tANS
// frequency table (up to 512 B) dwarfs any saving.
const plane16MinElems = 1024

// plane16Sample bounds the projection's histogram pass.
const plane16Sample = 2048

// plane16Split de-interleaves s into the high and low byte planes of dst,
// which must have room for 2*len(s) bytes. Returns the two plane slices.
func plane16Split(dst []byte, s []uint16) (hi, lo []byte) {
	n := len(s)
	hi, lo = dst[:n], dst[n:2*n]
	for i, v := range s {
		hi[i] = byte(v >> 8)
		lo[i] = byte(v)
	}
	return hi, lo
}

// plane16Project estimates the byte-plane size from a sampled high-plane
// histogram: order-0 entropy of the high byte plus one raw byte per value.
// This is the cheap gate — a full trial compression only runs when this
// projection beats the alternatives, so columns the codec cannot help never
// pay for a wasted tANS pass.
func plane16Project(s []uint16) int {
	n := len(s)
	stride := 1
	if n > plane16Sample {
		stride = n / plane16Sample
	}
	var hist [256]int32
	sampled := 0
	for i := 0; i < n && sampled < 2*plane16Sample; i += stride {
		hist[s[i]>>8]++
		sampled++
	}
	// Sum -p*log2(p) over the sampled high bytes.
	var bits float64
	inv := 1 / float64(sampled)
	for _, c := range hist {
		if c == 0 {
			continue
		}
		p := float64(c) * inv
		bits -= float64(p * math.Log2(p)) // explicit conversion forbids FMA fusion: the gate must not be arch-dependent
	}
	// Per value: entropy-coded high byte + raw low byte. The +512 covers the
	// tANS frequency table; the projection is deliberately optimistic, the
	// real trial below is what decides.
	return int(bits*float64(n)/8) + n + 512
}

// plane16Estimate returns the encoded size of the byte-plane form and the
// entropy-coded high plane, or ok=false when the codec does not apply. The
// high plane is compressed for real (there is no cheaper reliable proxy for
// what tANS will do), so callers should gate this behind a size threshold.
func (e *Encoder) plane16Estimate(s []uint16) (hiBody []byte, total int, ok bool) {
	n := len(s)
	if n < plane16MinElems {
		return nil, 0, false
	}
	if cap(e.plane16Scr) < 2*n {
		e.plane16Scr = make([]byte, 2*n)
	}
	hi, _ := plane16Split(e.plane16Scr[:2*n], s)

	hiBody = tans.Encode(e.plane16Enc[:0], hi)
	if len(hiBody) >= n {
		// Entropy coding did not pay on the high plane either — the whole
		// codec collapses to the native raw column, which the caller emits.
		e.plane16Enc = hiBody[:0]
		return nil, 0, false
	}
	e.plane16Enc = hiBody
	total = 2 + uvarintLen(uint64(n)) + uvarintLen(uint64(len(hiBody))) + len(hiBody) + n
	return hiBody, total, true
}

// writePackedPlane16Slice emits the byte-plane form using a high plane already
// compressed by plane16Estimate (the low plane is re-derived from s).
func (e *Encoder) writePackedPlane16Slice(s []uint16, hiBody []byte) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+10+len(hiBody)+n)
	out = append(out, tagPackRaw, qpackKindPlane16)
	out = appendUvarint(out, uint64(n))
	out = appendUvarint(out, uint64(len(hiBody)))
	out = append(out, hiBody...)
	base := len(out)
	out = out[:base+n]
	for i, v := range s {
		out[base+i] = byte(v)
	}
	e.buf = out
}

// readPackedPlane16Slice decodes a byte-plane u16 column. d.i points at the
// kind byte.
func (d *Decoder) readPackedPlane16Slice() ([]uint16, error) {
	if d.i >= len(d.buf) || d.buf[d.i] != qpackKindPlane16 {
		return nil, ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
	// Every value costs at least its raw low byte, so n is bounded by the
	// remaining input before anything is allocated.
	if rem := uint64(len(d.buf) - d.i); n64 > rem {
		return nil, ErrShortBuffer
	}
	n := int(n64)
	if n == 0 {
		return []uint16{}, nil
	}
	hiLen64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if hiLen64 > uint64(len(d.buf)-d.i) {
		return nil, ErrShortBuffer
	}
	// A valid encoder always compresses the high plane strictly below n bytes
	// (plane16Estimate declines otherwise), so hiLen >= n is malformed.
	if hiLen64 >= n64 {
		return nil, ErrBadTag
	}
	hiLen := int(hiLen64)
	hiBody := d.buf[d.i : d.i+hiLen]
	d.i += hiLen
	if d.i+n > len(d.buf) {
		return nil, ErrShortBuffer
	}
	lo := d.buf[d.i : d.i+n]
	d.i += n

	hi, err := tans.Decode(hiBody, n)
	if err != nil {
		return nil, ErrBadTag
	}
	out := make([]uint16, n)
	for i := range out {
		out[i] = uint16(hi[i])<<8 | uint16(lo[i])
	}
	return out, nil
}

// skipPlane16 walks a byte-plane body without decoding it. d.i points at the
// kind byte.
func (d *Decoder) skipPlane16() error {
	d.i++ // kind
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if n64 == 0 {
		return nil
	}
	// Bound n by the remaining bytes before adding it to an offset: the low
	// plane alone is one byte per value.
	if n64 > uint64(len(d.buf)-d.i) {
		return ErrShortBuffer
	}
	hiLen64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if hiLen64 > uint64(len(d.buf)-d.i) {
		return ErrShortBuffer
	}
	d.i += int(hiLen64)
	if uint64(len(d.buf)-d.i) < n64 {
		return ErrShortBuffer
	}
	d.i += int(n64)
	return nil
}
