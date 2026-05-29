package qdf

import (
	"math/bits"
	"slices"
)

// Patched Frame-of-Reference. Packs (v-min) at a reduced width b chosen so
// that the few values that don't fit (outliers) are cheaper to store in an
// exception list than to widen every slot. Selected by the picker only when
// strictly smaller than every other codec, so it never regresses.

// pforPlanUnsigned scans s (min already known, forBits = plain-FOR width) and
// returns the bit width b that minimizes the PFOR wire cost, that cost, and
// whether PFOR is a sensible candidate (ok=false for n<8 or forBits<2, where
// there is no room to patch). The exception value byte cost uses maxDelta as a
// conservative UPPER bound, so PFOR is never chosen when it would be larger.
func pforPlanUnsigned(s []uint64, mn uint64, forBits int) (b int, cost int, ok bool) {
	n := len(s)
	if n < 8 || forBits < 2 || forBits > qpackForMaxBits {
		return 0, 0, false
	}
	var hist [65]int
	var maxDelta uint64
	for _, v := range s {
		d := v - mn
		hist[bits.Len64(d)]++
		if d > maxDelta {
			maxDelta = d
		}
	}
	valLen := uvarintLen(maxDelta) // conservative upper bound per exception value
	hdr := 3 + uvarintLen(uint64(n)) + uvarintLen(mn)
	bestB, bestCost := -1, int(^uint(0)>>1)
	suffix := 0 // number of values with width > cand, accumulated as cand descends
	for cand := forBits - 1; cand >= 0; cand-- {
		suffix += hist[cand+1]
		body := (n*cand + 7) >> 3
		c := hdr + body + uvarintLen(uint64(suffix)) + suffix*(1+valLen)
		if c < bestCost {
			bestCost, bestB = c, cand
		}
	}
	if bestB < 0 {
		return 0, 0, false
	}
	return bestB, bestCost, true
}

// writePackedPForUint64Slice emits s as a tagPackPFor payload at width b.
// mn must equal the slice min; b must come from pforPlanUnsigned.
func (e *Encoder) writePackedPForUint64Slice(s []uint64, mn uint64, b int) {
	e.writeHeader()
	n := len(s)
	mask := uint64(1)<<uint(b) - 1
	bodyBytes := (n*b + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackPFor, qpackKindUint64)
	out = appendUvarint(out, uint64(n))
	out = append(out, byte(b))
	out = appendUvarint(out, mn)
	start := len(out)
	out = append(out, make([]byte, bodyBytes)...)
	body := out[start : start+bodyBytes]
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = (v - mn) & mask
		}
		bitPackChunkInto(body, chunk[:end-i], b, i)
	}
	excN := 0
	for _, v := range s {
		if pforIsException(v-mn, b) {
			excN++
		}
	}
	out = appendUvarint(out, uint64(excN))
	prev := 0
	for i, v := range s {
		d := v - mn
		if pforIsException(d, b) {
			out = appendUvarint(out, uint64(i-prev))
			out = appendUvarint(out, d)
			prev = i
		}
	}
	e.buf = out
}

// pforIsException reports whether delta d does not fit in b bits (and so must
// be patched). d==0 never needs patching (it fits any width, including b==0).
func pforIsException(d uint64, b int) bool {
	if d == 0 {
		return false
	}
	return b == 0 || d>>uint(b) != 0
}

// pforPlanSigned mirrors pforPlanUnsigned for int64 (deltas computed in
// wrapping uint64 space relative to min, min stored zigzag).
func pforPlanSigned(s []int64, mn int64, forBits int) (b int, cost int, ok bool) {
	n := len(s)
	if n < 8 || forBits < 2 || forBits > qpackForMaxBits {
		return 0, 0, false
	}
	mnU := uint64(mn)
	var hist [65]int
	var maxDelta uint64
	for _, v := range s {
		d := uint64(v) - mnU
		hist[bits.Len64(d)]++
		if d > maxDelta {
			maxDelta = d
		}
	}
	valLen := uvarintLen(maxDelta)
	hdr := 3 + uvarintLen(uint64(n)) + uvarintLen(zigzagEncode64(mn))
	bestB, bestCost := -1, int(^uint(0)>>1)
	suffix := 0
	for cand := forBits - 1; cand >= 0; cand-- {
		suffix += hist[cand+1]
		body := (n*cand + 7) >> 3
		c := hdr + body + uvarintLen(uint64(suffix)) + suffix*(1+valLen)
		if c < bestCost {
			bestCost, bestB = c, cand
		}
	}
	if bestB < 0 {
		return 0, 0, false
	}
	return bestB, bestCost, true
}

