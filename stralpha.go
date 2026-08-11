package qdf

import (
	"encoding/binary"

	"github.com/alex60217101990/qdf/internal/unsafestr"
)

// Per-field alphabet packing (tagStrAlpha) for the row-major struct path.
//
// The columnar codec in qpack_stralpha.go builds one alphabet for a whole
// column, which it can do because a column is contiguous. Row-major interleaves
// fields, so the alphabet is carried per field instead — named by ID when it is
// one everybody knows, declared once on the wire otherwise, and referenced
// afterwards.
//
// Scored per field against the raw floor on the access-log and log profiles,
// this wins where the string delta loses and loses where the delta wins:
// trace_id -45.4% (delta +5.9%), span_id -41.2% (delta +11.4%), request -23.2%
// (delta -1.2%). A per-VALUE alphabet loses almost everywhere (+18.5% on
// request) because the table ships with each value; amortising it per FIELD is
// the whole idea.

// Well-known alphabet IDs. The selector byte IS the ID, so these cost nothing
// beyond it and need no per-field memory on either side: a reader decodes such
// a value with no recollection of the field it came from, which is one less
// thing the six readers of struct values can get wrong.
const (
	strAlphaDecimal  = 1
	strAlphaHexLower = 2
	strAlphaHexUpper = 3
	strAlphaLowerNum = 4
	strAlphaB64URL   = 5

	strAlphaSelDeclare = 0x00
	strAlphaSelRef     = 0xFF
)

// strAlphaOrder lists the well-known alphabets narrowest first: fewer symbols
// means fewer bits per character, so the first match is also the best one.
var strAlphaOrder = [...]uint8{
	strAlphaDecimal, strAlphaHexLower, strAlphaHexUpper, strAlphaLowerNum, strAlphaB64URL,
}

type strAlphaSet struct {
	symbols []byte
	// ranges holds the closed byte ranges the alphabet is the union of, which
	// is what lets the membership test run a word at a time. Every well-known
	// alphabet here is such a union; an arbitrary set would not be.
	ranges [][2]byte
	bits   int
	member [256]bool
	code   [256]uint8
}

var strAlphaSets = buildStrAlphaSets()

func buildStrAlphaSets() [6]*strAlphaSet {
	mk := func(chars string, ranges ...[2]byte) *strAlphaSet {
		s := &strAlphaSet{symbols: []byte(chars), ranges: ranges}
		for i := range len(chars) {
			c := chars[i]
			s.member[c] = true
			s.code[c] = uint8(i)
		}
		s.bits = bitsForDistinct(len(chars))
		return s
	}
	var out [6]*strAlphaSet
	out[strAlphaDecimal] = mk("0123456789", [2]byte{'0', '9'})
	out[strAlphaHexLower] = mk("0123456789abcdef", [2]byte{'0', '9'}, [2]byte{'a', 'f'})
	out[strAlphaHexUpper] = mk("0123456789ABCDEF", [2]byte{'0', '9'}, [2]byte{'A', 'F'})
	out[strAlphaLowerNum] = mk("0123456789abcdefghijklmnopqrstuvwxyz",
		[2]byte{'0', '9'}, [2]byte{'a', 'z'})
	out[strAlphaB64URL] = mk("-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz",
		[2]byte{'-', '-'}, [2]byte{'0', '9'}, [2]byte{'A', 'Z'}, [2]byte{'_', '_'},
		[2]byte{'a', 'z'})
	return out
}

const (
	strAlphaOnes = 0x0101010101010101
	strAlphaHigh = 0x8080808080808080
)

// strAlphaInsideRange returns a word whose byte k has its high bit set when
// byte k of w lies inside [lo, hi].
//
// Borrow is made impossible rather than corrected for. Setting every byte's
// high bit before subtracting a value below 0x80 means the subtraction cannot
// borrow out of its own byte, so the surviving high bit means exactly "this
// byte is >= lo" — and symmetrically for "<= hi" with w's high bits cleared.
//
// The textbook hasless/hasmore pair is NOT enough here. Its ^w guard only
// suppresses a false flag when the byte's own high bit is set, so a byte below
// lo still borrows into its neighbour and marks it. That version reported 'A'
// as outside [A,Z] because the '6' beside it borrowed, and it read "486733g9"
// as lowercase hex — both caught by the cross-check against the scalar loop,
// which is why that test exists.
//
// Valid for lo, hi < 0x80 and for byte values < 0x80, which every well-known
// alphabet and every value it can accept satisfy.
func strAlphaInsideRange(w uint64, lo, hi byte) uint64 {
	lowBits := w &^ strAlphaHigh
	geLo := (lowBits | strAlphaHigh) - strAlphaOnes*uint64(lo)
	leHi := (strAlphaOnes*uint64(hi) | strAlphaHigh) - lowBits
	// A byte whose own high bit was set is not in any alphabet here; fold that
	// in so it can never be reported as inside.
	return geLo & leHi & ^w & strAlphaHigh
}

// strAlphaWellKnown returns the ID of the narrowest well-known alphabet that
// contains every character of s, or 0.
//
// Eight characters per iteration rather than eight table probes. It matters
// because this runs on every value of every string field including the ones
// that fail, and failing cheaply is what keeps a codec's cost proportional to
// its win — the same property that governs the delta's prefix compare.
func strAlphaWellKnown(s string) uint8 {
	if len(s) == 0 {
		return 0
	}
	b := unsafestr.Bytes(s)
	for _, id := range strAlphaOrder {
		set := strAlphaSets[id]
		if strAlphaAllIn(b, set) {
			return id
		}
	}
	return 0
}

