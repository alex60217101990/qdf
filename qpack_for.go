package qdf

import (
	"math"
	"slices"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/bitpack"
)

// Frame-of-Reference (FOR) bit-packing for integer slices.
//
// For a slice with min and max values, every element fits in
// ceil(log2(max-min+1)) bits once min has been subtracted. The encoded
// payload is one min reference plus a tightly packed bit-stream of the
// deltas, LSB-first within each byte. With clustered integer data this
// crushes the per-element cost from 64 bits to anywhere from 0 (all
// equal) to ~16-24 bits without measurably slowing the decoder.
//
// The packer caps bits at 56 so the running 64-bit accumulator never
// overflows during the partial-byte drain (worst case: 7 carry-over bits
// + 56 new bits = 63 < 64). Slices that genuinely need 57-64 bits per
// delta should pick the raw-LE codec, which strictly beats FOR there. The
// pure bit-packing kernels live in internal/bitpack.

const qpackForMaxBits = 56

// qpackMaxStandaloneCount caps the element count a constant-value codec
// (bitsPer == 0) may claim outside a columnar column. Such codecs carry an
// empty body, so the per-element byte bound that guards the bitsPer > 0 path
// does not exist; without this ceiling a ~14-byte header could drive a
// multi-GB make(). Columnar columns are already bounded by colLenOK; this
// mirrors the standalone cap RLE applies (qpack_rle.go).
//
// Pinned to maxColumnarElems (1<<24, 128 MiB for an int64 slice) — the same
// ceiling columnar already enforces for a constant column. The previous 1<<30
// allowed an 8 GiB allocation from a tiny hostile header (OOM-DoS); a constant
// slice above 16M elements is not a real workload and callers that large must
// stream/shard regardless.
const qpackMaxStandaloneCount = maxColumnarElems

// qpackForSizeUnsigned estimates the wire size, in bytes, of a FOR-packed
// encoding for n unsigned values whose delta range needs bits per slot
// and whose minimum value is m. Used to choose between raw and FOR.
func qpackForSizeUnsigned(n, bitsPer int, m uint64) int {
	// tag(1) + kind(1) + bits(1) + min varuint (1..10) + count varuint (1..5 for n<=2^28)
	// + ceil(n*bits/8) body.
	hdr := 3 + uvarintLen(m) + uvarintLen(uint64(n))
	body := (n*bitsPer + 7) >> 3
	return hdr + body
}

func qpackForSizeSigned(n, bitsPer int, m int64) int {
	hdr := 3 + uvarintLen(zigzagEncode64(m)) + uvarintLen(uint64(n))
	body := (n*bitsPer + 7) >> 3
	return hdr + body
}

// minMaxU64 returns the (min, max) of s. Caller must ensure len(s) > 0.
func minMaxU64(s []uint64) (uint64, uint64) {
	mn, mx := s[0], s[0]
	for _, v := range s[1:] {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

func minMaxI64(s []int64) (int64, int64) {
	mn, mx := s[0], s[0]
	for _, v := range s[1:] {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

// writePackedForUint64Slice emits a FOR-packed []uint64. Caller should
// already have chosen FOR over raw (e.g. via qpackForSizeUnsigned <
// raw-size). bitsPer must be in [0, 56]. min must equal the slice's min.
func (e *Encoder) writePackedForUint64Slice(s []uint64, mn uint64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyBytes := (n*bitsPer + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackFor, qpackKindUint64, byte(bitsPer))
	out = appendUvarint(out, mn)
	out = appendUvarint(out, uint64(n))
	if bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	// Subtract min on the fly to avoid an extra allocation.
	// Reuse a small stack-allocated chunk buffer for the delta values.
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = v - mn
		}
		bitpack.PackChunk(body, chunk[:end-i], bitsPer, i)
	}
	e.buf = out
}

// writePackedForInt64Slice mirrors writePackedForUint64Slice with a
// signed min stored as zigzag varuint.
func (e *Encoder) writePackedForInt64Slice(s []int64, mn int64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyBytes := (n*bitsPer + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackFor, qpackKindInt64, byte(bitsPer))
	out = appendUvarint(out, zigzagEncode64(mn))
	out = appendUvarint(out, uint64(n))
	if bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	clear(body)
	mnU := uint64(mn)
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = uint64(v) - mnU
		}
		bitpack.PackChunk(body, chunk[:end-i], bitsPer, i)
	}
	e.buf = out
}

// readPackedForHeader consumes the kind, bits, min, count fields after a
// tagPackFor tag. The tag itself must already have been consumed. The
// returned signedMin is meaningful only when the kind family is signed.
func (d *Decoder) readPackedForHeader(expectKind byte) (bitsPer int, unsignedMin uint64, signedMin int64, n int, body []byte, err error) {
	if d.i+2 > len(d.buf) {
		return 0, 0, 0, 0, nil, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, 0, 0, 0, nil, ErrTypeMismatch
	}
	b := d.buf[d.i]
	d.i++
	if int(b) > qpackForMaxBits {
		return 0, 0, 0, 0, nil, ErrBadTag
	}
	bitsPer = int(b)
	mn64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	if k&(1<<2) != 0 {
		signedMin = zigzagDecode64(mn64)
	} else {
		unsignedMin = mn64
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	// Validate body size in uint64 BEFORE the signed cast. A hostile
	// varuint with n64 > MaxInt would otherwise wrap int(n64) to
	// negative and the bounds check in the original form
	// (`d.i + bodyBytes > len(d.buf)`) accepts a negative bodyBytes,
	// then `d.buf[d.i : d.i+bodyBytes]` panics with a reverse-range
	// slice. Same shape as the Skip-path overflow fixed earlier;
	// applies to the body-reading path here too.
	rem := uint64(len(d.buf) - d.i)
	if bitsPer > 0 {
		if n64 > rem*8/uint64(bitsPer) {
			return 0, 0, 0, 0, nil, ErrShortBuffer
		}
	} else if n64 > qpackMaxStandaloneCount {
		// bitsPer == 0 (constant slice): empty body, so the per-element
		// bound above does not apply. Cap an implausible standalone count.
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	if n64 > uint64(math.MaxInt) { // 32-bit: rem*8 lets n64 exceed MaxInt -> int(n64) wraps negative -> make panics
		return 0, 0, 0, 0, nil, ErrInvalidLength
	}
	n = int(n64)
	bodyBytes := int((n64*uint64(bitsPer) + 7) / 8)
	body = d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	return bitsPer, unsignedMin, signedMin, n, body, nil
}

func (d *Decoder) readPackedForUint64Slice() ([]uint64, error) {
	bitsPer, mn, _, n, body, err := d.readPackedForHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return out, nil
	}
	bitpack.Unpack(out, body, bitsPer)
	if mn != 0 {
		for i := range out {
			out[i] += mn
		}
	}
	return out, nil
}

func (d *Decoder) readPackedForInt64Slice() ([]int64, error) {
	bitsPer, _, mn, n, body, err := d.readPackedForHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	if bitsPer == 0 {
		for i := range out {
			out[i] = mn
		}
		return out, nil
	}
	// int64 and uint64 share a layout, so decode straight into out viewed
	// as []uint64 and add the reference in place — no separate scratch
	// buffer or copy pass (mirrors the unsigned reader above).
	u := unsafe.Slice((*uint64)(unsafe.Pointer(unsafe.SliceData(out))), n)
	bitpack.Unpack(u, body, bitsPer)
	if mnU := uint64(mn); mnU != 0 {
		for i := range u {
			u[i] += mnU
		}
	}
	return out, nil
}
