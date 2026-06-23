package qdf

import (
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Alphabet-aware bit-packed string columns (tagColStrAlpha) for the columnar
// container. See wire.go for the format and rationale. This is the high-card,
// restricted-alphabet counterpart to tagColStrRaw: it targets ID columns whose
// bytes are all drawn from a small alphabet (hex/base32/base64/decimal — trace,
// span, request IDs, hashes, GUIDs), a class dict, front-coding and FSST all
// miss. Each character costs ceil(log2 |A|) bits instead of 8.

// qpackStrAlphaMaxAlphabet caps the alphabet size at 64: |A| <= 64 keeps the
// per-char code at <= 6 bits (a strict win over 8), and the one-pass scan bails
// the instant a 65th distinct byte appears, so a full-alphabet (text) column is
// rejected after reading only its first ~dozens of bytes — no full scan, no
// regression on the common non-ID column.
const qpackStrAlphaMaxAlphabet = 64

// alphaMinDistinctPct is the high-cardinality floor: alpha-packing only fires
// when at least this percentage of a leading sample is distinct. Low-card
// columns (constants, enums) are far cheaper as const/dict/interned references,
// and alpha's raw-floor gate alone would not catch that (a constant column
// packs smaller than raw yet far larger than a single interned value). Mirrors
// the dict probe's high-cardinality threshold.
const alphaMinDistinctPct = 70

// alphaProbeMinAvgLen is the minimum average value length (bytes) for the
// intern-aware columnar probe to credit alpha-packing. Short tokens pack no
// better than a dictionary once the per-row length prefix is paid, so crediting
// them could flip a borderline struct into columnar for no real gain. The ID
// columns alpha targets (hex/uuid/base32 — 16+ chars) clear this comfortably.
const alphaProbeMinAvgLen = 8

// tryWriteStringColumnAlpha attempts to emit strs as a tagColStrAlpha block.
// It returns true when the alphabet-packed form was written (and is strictly
// smaller than the raw per-value floor), false when the caller should fall
// through to the next codec. The alphabet scan is a single pass that bails the
// moment the alphabet exceeds 64 distinct bytes, so non-restricted columns pay
// only a short prefix scan before falling through unchanged.
func (e *Encoder) tryWriteStringColumnAlpha(strs []string) bool {
	n := len(strs)
	if n < qpackStrDictMinElems {
		return false
	}

	// Cheap high-cardinality gate first (shared with the dict pre-bail): a low-card
	// column (constant, enum, run-heavy) is cheaper as const/dict/interned
	// references than as packed chars, and the raw-floor gate below would not catch
	// that — so bail before the full alphabet scan. alpha wants HIGH cardinality,
	// the inverse of the dict bail (same 64-row window / 70% threshold).
	if !dictSampleHighCard(strs) {
		return false
	}

	// One pass: build the alphabet (byte -> dense code), and accumulate the raw
	// per-value floor, the total character count, fixed-length detection, and the
	// per-value length-prefix bytes (used only if the column is not fixed-length).
	// seen/code are stack arrays (zero-initialised, non-escaping); code[c] is only
	// read for bytes already marked in seen, so it needs no separate reset.
	var seen [256]bool
	var code [256]uint8
	var alphabet [qpackStrAlphaMaxAlphabet]byte
	a := 0
	totalChars := 0
	rawFloor := 0
	lenPrefixBytes := 0
	fixedLen := true
	firstLen := -1
	for _, s := range strs {
		l := len(s)
		if firstLen < 0 {
			firstLen = l
		} else if l != firstLen {
			fixedLen = false
		}
		totalChars += l
		rawFloor += uvarintLen(uint64(l)) + l
		lenPrefixBytes += uvarintLen(uint64(l))
		for i := range l {
			c := s[i]
			if !seen[c] {
				if a >= qpackStrAlphaMaxAlphabet {
					return false // alphabet too large to pack below 8 bits/char
				}
				seen[c] = true
				code[c] = uint8(a)
				alphabet[a] = c
				a++
			}
		}
	}
	if a < 2 {
		// a == 0 (all-empty) or a == 1 (single distinct byte): no positional
		// packing is possible/useful — the const/raw paths handle these.
		return false
	}
	cbits := bitsForDistinct(a) // ceil(log2 a), >= 1 since a >= 2

	bodyBytes := (totalChars*cbits + 7) >> 3
	overhead := 1 + uvarintLen(uint64(a)) + a + uvarintLen(uint64(n)) + 1 // tag,a,alphabet,n,flags
	if fixedLen {
		overhead += uvarintLen(uint64(firstLen))
	} else {
		overhead += lenPrefixBytes
	}
	// Never-larger gate: the packed block (header + body) must beat the raw
	// per-value floor (sum of length prefixes + bytes). rawFloor is the byte cost
	// of the per-value/raw path this codec replaces, so this guarantees the wire
	// never grows when the codec fires.
	if overhead+bodyBytes >= rawFloor {
		return false
	}

	e.writeHeader()
	out := e.buf
	out = append(out, tagColStrAlpha)
	out = appendUvarint(out, uint64(a))
	out = append(out, alphabet[:a]...)
	out = appendUvarint(out, uint64(n))
	var flags byte
	if fixedLen {
		flags |= 1
	}
	out = append(out, flags)
	if fixedLen {
		out = appendUvarint(out, uint64(firstLen))
	} else {
		for _, s := range strs {
			out = appendUvarint(out, uint64(len(s)))
		}
	}

	// Grow without the make([]byte) zero-fill: every body byte is written below,
	// so the (possibly stale) grown region needs no clearing.
	start := len(out)
	out = slices.Grow(out, bodyBytes)[:start+bodyBytes]
	body := out[start : start+bodyBytes]
	if cbits == 4 {
		// Hex / 16-symbol alphabet — the dominant ID case. Pack two nibbles per
		// byte (LSB-first: first char low, second char high).
		if fixedLen && firstLen&1 == 0 {
			// Fixed even length (32-char trace IDs, 16-char span IDs): every value
			// is a whole number of bytes, so nibbles never straddle a value boundary
			// — pack two-at-a-time with no carry branch.
			pos := 0
			for _, s := range strs {
				for i := 0; i < len(s); i += 2 {
					body[pos] = code[s[i]] | code[s[i+1]]<<4
					pos++
				}
			}
			e.buf = out
			return true
		}
		// Variable or odd length: carry a pending low nibble across boundaries.
		pos, pend := 0, -1
		for _, s := range strs {
			for i := 0; i < len(s); i++ {
				c := int(code[s[i]])
				if pend < 0 {
					pend = c
				} else {
					body[pos] = byte(pend | c<<4)
					pos++
					pend = -1
				}
			}
		}
		if pend >= 0 {
			body[pos] = byte(pend)
		}
		e.buf = out
		return true
	}
	// General LSB-first bit-writer over the char codes — identical layout to
	// bitpack.Pack, so bitpack.Unpack reverses it on decode (codes are < a <=
	// 2^cbits, so no masking is needed).
	var acc uint64
	var have uint
	pos := 0
	for _, s := range strs {
		for i := 0; i < len(s); i++ {
			acc |= uint64(code[s[i]]) << have
			have += uint(cbits)
			for have >= 8 {
				body[pos] = byte(acc)
				acc >>= 8
				have -= 8
				pos++
			}
		}
	}
	if have > 0 {
		body[pos] = byte(acc)
	}
	e.buf = out
	return true
}

// readStringColumnAlpha decodes a tagColStrAlpha block (tag at d.i) into the n
// per-row strings. All characters are materialised into ONE slab (a single
// allocation for the whole column) and returned as views into it; the bit-
// unpacked codes reuse the shared transient scratch. Bounds mirror the other
// columnar string readers so a hostile header cannot drive an oversized alloc.
func (d *Decoder) readStringColumnAlpha(n int) ([]string, error) {
	d.i++ // consume tagColStrAlpha
	a64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if a64 < 2 || a64 > qpackStrAlphaMaxAlphabet {
		// a >= 2 keeps cbits >= 1 (non-empty body), so totalChars is bounded by
		// the remaining buffer below before any allocation.
		return nil, ErrBadTag
	}
	a := int(a64)
	if a > len(d.buf)-d.i {
		return nil, ErrShortBuffer
	}
	alphabet := d.buf[d.i : d.i+a] // alias; only read, never returned
	d.i += a

	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if int(n64) != n {
		return nil, ErrTypeMismatch
	}
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	flags := d.buf[d.i]
	d.i++
	fixedLen := flags&1 != 0
	cbits := bitsForDistinct(a)

	// Lengths. fixedLen stores one shared length; otherwise n per-row varints.
	// For the varint case the region is scanned twice (once to sum totalChars and
	// bound it before allocating, once to slice the slab) — re-reading varints is
	// cheap and avoids a per-row lengths allocation, keeping decode allocs on par
	// with the raw path it replaces.
	rem := uint64(len(d.buf) - d.i)
	// Each char costs cbits bits in the packed body, so the total character count
	// is bounded by rem*8/cbits (NOT rem: cbits < 8 means totalChars can
	// legitimately exceed the remaining byte count). Cap that bound by maxInt as
	// well so the int(tc) narrowings below cannot truncate a multi-GiB count to a
	// negative length on a 32-bit build (which would bypass the body check and
	// panic), mirroring the origLen guard in decodeRANS.
	maxChars := rem * 8 / uint64(cbits)
	if maxInt := uint64(int(^uint(0) >> 1)); maxChars > maxInt {
		maxChars = maxInt
	}
	var (
		totalChars int
		fixedL     int
		lenStart   int
	)
	if fixedLen {
		l64, k := readUvarint(d.buf[d.i:])
		if k <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += k
		// totalChars = n*L; guard against multiplication overflow and the bound.
		tc := uint64(n) * l64
		if l64 != 0 && tc/l64 != uint64(n) {
			return nil, ErrInvalidLength // overflow
		}
		if tc > maxChars {
			return nil, ErrShortBuffer
		}
		totalChars = int(tc)
		fixedL = int(l64)
	} else {
		lenStart = d.i
		// Guard inside the loop so a hostile run of huge length varints bails
		// immediately rather than accumulating past the bound.
		var tc uint64
		for range n {
			l64, k := readUvarint(d.buf[d.i:])
			if k <= 0 {
				return nil, ErrInvalidLength
			}
			d.i += k
			tc += l64
			if tc > maxChars {
				return nil, ErrShortBuffer
			}
		}
		totalChars = int(tc)
	}

	bodyBytes := (totalChars*cbits + 7) >> 3
	if bodyBytes > len(d.buf)-d.i {
		return nil, ErrShortBuffer
	}
	body := d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes

	slab := make([]byte, totalChars)
	if cbits == 4 {
		// Hex / 16-symbol alphabet — the dominant ID case. Fuse the unpack and the
		// alphabet scatter into one pass over the body (two nibbles per byte),
		// skipping the uint64 scratch and the general bit-window decoder. a in
		// (8,16]; when a < 16 a nibble may exceed the alphabet and must be rejected.
		full := totalChars &^ 1
		k := 0
		if a64 == 16 {
			for ; k < full; k += 2 {
				b := body[k>>1]
				slab[k] = alphabet[b&0xf]
				slab[k+1] = alphabet[b>>4]
			}
		} else {
			for ; k < full; k += 2 {
				b := body[k>>1]
				lo, hi := b&0xf, b>>4
				if uint64(lo) >= a64 || uint64(hi) >= a64 {
					return nil, ErrBadTag
				}
				slab[k] = alphabet[lo]
				slab[k+1] = alphabet[hi]
			}
		}
		if k < totalChars { // trailing odd nibble (low half of the last byte)
			lo := body[k>>1] & 0xf
			if uint64(lo) >= a64 {
				return nil, ErrBadTag
			}
			slab[k] = alphabet[lo]
		}
	} else {
		// General path: unpack the char codes into the shared transient scratch,
		// then map each code to its alphabet byte.
		if cap(d.deltaScratch) < totalChars {
			d.deltaScratch = make([]uint64, totalChars)
		}
		codes := d.deltaScratch[:totalChars]
		bitpack.Unpack(codes, body, cbits)
		for k, c := range codes {
			if c >= a64 {
				return nil, ErrBadTag
			}
			slab[k] = alphabet[c]
		}
	}

	out := make([]string, n)
	off := 0
	if fixedLen {
		for i := range n {
			out[i] = unsafestr.String(slab[off : off+fixedL])
			off += fixedL
		}
	} else {
		j := lenStart
		for i := range n {
			l64, k := readUvarint(d.buf[j:])
			j += k
			l := int(l64)
			out[i] = unsafestr.String(slab[off : off+l])
			off += l
		}
	}
	return out, nil
}