func strAlphaAllIn(b []byte, set *strAlphaSet) bool {
	i := 0
	for ; i+8 <= len(b); i += 8 {
		w := binary.LittleEndian.Uint64(b[i:])
		// A byte is in the alphabet when it is inside at least one range, so OR
		// the per-range masks and require every byte to have been claimed.
		var inside uint64
		for _, r := range set.ranges {
			inside |= strAlphaInsideRange(w, r[0], r[1])
		}
		if inside != strAlphaHigh {
			return false
		}
	}
	for ; i < len(b); i++ {
		if !set.member[b[i]] {
			return false
		}
	}
	return true
}

// strAlphaCost is the exact byte count the matching append writes. tableLen is
// 0 for the well-known and reference forms.
func strAlphaCost(nchars, bits int, sel uint8, tableLen int) int {
	n := 1 + 1 + uvarintLen(uint64(nchars)) + (nchars*bits+7)/8
	if sel == strAlphaSelDeclare {
		n += uvarintLen(uint64(tableLen)) + tableLen
	}
	return n
}

// appendStrAlphaBody packs s LSB-first and appends it.
func appendStrAlphaBody(buf []byte, code *[256]uint8, bits int, s string) []byte {
	buf = appendUvarint(buf, uint64(len(s)))
	start := len(buf)
	buf = append(buf, make([]byte, (len(s)*bits+7)/8)...)
	body := buf[start:]
	bit := 0
	for i := range len(s) {
		v := uint32(code[s[i]])
		for b := range bits {
			if v&(1<<uint(b)) != 0 {
				body[bit>>3] |= 1 << uint(bit&7)
			}
			bit++
		}
	}
	return buf
}

func appendStrAlphaWellKnown(buf []byte, id uint8, s string) []byte {
	set := strAlphaSets[id]
	buf = append(buf, tagStrAlpha, id)
	return appendStrAlphaBody(buf, &set.code, set.bits, s)
}

func appendStrAlphaDeclared(buf []byte, alphabet []byte, code *[256]uint8, s string) []byte {
	buf = append(buf, tagStrAlpha, strAlphaSelDeclare)
	buf = appendUvarint(buf, uint64(len(alphabet)))
	buf = append(buf, alphabet...)
	return appendStrAlphaBody(buf, code, bitsForDistinct(len(alphabet)), s)
}

func appendStrAlphaRef(buf []byte, code *[256]uint8, bits int, s string) []byte {
	buf = append(buf, tagStrAlpha, strAlphaSelRef)
	return appendStrAlphaBody(buf, code, bits, s)
}

// strAlphaTable is a field's declared alphabet on the decode side.
type strAlphaTable struct {
	alphabet []byte
	bits     int
}

// readStrAlpha decodes one tagStrAlpha value.
//
// tbl is the table this field declared earlier, or nil when the caller has no
// field context; a declaration records itself in it so the reference forms that
// follow can use it. st, when non-nil, owns the arena the value is cut from.
//
// Every length is validated against the buffer before a byte is read, and every
// character code against the alphabet before it indexes it: a hostile stream
// can claim an alphabet larger than it ships, or a code past the end of one.
func readStrAlpha(buf []byte, tbl *strAlphaTable, st *decState) (string, int, error) {
	if len(buf) < 2 || buf[0] != tagStrAlpha {
		return "", 0, ErrBadTag
	}
	sel := buf[1]
	i := 2

	var alphabet []byte
	var bits int
	switch {
	case sel == strAlphaSelDeclare:
		a64, n := readUvarint(buf[i:])
		if n <= 0 {
			return "", 0, ErrShortBuffer
		}
		i += n
		if a64 < 2 || a64 > qpackStrAlphaMaxAlphabet {
			return "", 0, ErrInvalidLength
		}
		a := int(a64)
		if a > len(buf)-i {
			return "", 0, ErrShortBuffer
		}
		alphabet = buf[i : i+a]
		i += a
		bits = bitsForDistinct(a)
		if tbl != nil {
			tbl.alphabet, tbl.bits = alphabet, bits
		}
	case sel == strAlphaSelRef:
		if tbl == nil || len(tbl.alphabet) < 2 {
			return "", 0, ErrUnknownStateID
		}
		alphabet, bits = tbl.alphabet, tbl.bits
	case int(sel) < len(strAlphaSets) && strAlphaSets[sel] != nil:
		set := strAlphaSets[sel]
		alphabet, bits = set.symbols, set.bits
	default:
		return "", 0, ErrBadTag
	}

	n64, n := readUvarint(buf[i:])
	if n <= 0 {
		return "", 0, ErrShortBuffer
	}
	i += n
	// Each character needs at least one bit, so a count past eight times the
	// remaining bytes is malformed — checked before the multiplication below so
	// nchars*bits cannot overflow.
	if n64 > uint64(len(buf)-i)*8 {
		return "", 0, ErrShortBuffer
	}
	nchars := int(n64)
	need := (nchars*bits + 7) / 8
	if need > len(buf)-i {
		return "", 0, ErrShortBuffer
	}
	body := buf[i : i+need]
	i += need

	var out []byte
	if st != nil {
		out = st.strDeltaAlloc(nchars)
	} else {
		out = make([]byte, nchars)
	}
	bit := 0
	mask := uint32(1)<<uint(bits) - 1
	for k := range nchars {
		var v uint32
		for b := range bits {
			if body[bit>>3]&(1<<uint(bit&7)) != 0 {
				v |= 1 << uint(b)
			}
			bit++
		}
		v &= mask
		if int(v) >= len(alphabet) {
			return "", 0, ErrInvalidLength
		}
		out[k] = alphabet[v]
	}
	// The bytes are ours — from the arena or freshly made — so the header
	// aliases them rather than paying a second copy, exactly as the delta does.
	return unsafestr.String(out), i, nil
}
