package qdf

import (
	"math"
	"math/bits"
	"slices"
)

// ALP (Adaptive Lossless floating-Point, CWI 2023), decimal path. Encodes a
// []float64 whose values are decimals in disguise (quantized telemetry, prices,
// percentages, latencies) as integer mantissas I = round(v * 10^d) bit-packed
// with Frame-of-Reference, plus an exception list for any value that does not
// reconstruct bit-for-bit (NaN/±Inf/-0 and genuinely-irrational floats). Wins
// big over Gorilla on quantized data and decodes far faster; the float64 picker
// keeps Gorilla/raw for pure-smooth data where ALP would lose.

const alpMaxExp = 18

var (
	alpPow10 [alpMaxExp + 1]float64
	alpInv10 [alpMaxExp + 1]float64
)

func init() {
	p := 1.0
	for i := 0; i <= alpMaxExp; i++ {
		alpPow10[i] = p
		alpInv10[i] = 1.0 / p
		p *= 10
	}
}

// alpMaxExpSearch caps the effective-exponent search. 0..15 covers every
// practical decimal resolution; 16..18 only ever add exceptions.
const alpMaxExpSearch = 15

// alpMaxElems caps the element count ALP will encode or accept. A constant
// (width==0) slice carries no per-element bytes, so n is otherwise unbounded by
// the buffer; capping on both sides keeps a hostile header from forcing an
// oversized allocation while never rejecting anything the encoder emits.
const alpMaxElems = 1 << 24

// alpFloatPlan is the chosen encoding parameters for one []float64 block.
type alpFloatPlan struct {
	d      int   // effective decimal exponent (e-f)
	forMin int64 // FOR reference of the non-exception integer mantissas
	width  uint8 // bits per packed element, 0..56
	exc    int   // exception count
}

// alpScoreExp evaluates one effective exponent d over s: returns the FOR min,
// bit width, exception count. Round-trip uses exact float64 equality.
func alpScoreExp(s []float64, d int) (forMin int64, width uint8, exc int) {
	pe := alpPow10[d]
	ie := alpInv10[d]
	mn := int64(math.MaxInt64)
	mx := int64(math.MinInt64)
	for _, v := range s {
		I := int64(math.RoundToEven(v * pe))
		// Bit-exact round-trip check. Arithmetic == would treat -0.0 as equal
		// to +0.0 and silently drop the sign bit, and NaN != NaN; comparing
		// bits makes -0.0/NaN/±Inf land in the exception list as the spec
		// requires (exact float64 bit equality).
		if math.Float64bits(float64(I)*ie) != math.Float64bits(v) {
			exc++
			continue
		}
		if I < mn {
			mn = I
		}
		if I > mx {
			mx = I
		}
	}
	if mn > mx { // all exceptions
		return 0, 0, exc
	}
	return mn, uint8(bits.Len64(uint64(mx - mn))), exc
}

// alpChooseExp samples ~32 evenly-spaced values to pick the cheapest effective
// exponent in 0..alpMaxExpSearch by FOR-width + exception cost on the sample.
func alpChooseExp(s []float64) int {
	const sampleN = 32
	sample := s
	if len(s) > sampleN {
		sample = make([]float64, 0, sampleN)
		step := len(s) / sampleN
		for i := 0; i < len(s); i += step {
			sample = append(sample, s[i])
		}
	}
	bestD, bestCost := 0, math.MaxInt64
	for d := 0; d <= alpMaxExpSearch; d++ {
		_, w, exc := alpScoreExp(sample, d)
		cost := (int(w)*len(sample)+7)/8 + exc*10
		if cost < bestCost {
			bestCost, bestD = cost, d
		}
	}
	return bestD
}

// alpPlanFloat64 chooses the exponent, scores the full block, and returns a
// conservative upper bound on the encoded byte size. ok is false when ALP is
// not applicable: width exceeds the 56-bit FOR cap, or so many values are
// exceptions that ALP can never beat raw. The byte estimate over-counts
// exception positions (5-byte varuint) so the picker never chooses ALP when it
// would actually grow the wire.
func alpPlanFloat64(s []float64) (plan alpFloatPlan, estBytes int, ok bool) {
	n := len(s)
	if n == 0 {
		return alpFloatPlan{}, 2 + 1, true // tag+kind+varuint(0)
	}
	if n > alpMaxElems {
		return alpFloatPlan{}, 0, false // too large; raw/Gorilla (buffer-bounded) handle it
	}
	d := alpChooseExp(s)
	forMin, width, exc := alpScoreExp(s, d)
	if width > qpackForMaxBits {
		return alpFloatPlan{}, 0, false
	}
	if exc >= n { // nothing packs; raw/Gorilla strictly better
		return alpFloatPlan{}, 0, false
	}
	plan = alpFloatPlan{d: d, forMin: forMin, width: width, exc: exc}
	body := (int(width)*n + 7) / 8
	estBytes = 2 + uvarintLen(uint64(n)) + 1 + uvarintLen(zigzagEncode64(forMin)) +
		1 + body + uvarintLen(uint64(exc)) + exc*(5+8)
	return plan, estBytes, true
}

