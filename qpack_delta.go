package qdf

import (
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
)

// Delta + FOR codec for integer slices.
//
// First value is stored verbatim. For i >= 1 we compute Δᵢ = aᵢ - aᵢ₋₁,
// find (minΔ, maxΔ), and bit-pack (Δᵢ - minΔ) into ceil(bits/8) per slot.
// For monotonic data (timestamps, IDs, counters) Δᵢ are tiny and clustered
// so bitsPer often collapses to 1..16 bits. For mixed-direction data the
// zigzag bias rolled into minΔ keeps the bit width compact.
//
// Wire payload (after the tag and kind byte):
//   bits           (1 byte, 0..56)
//   firstVal       (varuint for unsigned kinds; zigzag varuint for signed)
//   minDelta       (zigzag varuint, signed)
//   n              (varuint)
//   body           ceil((n-1)*bits/8) bytes when n>=2, else absent
//
// Arithmetic is performed in uint64 for the unsigned variant (wraps
// modulo 2^64, which is the inverse of the modular subtraction the
// encoder used). The signed variant promotes everything to int64.

// computeDeltaStatsU64 inspects an unsigned monotonic-or-not slice and
// returns the first value, the minimum delta (as int64), and the bits
// required to represent (Δᵢ - minΔ) for every i >= 1.
func computeDeltaStatsU64(s []uint64) (first uint64, minDelta int64, bitsPer int) {
	if len(s) == 0 {
		return 0, 0, 0
	}
	first = s[0]
	if len(s) == 1 {
		return first, 0, 0
	}
	minD := int64(s[1] - s[0])
	maxD := minD
	prev := s[1]
	for i := 2; i < len(s); i++ {
		d := int64(s[i] - prev)
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
		prev = s[i]
	}
	bitsPer = bitpack.BitsForDelta(uint64(maxD) - uint64(minD))
	return first, minD, bitsPer
}

func computeDeltaStatsI64(s []int64) (first int64, minDelta int64, bitsPer int) {
	if len(s) == 0 {
		return 0, 0, 0
	}
	first = s[0]
	if len(s) == 1 {
		return first, 0, 0
	}
	minD := s[1] - s[0]
	maxD := minD
	prev := s[1]
	for i := 2; i < len(s); i++ {
		d := s[i] - prev
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
		prev = s[i]
	}
	bitsPer = bitpack.BitsForDelta(uint64(maxD) - uint64(minD))
	return first, minD, bitsPer
}

// writePackedDeltaForUint64Slice emits the delta+FOR codec for s. Caller
// must have called computeDeltaStatsU64 already to pick first/minDelta/
// bitsPer; bitsPer must be <= qpackForMaxBits.
func (e *Encoder) writePackedDeltaForUint64Slice(s []uint64, first uint64, minDelta int64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyN := 0
	if n >= 2 {
		bodyN = ((n - 1) * bitsPer) >> 3
		if ((n-1)*bitsPer)&7 != 0 {
			bodyN++
		}
	}
	out := slices.Grow(e.buf, 3+10+10+10+bodyN)
	out = append(out, tagPackDeltaFor, qpackKindUint64, byte(bitsPer))
	out = appendUvarint(out, first)
	out = appendUvarint(out, zigzagEncode64(minDelta))
	out = appendUvarint(out, uint64(n))
	if n < 2 || bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyN]
	body := out[start : start+bodyN]
	clear(body)
	minU := uint64(minDelta)
	var chunk [64]uint64
	prev := s[0]
	wrote := 0
	for i := 1; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		k := 0
		for j := i; j < end; j++ {
			d := s[j] - prev
			chunk[k] = d - minU
			prev = s[j]
			k++
		}
		bitpack.PackChunk(body, chunk[:k], bitsPer, wrote)
		wrote += k
	}
	e.buf = out
}