// writePackedPForInt64Slice emits s as a tagPackPFor payload at width b.
// mn must equal the slice min; b must come from pforPlanSigned.
func (e *Encoder) writePackedPForInt64Slice(s []int64, mn int64, b int) {
	e.writeHeader()
	n := len(s)
	mnU := uint64(mn)
	mask := uint64(1)<<uint(b) - 1
	bodyBytes := (n*b + 7) >> 3
	out := slices.Grow(e.buf, 3+10+10+bodyBytes)
	out = append(out, tagPackPFor, qpackKindInt64)
	out = appendUvarint(out, uint64(n))
	out = append(out, byte(b))
	out = appendUvarint(out, zigzagEncode64(mn))
	start := len(out)
	out = append(out, make([]byte, bodyBytes)...)
	body := out[start : start+bodyBytes]
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			chunk[j] = (uint64(v) - mnU) & mask
		}
		bitPackChunkInto(body, chunk[:end-i], b, i)
	}
	excN := 0
	for _, v := range s {
		if pforIsException(uint64(v)-mnU, b) {
			excN++
		}
	}
	out = appendUvarint(out, uint64(excN))
	prev := 0
	for i, v := range s {
		d := uint64(v) - mnU
		if pforIsException(d, b) {
			out = appendUvarint(out, uint64(i-prev))
			out = appendUvarint(out, d)
			prev = i
		}
	}
	e.buf = out
}

// readPackedPForInt64Slice decodes a tagPackPFor int64 payload (tag already
// consumed) into a []int64.
func (d *Decoder) readPackedPForInt64Slice() ([]int64, error) {
	if d.i+1 > len(d.buf) || d.buf[d.i] != qpackKindInt64 {
		return nil, ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	b := int(d.buf[d.i])
	d.i++
	if b > qpackForMaxBits {
		return nil, ErrBadTag
	}
	mz, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	mnU := uint64(zigzagDecode64(mz))
	rem := uint64(len(d.buf) - d.i)
	if b > 0 && n64 > rem*8/uint64(b) {
		return nil, ErrShortBuffer
	}
	n := int(n64)
	bodyBytes := (n*b + 7) >> 3
	if d.i+bodyBytes > len(d.buf) {
		return nil, ErrShortBuffer
	}
	tmp := make([]uint64, n)
	if b > 0 {
		bitUnpackU64LE(tmp, d.buf[d.i:d.i+bodyBytes], b)
	}
	d.i += bodyBytes
	out := make([]int64, n)
	for k := range out {
		out[k] = int64(mnU + tmp[k])
	}
	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if excN64 > n64 {
		return nil, ErrInvalidLength
	}
	pos := 0
	for j := uint64(0); j < excN64; j++ {
		dp, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		delta, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		pos += int(dp)
		if pos < 0 || pos >= n {
			return nil, ErrInvalidLength
		}
		out[pos] = int64(mnU + delta)
	}
	return out, nil
}

// readPackedPForUint64Slice decodes a tagPackPFor payload (tag already
// consumed) into a []uint64.
func (d *Decoder) readPackedPForUint64Slice() ([]uint64, error) {
	if d.i+1 > len(d.buf) {
		return nil, ErrShortBuffer
	}
	if d.buf[d.i] != qpackKindUint64 {
		return nil, ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	b := int(d.buf[d.i])
	d.i++
	if b > qpackForMaxBits {
		return nil, ErrBadTag
	}
	mn, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	rem := uint64(len(d.buf) - d.i)
	if b > 0 && n64 > rem*8/uint64(b) {
		return nil, ErrShortBuffer
	}
	n := int(n64)
	bodyBytes := (n*b + 7) >> 3
	if d.i+bodyBytes > len(d.buf) {
		return nil, ErrShortBuffer
	}
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	out := make([]uint64, n)
	if b > 0 {
		bitUnpackU64LE(out, body, b)
	}
	if mn != 0 {
		for k := range out {
			out[k] += mn
		}
	}
	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if excN64 > n64 {
		return nil, ErrInvalidLength
	}
	pos := 0
	for j := uint64(0); j < excN64; j++ {
		dp, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		delta, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		pos += int(dp)
		if pos < 0 || pos >= n {
			return nil, ErrInvalidLength
		}
		out[pos] = mn + delta
	}
	return out, nil
}