// writePackedALPFloat64Slice emits s under the ALP decimal codec using a
// pre-computed plan (from alpPlanFloat64). The caller guarantees plan.ok.
func (e *Encoder) writePackedALPFloat64Slice(s []float64, plan alpFloatPlan) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+1+10+1+(int(plan.width)*n+7)/8+10+plan.exc*(5+8))
	out = append(out, tagPackALP, qpackKindFloat64)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	out = append(out, byte(plan.d))
	out = appendUvarint(out, zigzagEncode64(plan.forMin))
	out = append(out, plan.width)

	pe := alpPow10[plan.d]
	ie := alpInv10[plan.d]

	if plan.width > 0 {
		// Pack I_i - forMin; exceptions occupy a 0 slot.
		packed := make([]uint64, n)
		for i, v := range s {
			I := int64(math.RoundToEven(v * pe))
			if math.Float64bits(float64(I)*ie) == math.Float64bits(v) {
				packed[i] = uint64(I - plan.forMin)
			}
		}
		bodyBytes := (int(plan.width)*n + 7) / 8
		base := len(out)
		out = append(out, make([]byte, bodyBytes)...)
		bitPackU64LE(out[base:], packed, int(plan.width))
	}

	// Exception list.
	out = appendUvarint(out, uint64(plan.exc))
	for i, v := range s {
		I := int64(math.RoundToEven(v * pe))
		if math.Float64bits(float64(I)*ie) != math.Float64bits(v) {
			out = appendUvarint(out, uint64(i))
			out = appendU64(out, math.Float64bits(v))
		}
	}
	e.buf = out
}

// readPackedALPFloat64Slice decodes an ALP payload. The cursor (d.i) must point
// at the kind byte (the caller consumed the tag). Bounds-checked against hostile
// input: width capped at 56, body size validated overflow-safe, exception
// positions validated < n.
func (d *Decoder) readPackedALPFloat64Slice() ([]float64, error) {
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	if d.buf[d.i] != qpackKindFloat64 {
		return nil, ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if n64 == 0 {
		return []float64{}, nil
	}
	if n64 > alpMaxElems {
		return nil, ErrInvalidLength
	}
	// Header: d(1), forMin zigzag-varuint, width(1).
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	expD := int(d.buf[d.i])
	d.i++
	if expD > alpMaxExp {
		return nil, ErrBadTag
	}
	fm64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	forMin := zigzagDecode64(fm64)
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	width := int(d.buf[d.i])
	d.i++
	if width > qpackForMaxBits {
		return nil, ErrBadTag
	}
	// Body, overflow-safe size check (mirrors readPackedForHeader).
	rem := uint64(len(d.buf) - d.i)
	if width > 0 && n64 > rem*8/uint64(width) {
		return nil, ErrShortBuffer
	}
	n := int(n64)
	ie := alpInv10[expD]
	out := make([]float64, n)
	if width > 0 {
		bodyBytes := int((n64*uint64(width) + 7) / 8)
		packed := make([]uint64, n)
		bitUnpackU64LE(packed, d.buf[d.i:d.i+bodyBytes], width)
		d.i += bodyBytes
		for i := range out {
			out[i] = float64(int64(packed[i])+forMin) * ie
		}
	} else {
		fv := float64(forMin) * ie
		for i := range out {
			out[i] = fv
		}
	}
	// Exception list.
	excN, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if excN > n64 {
		return nil, ErrInvalidLength
	}
	for range excN {
		pos64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return nil, ErrInvalidLength
		}
		d.i += nr
		if pos64 >= n64 {
			return nil, ErrInvalidLength
		}
		if d.i+8 > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out[pos64] = math.Float64frombits(readU64(d.buf[d.i:]))
		d.i += 8
	}
	return out, nil
}
