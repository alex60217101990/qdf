package qdf

import (
	"encoding/binary"
	"sync/atomic"

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
// request) because the table ships with each value; amortizing it per FIELD is
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

// strAlphaNotMember marks a byte outside a learned alphabet in its code table.
// Real codes are below qpackStrAlphaMaxAlphabet, so the high bit is unambiguous
// and survives an OR — which is what makes the membership test branchless.
const strAlphaNotMember = 0xFF

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
// lo still borrows into its neighbor and marks it. That version reported 'A'
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

// strAlphaWorthIt reports whether a value of n characters packed at the given
// width clears the gain bar.
//
// Length and width only, never the bytes — which is the point. The bar cannot
// depend on content, so it belongs BEFORE the membership scan rather than
// after it: a value too short to profit is declined without being read at all.
func strAlphaWorthIt(n, bits, baselineCost int) bool {
	return strAlphaCost(n, bits, strAlphaSelRef, 0)*strAlphaMinGainDen < baselineCost*strAlphaMinGainNum
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
	// Shift register: characters accumulate into a 64-bit window and leave four
	// bytes at a time. The obvious form tests and sets one bit per iteration —
	// len(s)*bits branches for every value — which profiled at 11.7% of RTB
	// encode, the largest single cost in this package's own code.
	//
	// The window can hold 31 leftover bits plus an 8-bit code, so it never
	// overflows, and a flush only happens with 32 whole bits due, so the
	// four-byte store is always inside the body.
	var acc uint64
	nb, pos := 0, 0
	for i := range len(s) {
		acc |= uint64(code[s[i]]) << uint(nb)
		nb += bits
		if nb >= 32 {
			binary.LittleEndian.PutUint32(body[pos:], uint32(acc))
			acc >>= 32
			nb -= 32
			pos += 4
		}
	}
	for nb > 0 {
		body[pos] = byte(acc)
		acc >>= 8
		nb -= 8
		pos++
	}
	return buf
}

func appendStrAlphaWellKnown(buf []byte, id uint8, s string) []byte {
	set := strAlphaSets[id]
	buf = append(buf, tagStrAlpha, id)
	return appendStrAlphaBody(buf, &set.code, set.bits, s)
}

func appendStrAlphaDeclared(buf, alphabet []byte, code *[256]uint8, s string) []byte {
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
	// Reslice to exactly nchars and range over it below: the arena's Alloc does
	// not inline, so len(out) is opaque to the prover and the store in the
	// unpack loop carried a bounds check PER DECODED CHARACTER. Ranging over a
	// slice of known length removes it.
	out = out[:nchars]
	// The mirror of the writer's shift register, and for the same reason: a bit
	// at a time is bits branches per character on a path that runs for every
	// packed value on the wire.
	mask := uint32(1)<<uint(bits) - 1
	var acc uint64
	nb, pos := 0, 0
	for k := range out {
		if nb < bits {
			// Refill. nb is below eight here, so a 32-bit load can never push
			// the window past 64.
			if pos+4 <= len(body) {
				acc |= uint64(binary.LittleEndian.Uint32(body[pos:])) << uint(nb)
				nb += 32
				pos += 4
			} else {
				for pos < len(body) && nb <= 56 {
					acc |= uint64(body[pos]) << uint(nb)
					nb += 8
					pos++
				}
			}
			if nb < bits {
				// The length prefix promised more characters than the body can
				// hold. Validated above, so this is belt and braces on a path
				// that parses untrusted bytes.
				return "", 0, ErrShortBuffer
			}
		}
		v := uint32(acc) & mask
		acc >>= uint(bits)
		nb -= bits
		if int(v) >= len(alphabet) {
			return "", 0, ErrInvalidLength
		}
		out[k] = alphabet[v]
	}
	// The bytes are ours — from the arena or freshly made — so the header
	// aliases them rather than paying a second copy, exactly as the delta does.
	return unsafestr.String(out), i, nil
}

// Firing counters, gated exactly as the delta's are: an atomic increment on
// every eligible value is not free, and these exist only so acceptance tests
// can assert which form ran. strDeltaCount turns both sets on in the test
// binary.
var (
	strAlphaEmittedWK   atomic.Int64
	strAlphaEmittedDecl atomic.Int64
	strAlphaEmittedRef  atomic.Int64
)

const (
	// strAlphaStableN values with no new symbol commit the alphabet. Short
	// enough to give up almost nothing on a field of thousands, long enough
	// that a field with a genuinely wide character set reveals it before a
	// table is written.
	strAlphaStableN = 16
	// strAlphaMinGainNum/Den: the packed form must cost less than THREE QUARTERS
	// of what it replaces.
	//
	// Not the delta's half, and the difference is not a preference. The delta's
	// saving is data-dependent and often a byte or two, so a high bar is what
	// stops it trading a full decode-side materialization for nothing. Alpha's
	// saving is structural — exactly 1 - bits/8 of the value, before overhead —
	// so a halving bar rejects the case the codec exists for: a 32-character hex
	// id packs to 19 bytes against 34, a 44% saving that is not a halving.
	//
	// Three quarters keeps that and still declines the shapes where the form is
	// marginal: base64url at 6 bits saves 25% before overhead, which on a short
	// value does not clear the bar.
	strAlphaMinGainNum = 3
	strAlphaMinGainDen = 4
	// strAlphaProbeN values without a single emission mute the field, and
	// strAlphaRearmN values later it is offered again.
	//
	// Without this the codec costs more than it saves. A free-text field runs
	// the learning byte-loop on every value until its symbol count reaches the
	// cap, and measured on the access-log corpus that was +46% encode CPU for
	// nothing: the field could never pack. The delta carries the same gate for
	// the same reason. Rearming matters as much as muting — a field whose data
	// turns narrow part-way through must be recoverable, or the gate becomes a
	// guess that costs wire.
	strAlphaProbeN = 24
	strAlphaRearmN = 2048
)

// tryWriteStringFieldAlpha writes s packed against this field's alphabet and
// reports whether it did. baselineCost is what the value would cost in the form
// the encoder would otherwise write.
func (e *Encoder) tryWriteStringFieldAlpha(s string, fs *strFieldState, baselineCost int) bool {
	if e.tryWriteStringFieldAlphaInner(s, fs, baselineCost) {
		fs.alphaProbe = 0
		return true
	}
	fs.alphaProbe++
	// Only once learning has actually started: while a field is still warming
	// up it has not been offered anything to decline, and muting it then would
	// retire the codec before it ever ran.
	if fs.learn != nil && fs.alphaProbe >= strAlphaProbeN && !fs.alphaDeclared {
		// Nothing emitted across a full probe run and no table to reference:
		// this field is not what the codec is for.
		fs.alphaMuted, fs.alphaProbe = true, 0
		fs.learn = nil
	}
	return false
}

func (e *Encoder) tryWriteStringFieldAlphaInner(s string, fs *strFieldState, baselineCost int) bool {
	// alphaOff and alphaMuted are checked by the caller, which is what keeps a
	// declined field from paying for this call at all.
	if len(s) == 0 {
		return false
	}

	// A well-known alphabet needs no state and no table, so it is tried first
	// and re-tested per value: the membership scan stops at the first character
	// outside the set, which makes it cheap exactly where it is about to fail.
	if fs.learn == nil {
		if fs.alphaID == 0 {
			// Discovery is a scan of the whole value, so an O(1) test comes
			// first. The narrowest well-known alphabet is ten symbols at four
			// bits, so a length that four bits cannot make pay is a length no
			// well-known set can.
			if !strAlphaWorthIt(len(s), 4, baselineCost) {
				return false
			}
			fs.alphaID = strAlphaWellKnown(s)
		}
		if id := fs.alphaID; id != 0 {
			set := strAlphaSets[id]
			switch {
			case !strAlphaWorthIt(len(s), set.bits, baselineCost):
				// This set cannot pay at this length. Two different reasons,
				// and they must not be confused: if not even a one-bit table
				// could win, the value is simply short and the field keeps the
				// alphabet it matched — dropping it here would cost every
				// later long value. Otherwise the set is wider than the field
				// needs, and a learned table is worth trying: values of
				// nothing but lowercase letters match the 36-symbol set at six
				// bits where their real 26 symbols pack at five.
				if !strAlphaWorthIt(len(s), 1, baselineCost) {
					return false
				}
				fs.alphaID = 0
			case strAlphaAllIn(unsafestr.Bytes(s), set):
				e.buf = appendStrAlphaWellKnown(e.buf, id, s)
				if strDeltaCount {
					strAlphaEmittedWK.Add(1)
				}
				return true
			default:
				fs.alphaID = 0 // a value fell outside the set this field matched
			}
		}
		// Count before learning. A table cannot be declared before
		// strAlphaStableN stable values have gone by, so on a field that never
		// gets that far the 288-byte table, the per-value byte loop and the
		// symbol bookkeeping are all spent on an emission that can never
		// happen. Measured on a fifteen-element slice: +74.4% encode CPU for
		// ZERO bytes of wire — the run ends one value before the table would
		// have been declared.
		//
		// Slice length is NOT the signal, and using it was a bug: one field's
		// state outlives the slice it was bound for. A payload of four hundred
		// records with two elements each shows a field eight hundred values,
		// and gating on "two" refuses a table that pays for itself many times
		// over. What matters is how many values this FIELD has seen, which is
		// what alphaProbe already counts.
		fs.alphaWarm++
		if fs.alphaWarm < strAlphaStableN {
			return false
		}
		fs.learn = newStrAlphaLearn()
	}

	// Accumulate, and declare when the set stops growing.
	l := fs.learn
	// A byte this field has not seen contributes the sentinel, whose high bit
	// survives the OR, so the settled case — every byte already known, which is
	// what a field looks like once it has stabilized — costs one OR per byte
	// and a single test, rather than a branch per byte.
	var unknown uint8
	for i := range len(s) {
		unknown |= l.code[s[i]]
	}
	if unknown&0x80 != 0 {
		if fs.alphaDeclared {
			// The table is on the wire and the decoder built its mapping from
			// those exact bytes, so it can never be revised: widening it would
			// re-number every symbol and change the packed width, and every
			// later reference would decode against a table the reader does not
			// have. Not a bigger wire — a desync. The value takes its ordinary
			// form and the table stays as declared.
			return false
		}
		for i := range len(s) {
			c := s[i]
			if l.code[c] != strAlphaNotMember {
				continue
			}
			if len(l.symbols) >= qpackStrAlphaMaxAlphabet {
				// Too wide to pack below eight bits per character. Marking the
				// field off is what keeps the scan from being spent on it again.
				fs.alphaOff = true
				fs.learn = nil
				return false
			}
			l.code[c] = uint8(len(l.symbols))
			l.symbols = append(l.symbols, c)
		}
		l.bits = uint8(bitsForDistinct(len(l.symbols)))
		l.stable = 0
		// A growing table is not evidence the codec is failing — it is evidence
		// it is still learning. The probe budget measures declines under a
		// SETTLED alphabet, so growth restarts it; otherwise the twenty-four
		// probes run out while the symbol set is still filling and the field is
		// retired before it ever gets to declare.
		fs.alphaProbe = 0
		// A table already on the wire cannot be revised, so a value that
		// introduces a new symbol takes its ordinary form.
		return false
	}
	if len(l.symbols) < 2 {
		return false
	}
	l.stable++
	bits := int(l.bits)

	if fs.alphaDeclared {
		if !strAlphaWorthIt(len(s), bits, baselineCost) {
			return false
		}
		e.buf = appendStrAlphaRef(e.buf, &l.code, bits, s)
		if strDeltaCount {
			strAlphaEmittedRef.Add(1)
		}
		return true
	}
	if l.stable < strAlphaStableN {
		return false
	}
	// The table is a one-time cost for the whole field, so charging it to the
	// single value that happens to declare it is the wrong test: a 26-symbol
	// table is 28 bytes, which no individual value can absorb, and the gate
	// would refuse every table of every size forever.
	//
	// Judge it the way it is actually paid instead. Each value from here on
	// saves (8-bits)/8 of its length, and this field has already produced
	// strAlphaStableN values with a settled alphabet — so require that those
	// already-seen values would themselves have covered the table. A field that
	// stops after the declaration has lost a bounded number of bytes once; one
	// that continues, which the stability run is evidence of, is ahead from the
	// next value on.
	if !strAlphaWorthIt(len(s), bits, baselineCost) {
		return false
	}
	tableCost := 2 + len(l.symbols)
	savedPerValue := len(s) * (8 - bits) / 8
	if savedPerValue*strAlphaStableN < tableCost {
		return false
	}
	e.buf = appendStrAlphaDeclared(e.buf, l.symbols, &l.code, s)
	if strDeltaCount {
		strAlphaEmittedDecl.Add(1)
	}
	fs.alphaDeclared = true
	return true
}
