package qdf

import (
	"math"
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
)

// ALP-RD ("Real Doubles") codec for []float64 that ALP's decimal transform
// cannot capture, from the ALP paper (Afroozeh & Boncz, SIGMOD 2023). Each
// double is split at a chosen cut: the LEFT leftBW bits (sign + exponent +
// top mantissa, leftBW in [8,16]) repeat heavily on real-world series and are
// dictionary-coded with at most 8 entries (ceil(log2 D) bits/value); the
// RIGHT 64-leftBW bits are raw-bitpacked. Values whose left pattern misses
// the dictionary become exceptions (position + raw left bits). Lossless for
// every bit pattern including NaN/±Inf.
//
// Wire (shares tagPackALP with kind qpackKindALPRD64):
//
//	tag, kind, varuint(n),
//	byte(leftBW), byte(D), D x u16 LE dictionary,
//	ceil(cbits*n/8) LSB-first code body (absent when D == 1),
//	ceil((64-leftBW)*n/8) LSB-first right-bits body,
//	varuint(excN), excN x (varuint(pos), u16 LE left bits)
const (
	alprdMaxDict = 8  // 3-bit code ceiling, per the paper
	alprdMinLeft = 8  // right width <= 56 keeps bitpack in range
	alprdMaxLeft = 16 // left patterns must fit u16
	// alprdSample bounds the planner's frequency-count pass.
	alprdSample = 1024
)

type alprdPlan struct {
	dict   [alprdMaxDict]uint16
	dictN  uint8
	leftBW uint8
	cbits  uint8 // ceil(log2 dictN)
	exc    int   // exact exception count over the full slice
}

// alprdCodeBits returns ceil(log2 d) for d in [1,8].
func alprdCodeBits(d int) uint8 {
	switch {
	case d <= 1:
		return 0
	case d <= 2:
		return 1
	case d <= 4:
		return 2
	default:
		return 3
	}
}

// alprdPlanFloat64 picks the cut point and dictionary on a sample, then does
// one exact pass to count exceptions. estBytes is a safe upper bound: the
// picker may choose ALP-RD only when it strictly beats the alternatives, so
// the codec never grows the wire. ok is false when even the best plan cannot
// beat raw.
func alprdPlanFloat64(s []float64) (plan alprdPlan, estBytes int, ok bool) {
	n := len(s)
	if n < 16 {
		return plan, 0, false // header + dictionary dominate tiny slices
	}

	stride := 1
	if n > alprdSample {
		stride = n / alprdSample
	}

	// For each candidate cut, count sample frequencies of the left pattern
	// and keep the top-8 coverage. Patterns are u16; a tiny open-addressed
	// table avoids a map allocation.
	const tabSize = 4096 // strictly > the max sample count below, so probing always terminates
	var keys [tabSize]uint16
	var cnts [tabSize]int32

	bestBits := math.MaxInt
	for leftBW := alprdMinLeft; leftBW <= alprdMaxLeft; leftBW++ {
		shift := uint(64 - leftBW)
		for i := range tabSize {
			cnts[i] = 0
		}
		sampled := 0
		for i := 0; i < n && sampled < 2*alprdSample; i += stride {
			left := uint16(math.Float64bits(s[i]) >> shift)
			h := int(left*40503) & (tabSize - 1)
			for cnts[h] != 0 && keys[h] != left {
				h = (h + 1) & (tabSize - 1)
			}
			keys[h] = left
			cnts[h]++
			sampled++
		}
		// Top-8 coverage by simple selection (table is small).
		var top [alprdMaxDict]int32
		var topKey [alprdMaxDict]uint16
		for i := range tabSize {
			c := cnts[i]
			if c == 0 {
				continue
			}
			for j := range alprdMaxDict {
				if c > top[j] {
					copy(top[j+1:], top[j:alprdMaxDict-1])
					copy(topKey[j+1:], topKey[j:alprdMaxDict-1])
					top[j] = c
					topKey[j] = keys[i]
					break
				}
			}
		}
		d := 0
		covered := int32(0)
		for j := range alprdMaxDict {
			if top[j] == 0 {
				break
			}
			d++
			covered += top[j]
		}
		if d == 0 {
			continue
		}
		cbits := int(alprdCodeBits(d))
		// Projected bits/value: codes + right bits + amortized exceptions
		// (u16 left + ~2-byte position each).
		excFrac := float64(sampled-int(covered)) / float64(sampled)
		bits := float64(cbits) + float64(64-leftBW) + excFrac*float64(16+16)
		total := int(bits * float64(n))
		if total < bestBits {
			bestBits = total
			plan.leftBW = uint8(leftBW)
			plan.dictN = uint8(d)
			plan.cbits = uint8(cbits)
			copy(plan.dict[:], topKey[:d])
		}
	}
	if plan.dictN == 0 {
		return plan, 0, false
	}

	// Exact exception count over the full slice (8 u16 compares per value).
	shift := uint(64 - plan.leftBW)
	dict := plan.dict[:plan.dictN]
	exc := 0
	for _, v := range s {
		left := uint16(math.Float64bits(v) >> shift)
		if !slices.Contains(dict, left) {
			exc++
		}
	}
	plan.exc = exc

	rightBody := (int(64-plan.leftBW)*n + 7) / 8
	codeBody := (int(plan.cbits)*n + 7) / 8
	estBytes = 2 + uvarintLen(uint64(n)) + 2 + 2*int(plan.dictN) +
		codeBody + rightBody + uvarintLen(uint64(exc)) + exc*(5+2)
	if estBytes >= 2+uvarintLen(uint64(n))+8*n {
		return plan, 0, false
	}
	return plan, estBytes, true
}