func (e *Encoder) writePackedDeltaForInt64Slice(s []int64, first int64, minDelta int64, bitsPer int) {
	e.writeHeader()
	n := len(s)
	bodyN := 0
	if n >= 2 {
		bodyN = ((n - 1) * bitsPer) >> 3
		if ((n-1)*bitsPer)&7 != 0 {
			bodyN++
		}
	}
	out := slices.Grow(e.buf, 3+10+10+10+bodyN)
	out = append(out, tagPackDeltaFor, qpackKindInt64, byte(bitsPer))
	out = appendUvarint(out, zigzagEncode64(first))
	out = appendUvarint(out, zigzagEncode64(minDelta))
	out = appendUvarint(out, uint64(n))
	if n < 2 || bitsPer == 0 {
		e.buf = out
		return
	}
	start := len(out)
	out = out[:start+bodyN]
	body := out[start : start+bodyN]
	clear(body)
	minU := uint64(minDelta)
	var chunk [64]uint64
	prev := s[0]
	wrote := 0
	for i := 1; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		k := 0
		for j := i; j < end; j++ {
			d := s[j] - prev
			chunk[k] = uint64(d) - minU
			prev = s[j]
			k++
		}
		bitpack.PackChunk(body, chunk[:k], bitsPer, wrote)
		wrote += k
	}
	e.buf = out
}

// readPackedDeltaForHeader consumes the kind, bits, firstVal, minDelta,
// count fields after a tagPackDeltaFor tag (the tag itself already
// consumed). signedFirst is set for signed kinds; unsignedFirst for
// unsigned kinds.
func (d *Decoder) readPackedDeltaForHeader(expectKind byte) (bitsPer int, unsignedFirst uint64, signedFirst int64, minDelta int64, n int, body []byte, err error) {
	if d.i+2 > len(d.buf) {
		return 0, 0, 0, 0, 0, nil, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, 0, 0, 0, 0, nil, ErrTypeMismatch
	}
	bp := d.buf[d.i]
	d.i++
	if int(bp) > qpackForMaxBits {
		return 0, 0, 0, 0, 0, nil, ErrBadTag
	}
	bitsPer = int(bp)
	first64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	if k&(1<<2) != 0 {
		signedFirst = zigzagDecode64(first64)
	} else {
		unsignedFirst = first64
	}
	mn64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	minDelta = zigzagDecode64(mn64)
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, 0, 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) {
		return 0, 0, 0, 0, 0, nil, ErrInvalidLength
	}
	if n64 < 2 {
		return bitsPer, unsignedFirst, signedFirst, minDelta, int(n64), nil, nil
	}
	// Validate in uint64 before the signed cast; the same body-shape
	// overflow that produced a reverse-range panic in readPackedForHeader
	// applies here.
	rem := uint64(len(d.buf) - d.i)
	if bitsPer > 0 && (n64-1) > rem*8/uint64(bitsPer) {
		return 0, 0, 0, 0, 0, nil, ErrShortBuffer
	}
	n = int(n64)
	bodyBytes := int(((n64-1)*uint64(bitsPer) + 7) / 8)
	body = d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	return bitsPer, unsignedFirst, signedFirst, minDelta, n, body, nil
}

func (d *Decoder) readPackedDeltaForUint64Slice() ([]uint64, error) {
	bitsPer, first, _, minDelta, n, body, err := d.readPackedDeltaForHeader(qpackKindUint64)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	if n == 0 {
		return out, nil
	}
	out[0] = first
	if n == 1 {
		return out, nil
	}
	if bitsPer == 0 {
		step := uint64(minDelta)
		v := first
		for i := 1; i < n; i++ {
			v += step
			out[i] = v
		}
		return out, nil
	}
	if cap(d.deltaScratch) < n-1 {
		d.deltaScratch = make([]uint64, n-1)
	}
	tmp := d.deltaScratch[:n-1]
	bitpack.Unpack(tmp, body, bitsPer)
	minU := uint64(minDelta)
	v := first
	for i, dv := range tmp {
		v += dv + minU
		out[i+1] = v
	}
	return out, nil
}

func (d *Decoder) readPackedDeltaForInt64Slice() ([]int64, error) {
	bitsPer, _, first, minDelta, n, body, err := d.readPackedDeltaForHeader(qpackKindInt64)
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	if n == 0 {
		return out, nil
	}
	out[0] = first
	if n == 1 {
		return out, nil
	}
	if bitsPer == 0 {
		v := first
		for i := 1; i < n; i++ {
			v += minDelta
			out[i] = v
		}
		return out, nil
	}
	if cap(d.deltaScratch) < n-1 {
		d.deltaScratch = make([]uint64, n-1)
	}
	tmp := d.deltaScratch[:n-1]
	bitpack.Unpack(tmp, body, bitsPer)
	minU := uint64(minDelta)
	v := uint64(first)
	for i, dv := range tmp {
		v += dv + minU
		out[i+1] = int64(v)
	}
	return out, nil
}
