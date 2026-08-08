package qdf

import (
	"encoding/binary"
	"math/bits"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Front-delta string columns (tagColStrFrontDelta) for the columnar path.
//
// A high-cardinality text column — request lines, user agents, error strings —
// reaches neither the dictionary (too many distinct values) nor the alphabet
// packer (unrestricted alphabet), so today every value pays its full length.
// Its values are nonetheless close to each other: they differ in an id, a
// version, a parameter. This codec stores that difference.
//
// Per row: the length of the prefix shared with the previous row, optionally
// the length of the shared suffix, and the bytes between. Every frontDeltaBlock
// rows the chain resets and a row is written in full, so a reader can start at
// a block boundary instead of decoding the column from the top — predicate
// pushdown materialises only surviving rows and must keep that property.
//
// FSST covers the same class of data and compresses harder, but costs ~22x the
// encode (measured 242 MB/s against Balanced's 3.5 GB/s), which is why it is
// opt-in. This codec is the cheap half of that trade: one byte comparison per
// row, no table, no training pass.

// frontDeltaBlock is the anchor interval. It is part of the wire format, not a
// tuning knob: the decoder derives anchor positions from it, so a stream
// written with another value would decode as garbage.
const frontDeltaBlock = 64

// frontDeltaSampleN bounds the gate's projection pass. A column that cannot
// benefit pays this many comparisons and no more.
const frontDeltaSampleN = 256

// frontDeltaMinElems is the smallest column worth a block header.
const frontDeltaMinElems = 32

// frontDeltaMinGainShift sets the margin the projection must clear: the coded
// form must save at least 1/16 of the raw floor. A projection is an estimate
// from a prefix of the column, so a thin win is not evidence of a real one.
const frontDeltaMinGainShift = 4

// frontDeltaMode selects whether rows also share a suffix. It is decided per
// column and travels in the block's flag byte.
type frontDeltaMode uint8

const (
	frontDeltaFrontOnly frontDeltaMode = 0
	frontDeltaFrontBack frontDeltaMode = 1
)

// frontDeltaCommonPrefix returns the length of the longest byte prefix a and b
// share.
//
// Eight bytes at a time: this comparison is the codec's inner loop, one call
// per row, so it decides what the codec costs. XOR two words, and the first
// set bit marks the first differing byte — TrailingZeros64>>3 turns that into a
// byte count. Measured against the byte-at-a-time form on 4096 user-agent
// strings: 31-35µs against 95-144µs, identical results.
func frontDeltaCommonPrefix(a, b string) int {
	n := min(len(a), len(b))
	ab, bb := unsafestr.Bytes(a), unsafestr.Bytes(b)
	i := 0
	for ; i+8 <= n; i += 8 {
		if x := binary.LittleEndian.Uint64(ab[i:]) ^ binary.LittleEndian.Uint64(bb[i:]); x != 0 {
			return i + bits.TrailingZeros64(x)>>3
		}
	}
	for ; i < n; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

// frontDeltaCommonSuffix returns the length of the longest byte suffix a and b
// share without either side reaching back into the first skip bytes, which the
// prefix has already claimed. Without that bound a value like "aaaa" against
// "aaaa" would report a prefix of 4 and a suffix of 4 and the decoder would
// reconstruct eight bytes from four.
//
// Word-wise from the far end, mirroring the prefix scan. BigEndian is what
// makes it work: it puts the LAST byte of the loaded word in the low bits, so
// TrailingZeros64>>3 again counts matching bytes — this time from the end.
func frontDeltaCommonSuffix(a, b string, skip int) int {
	n := min(len(a), len(b)) - skip
	if n <= 0 {
		return 0
	}
	ab, bb := unsafestr.Bytes(a), unsafestr.Bytes(b)
	i := 0
	for ; i+8 <= n; i += 8 {
		x := binary.BigEndian.Uint64(ab[len(a)-i-8:]) ^ binary.BigEndian.Uint64(bb[len(b)-i-8:])
		if x != 0 {
			return i + bits.TrailingZeros64(x)>>3
		}
	}
	for ; i < n; i++ {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			break
		}
	}
	return i
}

// frontDeltaProject decides whether the codec pays and in which mode, from a
// bounded prefix of the column. It is the cheap gate in front of the real
// encode: a column with nothing to share falls through with its wire unchanged
// after at most frontDeltaSampleN comparisons.
//
// Both modes are scored in the same pass, from the same comparison, and the
// winner is the one with fewer bytes — the shape alpScoreExp already uses to
// choose between two ALP reconstructions.
func frontDeltaProject(strs []string) (frontDeltaMode, bool) {
	n := len(strs)
	if n < frontDeltaMinElems {
		return 0, false
	}
	sample := min(n, frontDeltaSampleN)

	raw, front, back := 0, 0, 0
	prev := ""
	for i := range sample {
		s := strs[i]
		raw += uvarintLen(uint64(len(s))) + len(s)

		if i%frontDeltaBlock == 0 {
			prev = "" // anchor: nothing to share with
		}
		p := frontDeltaCommonPrefix(s, prev)
		q := frontDeltaCommonSuffix(s, prev, p)

		front += uvarintLen(uint64(p)) + uvarintLen(uint64(len(s)-p)) + (len(s) - p)
		back += uvarintLen(uint64(p)) + uvarintLen(uint64(q)) +
			uvarintLen(uint64(len(s)-p-q)) + (len(s) - p - q)
		prev = s
	}

	best, mode := front, frontDeltaFrontOnly
	if back < best {
		best, mode = back, frontDeltaFrontBack
	}
	// Require a real margin, not a byte or two: the sample is an estimate.
	return mode, best+raw>>frontDeltaMinGainShift < raw
}