// writePackedALPRDFloat64Slice emits s under the ALP-RD codec using a
// pre-computed plan from alprdPlanFloat64.
func (e *Encoder) writePackedALPRDFloat64Slice(s []float64, plan alprdPlan) {
	e.writeHeader()
	n := len(s)
	rightBW := 64 - int(plan.leftBW)
	rightBody := (rightBW*n + 7) / 8
	codeBody := (int(plan.cbits)*n + 7) / 8
	out := slices.Grow(e.buf, 2+10+2+2*int(plan.dictN)+codeBody+rightBody+10+plan.exc*7)
	out = append(out, tagPackALP, qpackKindALPRD64)
	out = appendUvarint(out, uint64(n))
	out = append(out, plan.leftBW, plan.dictN)
	for _, dv := range plan.dict[:plan.dictN] {
		out = append(out, byte(dv), byte(dv>>8))
	}

	// Stage codes and right bits; exceptions get code 0 (decoder overwrites
	// their left bits from the exception list).
	if cap(e.alpScratch) < n {
		e.alpScratch = make([]uint64, n)
	}
	rights := e.alpScratch[:n]
	if cap(e.wideU64) < n {
		e.wideU64 = make([]uint64, n)
	}
	codes := e.wideU64[:n]
	excIdx := e.alpExcScratch[:0]

	shift := uint(rightBW)
	rightMask := uint64(1)<<shift - 1
	dict := plan.dict[:plan.dictN]
	for i, v := range s {
		b := math.Float64bits(v)
		rights[i] = b & rightMask
		left := uint16(b >> shift)
		code := 0
		found := false
		for j, dv := range dict {
			if dv == left {
				code = j
				found = true
				break
			}
		}
		if !found {
			excIdx = append(excIdx, i)
		}
		codes[i] = uint64(code)
	}

	if plan.cbits > 0 {
		base := len(out)
		out = out[:base+codeBody]
		bitpack.Pack(out[base:base+codeBody], codes, int(plan.cbits))
	}
	base := len(out)
	out = out[:base+rightBody]
	bitpack.Pack(out[base:base+rightBody], rights, rightBW)

	out = appendUvarint(out, uint64(len(excIdx)))
	for _, i := range excIdx {
		out = appendUvarint(out, uint64(i))
		left := uint16(math.Float64bits(s[i]) >> shift)
		out = append(out, byte(left), byte(left>>8))
	}
	e.alpExcScratch = excIdx
	e.buf = out
}

