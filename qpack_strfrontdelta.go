package qdf

import (
	"encoding/binary"
	"math/bits"
	"slices"

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
// pushdown materializes only surviving rows and must keep that property.
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
	// High-cardinality gate first, the same one the alphabet packer uses. A
	// low-cardinality column is cheaper as interned references — a repeat costs
	// one state-ref byte — and this codec cannot beat that: it still spends
	// three varints per row. Scoring against the raw per-value floor instead
	// would say otherwise, because raw is not what such a column would actually
	// get. Without this gate an all-identical column is claimed here and comes
	// out larger than the const form it belongs in.
	if !dictSampleHighCard(strs) {
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

// tryWriteStringColumnFrontDelta attempts to emit strs as a tagColStrFrontDelta
// block. It returns true when the block was written and is strictly smaller
// than the per-value raw floor, false when the caller should fall through to
// the next codec — in which case the encoder's buffer is untouched.
//
// The gate runs twice on purpose. frontDeltaProject decides from a sample
// whether to bother building the streams at all; the exact totals are then
// compared before a byte is emitted, so a column whose tail behaves unlike its
// head cannot make the wire grow.
func (e *Encoder) tryWriteStringColumnFrontDelta(strs []string) bool {
	n := len(strs)
	mode, ok := frontDeltaProject(strs)
	if !ok {
		return false
	}
	withSuffix := mode == frontDeltaFrontBack

	// Build the length stream and measure the body exactly. The staging lives
	// on the encoder so a wide batch does not reallocate it per column.
	lens := e.frontDeltaLens[:0]
	totalMid, rawFloor, lenBytes := 0, 0, 0
	prev := ""
	for i, s := range strs {
		rawFloor += uvarintLen(uint64(len(s))) + len(s)
		if i%frontDeltaBlock == 0 {
			prev = ""
		}
		p := frontDeltaCommonPrefix(s, prev)
		q := 0
		if withSuffix {
			q = frontDeltaCommonSuffix(s, prev, p)
		}
		m := len(s) - p - q
		lens = append(lens, uint32(p), uint32(q), uint32(m))
		totalMid += m
		// Measure the length stream here rather than in a second walk over
		// lens: the varint widths are known the moment the values are, and a
		// separate pass would re-read 12 bytes per row for nothing.
		lenBytes += uvarintLen(uint64(p)) + uvarintLen(uint64(m))
		if withSuffix {
			lenBytes += uvarintLen(uint64(q))
		}
		prev = s
	}
	e.frontDeltaLens = lens

	hdr := 2 + uvarintLen(uint64(n)) + uvarintLen(uint64(totalMid))
	if hdr+lenBytes+totalMid >= rawFloor {
		return false // exact total lost: leave the column to the next codec
	}

	e.writeHeader()
	out := slices.Grow(e.buf, hdr+lenBytes+totalMid)
	out = append(out, tagColStrFrontDelta)
	out = appendUvarint(out, uint64(n))
	var flags byte
	if withSuffix {
		flags = 1
	}
	out = append(out, flags)
	out = appendUvarint(out, uint64(totalMid))
	for i := 0; i < len(lens); i += 3 {
		out = appendUvarint(out, uint64(lens[i]))
		if withSuffix {
			out = appendUvarint(out, uint64(lens[i+1]))
		}
		out = appendUvarint(out, uint64(lens[i+2]))
	}
	for i, s := range strs {
		p, q := int(lens[3*i]), int(lens[3*i+1])
		out = append(out, s[p:len(s)-q]...)
	}
	e.buf = out
	return true
}

// readStringColumnFrontDelta decodes a tagColStrFrontDelta block. d.i points at
// the tag on entry.
//
// The length stream is walked twice: once to validate every field and total the
// reconstructed size, then once to build. Two passes are what let the slab be
// allocated exactly once, and they are also what lets every bound be checked
// before a byte is copied — a hostile stream cannot reach past the buffer or
// claim a body it did not send.
func (d *Decoder) readStringColumnFrontDelta(n int) ([]string, error) {
	if d.i >= len(d.buf) || d.buf[d.i] != tagColStrFrontDelta {
		return nil, ErrTypeMismatch
	}
	d.i++

	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if !d.colLenOK(n64) || n64 != uint64(n) {
		return nil, ErrInvalidLength
	}

	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	flags := d.buf[d.i]
	d.i++
	if flags&^1 != 0 {
		return nil, ErrBadTag // reserved bits must be zero
	}
	withSuffix := flags&1 != 0

	totalMid64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if totalMid64 > uint64(len(d.buf)-d.i) {
		return nil, ErrShortBuffer
	}
	totalMid := int(totalMid64)

	// Pass one: validate and total. lenStart lets pass two re-read the same
	// varints without storing them.
	lenStart := d.i
	total, sumMid := 0, 0
	prevLen := 0
	for i := range n {
		if i%frontDeltaBlock == 0 {
			prevLen = 0
		}
		p, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		q := uint64(0)
		if withSuffix {
			q, nr = readUvarint(d.buf[d.i:])
			if nr <= 0 {
				return nil, ErrInvalidLength
			}
			d.i += nr
		}
		m, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr

		// The two references must fit inside the previous row and must not
		// overlap: without this a row could claim more bytes than exist.
		if p+q > uint64(prevLen) {
			return nil, ErrBadTag
		}
		if m > totalMid64 {
			return nil, ErrInvalidLength
		}
		sumMid += int(m)
		if sumMid > totalMid {
			return nil, ErrInvalidLength
		}
		rowLen := int(p) + int(q) + int(m)
		total += rowLen
		// total is a SUM of reconstructed lengths, so unlike the raw column —
		// whose total is one wire value bounded by the remaining buffer — it can
		// legitimately exceed the body. That is the compression. But a row may
		// reuse its predecessor whole (p == prevLen, m == 0) for two or three
		// bytes of wire, so a crafted column adds prevLen per row indefinitely
		// and the slab below would be sized by the product of two independent
		// wire values. Bound it by the same ceiling the rest of the columnar
		// decoder uses.
		if total > maxColumnarBytes {
			return nil, ErrInvalidLength
		}
		prevLen = rowLen
	}
	if sumMid != totalMid {
		return nil, ErrInvalidLength // the declared body and the lengths disagree
	}
	if d.i+totalMid > len(d.buf) {
		return nil, ErrShortBuffer
	}
	body := d.buf[d.i : d.i+totalMid]
	bodyEnd := d.i + totalMid

	// Pass two: build. One slab holds every reconstructed value; the returned
	// strings alias it, the same shape readStringColumnAlpha uses.
	slab := make([]byte, 0, total)
	out := d.colStrScratch(n)
	d.i = lenStart
	mid := 0
	prev := ""
	for i := range n {
		if i%frontDeltaBlock == 0 {
			prev = ""
		}
		p, nr := readUvarint(d.buf[d.i:])
		d.i += nr
		q := uint64(0)
		if withSuffix {
			q, nr = readUvarint(d.buf[d.i:])
			d.i += nr
		}
		m, nr := readUvarint(d.buf[d.i:])
		d.i += nr

		off := len(slab)
		slab = append(slab, prev[:p]...)
		slab = append(slab, body[mid:mid+int(m)]...)
		slab = append(slab, prev[len(prev)-int(q):]...)
		mid += int(m)
		out[i] = unsafestr.String(slab[off:])
		prev = out[i]
	}
	d.i = bodyEnd
	return out, nil
}
