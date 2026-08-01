package qdf

import (
	"math"
	"math/bits"
	"slices"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/bitpack"
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
	// Each exception is written as uvarint(i-prev) + value; the index gap is at
	// most n, so uvarintLen(n) is a safe per-exception upper bound. Charging 1
	// byte (as before) under-counted far-apart outliers, letting PFOR be picked
	// while its real output exceeded the runner-up — a never-larger violation.
	gapLen := uvarintLen(uint64(n))
	hdr := 3 + uvarintLen(uint64(n)) + uvarintLen(mn)
	bestB, bestCost := -1, math.MaxInt
	suffix := 0 // number of values with width > cand, accumulated as cand descends
	for cand := forBits - 1; cand >= 0; cand-- {
		suffix += hist[cand+1]
		body := (n*cand + 7) >> 3
		c := hdr + body + uvarintLen(uint64(suffix)) + suffix*(gapLen+valLen)
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
	out = out[:start+bodyBytes] // capacity guaranteed by slices.Grow at line 66
	body := out[start : start+bodyBytes]
	// Zero-initialize body bytes since PackChunk may read-modify-write on byte boundaries
	clear(body)
	// Collect exceptions during the pack pass so no second O(n) scan of s is
	// needed. excPos stores pre-computed relative positions (i+j - prev); excDelta
	// stores the corresponding delta values. Both slices reuse e.pforExcPos /
	// e.pforExcDelta across calls — same lifecycle as alpScratch.
	excPos := e.pforExcPos[:0]
	excDelta := e.pforExcDelta[:0]
	prev := 0
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			d := v - mn
			chunk[j] = d & mask
			if pforIsException(d, b) {
				excPos = append(excPos, uint64(i+j-prev))
				excDelta = append(excDelta, d)
				prev = i + j
			}
		}
		bitpack.PackChunk(body, chunk[:end-i], b, i)
	}
	e.pforExcPos = excPos
	e.pforExcDelta = excDelta
	out = appendUvarint(out, uint64(len(excPos)))
	for i, rp := range excPos {
		out = appendUvarint(out, rp)
		out = appendUvarint(out, excDelta[i])
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
	gapLen := uvarintLen(uint64(n)) // uvarint(i-prev) <= uvarint(n); see pforPlanUnsigned
	hdr := 3 + uvarintLen(uint64(n)) + uvarintLen(zigzagEncode64(mn))
	bestB, bestCost := -1, math.MaxInt
	suffix := 0
	for cand := forBits - 1; cand >= 0; cand-- {
		suffix += hist[cand+1]
		body := (n*cand + 7) >> 3
		c := hdr + body + uvarintLen(uint64(suffix)) + suffix*(gapLen+valLen)
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
	out = out[:start+bodyBytes] // capacity guaranteed by slices.Grow at line 153
	body := out[start : start+bodyBytes]
	// Zero-initialize body bytes since PackChunk may read-modify-write on byte boundaries
	clear(body)
	// Collect exceptions during the pack pass (same pattern as the uint64 writer).
	excPos := e.pforExcPos[:0]
	excDelta := e.pforExcDelta[:0]
	prev := 0
	var chunk [64]uint64
	for i := 0; i < n; i += len(chunk) {
		end := min(i+len(chunk), n)
		for j, v := range s[i:end] {
			d := uint64(v) - mnU
			chunk[j] = d & mask
			if pforIsException(d, b) {
				excPos = append(excPos, uint64(i+j-prev))
				excDelta = append(excDelta, d)
				prev = i + j
			}
		}
		bitpack.PackChunk(body, chunk[:end-i], b, i)
	}
	e.pforExcPos = excPos
	e.pforExcDelta = excDelta
	out = appendUvarint(out, uint64(len(excPos)))
	for i, rp := range excPos {
		out = appendUvarint(out, rp)
		out = appendUvarint(out, excDelta[i])
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
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
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
	if b > 0 {
		if n64 > rem*8/uint64(b) {
			return nil, ErrShortBuffer
		}
	} else if n64 > qpackMaxStandaloneCount {
		// b == 0 (constant base): empty packed body, no per-element bound.
		return nil, ErrInvalidLength
	}
	if n64 > uint64(math.MaxInt) { // 32-bit: rem*8 lets n64 exceed MaxInt -> int(n64) wraps negative
		return nil, ErrInvalidLength
	}
	n := int(n64)
	bodyBytes := (n*b + 7) >> 3
	if d.i+bodyBytes > len(d.buf) {
		return nil, ErrShortBuffer
	}
	out := make([]int64, n)
	// Decode straight into out viewed as []uint64 and add the reference in
	// place — no separate scratch buffer or copy pass (mirrors the unsigned
	// reader). The exception loop below overwrites individual slots.
	u := unsafe.Slice((*uint64)(unsafe.Pointer(unsafe.SliceData(out))), n)
	if b > 0 {
		bitpack.Unpack(u, d.buf[d.i:d.i+bodyBytes], b)
	}
	d.i += bodyBytes
	if mnU != 0 {
		for k := range u {
			u[k] += mnU
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
	for i := range excN64 {
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
		if i > 0 && dp == 0 {
			return nil, ErrInvalidLength
		}
		if dp > uint64(n-pos) {
			return nil, ErrInvalidLength
		}
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
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
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
	if b > 0 {
		if n64 > rem*8/uint64(b) {
			return nil, ErrShortBuffer
		}
	} else if n64 > qpackMaxStandaloneCount {
		// b == 0 (constant base): empty packed body, no per-element bound.
		return nil, ErrInvalidLength
	}
	if n64 > uint64(math.MaxInt) { // 32-bit: rem*8 lets n64 exceed MaxInt -> int(n64) wraps negative
		return nil, ErrInvalidLength
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
		bitpack.Unpack(out, body, b)
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
	for i := range excN64 {
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
		if i > 0 && dp == 0 {
			return nil, ErrInvalidLength
		}
		if dp > uint64(n-pos) {
			return nil, ErrInvalidLength
		}
		pos += int(dp)
		if pos < 0 || pos >= n {
			return nil, ErrInvalidLength
		}
		out[pos] = mn + delta
	}
	return out, nil
}