// readPackedALPRDFloat64Slice decodes an ALP-RD body. The tag must already be
// consumed and the next byte must be qpackKindALPRD64.
func (d *Decoder) readPackedALPRDFloat64Slice() ([]float64, error) {
	if d.i >= len(d.buf) || d.buf[d.i] != qpackKindALPRD64 {
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
	// Every value costs >= 48 right bits on a valid stream; bound n by the
	// remaining bytes before any allocation.
	if rem := uint64(len(d.buf) - d.i); n64 > rem/6+1 {
		return nil, ErrShortBuffer
	}
	if n64 > uint64(math.MaxInt) {
		return nil, ErrInvalidLength
	}
	n := int(n64)
	if n == 0 {
		return []float64{}, nil
	}
	if d.i+2 > len(d.buf) {
		return nil, ErrShortBuffer
	}
	leftBW := int(d.buf[d.i])
	dictN := int(d.buf[d.i+1])
	d.i += 2
	if leftBW < alprdMinLeft || leftBW > alprdMaxLeft || dictN < 1 || dictN > alprdMaxDict {
		return nil, ErrBadTag
	}
	if d.i+2*dictN > len(d.buf) {
		return nil, ErrShortBuffer
	}
	var dict [alprdMaxDict]uint64
	for j := range dictN {
		dict[j] = uint64(d.buf[d.i]) | uint64(d.buf[d.i+1])<<8
		d.i += 2
	}

	cbits := int(alprdCodeBits(dictN))
	rightBW := 64 - leftBW
	codeBody := (cbits*n + 7) / 8
	rightBody := (rightBW*n + 7) / 8
	if d.i+codeBody+rightBody > len(d.buf) {
		return nil, ErrShortBuffer
	}

	// One pooled scratch carries both halves: codes in [:n], rights in [n:2n].
	if cap(d.deltaScratch) < 2*n {
		d.deltaScratch = make([]uint64, 2*n)
	}
	codes := d.deltaScratch[:n]
	if cbits > 0 {
		bitpack.Unpack(codes, d.buf[d.i:d.i+codeBody], cbits)
		d.i += codeBody
	} else {
		clear(codes)
	}
	rights := d.deltaScratch[n : 2*n]
	bitpack.Unpack(rights, d.buf[d.i:d.i+rightBody], rightBW)
	d.i += rightBody

	out := make([]float64, n)
	shift := uint(rightBW)
	for i := range out {
		c := codes[i]
		if c >= uint64(dictN) {
			return nil, ErrBadTag
		}
		out[i] = math.Float64frombits(dict[c]<<shift | rights[i])
	}

	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if excN64 > uint64(n) {
		return nil, ErrBadTag
	}
	for range int(excN64) {
		pos64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if pos64 >= uint64(n) || d.i+2 > len(d.buf) {
			return nil, ErrBadTag
		}
		left := uint64(d.buf[d.i]) | uint64(d.buf[d.i+1])<<8
		d.i += 2
		i := int(pos64)
		out[i] = math.Float64frombits(left<<shift | rights[i])
	}
	return out, nil
}

// skipALPRD walks an ALP-RD body without decoding it (Skip support). d.i
// points at the kind byte.
func (d *Decoder) skipALPRD() error {
	d.i++ // kind
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if n64 == 0 {
		return nil
	}
	if n64 > uint64(math.MaxInt/8) {
		return ErrInvalidLength
	}
	n := int(n64)
	if d.i+2 > len(d.buf) {
		return ErrShortBuffer
	}
	leftBW := int(d.buf[d.i])
	dictN := int(d.buf[d.i+1])
	d.i += 2
	if leftBW < alprdMinLeft || leftBW > alprdMaxLeft || dictN < 1 || dictN > alprdMaxDict {
		return ErrBadTag
	}
	codeBody := (int(alprdCodeBits(dictN))*n + 7) / 8
	rightBody := ((64-leftBW)*n + 7) / 8
	skip := 2*dictN + codeBody + rightBody
	if skip < 0 || d.i+skip > len(d.buf) {
		return ErrShortBuffer
	}
	d.i += skip
	excN64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return ErrInvalidLength
	}
	d.i += nr
	if excN64 > n64 {
		return ErrBadTag
	}
	for range int(excN64) {
		_, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return ErrInvalidLength
		}
		d.i += nr + 2
		if d.i > len(d.buf) {
			return ErrShortBuffer
		}
	}
	return nil
}
