package qdf

import (
	"math"
	"math/bits"
	"slices"

	"github.com/alex60217101990/qdf/internal/bitpack"
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

// alpExactFlag marks, in the wire's exponent byte, that the block reconstructs
// values by DIVIDING the integer mantissa by 10^d rather than multiplying it
// by a precomputed 1/10^d.
//
// Multiplying by the inverse is an approximation: 1/10^d is not representable,
// so the map I -> I*(1/10^d) misses many doubles that are exactly round(x,d).
// Measured over decimal columns, 13-35% of perfectly representable values
// failed the encoder's round-trip check and were stored as raw exceptions,
// which is what actually dominated the block size (a 2-decimal price column
// spent 11 KB on the packed body and 14 KB on 1087 spurious exceptions).
// Dividing by the exactly-representable 10^d is correctly rounded and has zero
// false exceptions; it measured ~3% slower on the decode loop, which is
// memory-bound rather than divider-bound.
//
// The flag lives in the high bit of the exponent byte because a decoder that
// predates it validates that byte against alpMaxExp (18) and returns ErrBadTag
// — a loud failure rather than silently different values. New decoders read
// both forms, so every blob ever written stays readable.
const alpExactFlag = 0x80

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
	// exact selects DIVISION by 10^d over multiplication by a precomputed
	// 1/10^d when reconstructing. Division is correctly rounded and costs no
	// false exceptions, but it is a newer wire form (alpExactFlag) and a few
	// percent slower to decode — so it is only chosen when it actually
	// removes exceptions. Columns that score identically keep the original
	// multiply form: same bytes, same speed, readable by older decoders.
	exact bool
}

// alpRoundTrips reports whether the integer mantissa iv reconstructs v exactly
// under the block's chosen operation. Encoder and decoder MUST agree on it —
// that is what the exponent byte's alpExactFlag carries.
func alpRoundTrips(iv int64, v, pe float64, exact bool) bool {
	if exact {
		return math.Float64bits(float64(iv)/pe) == math.Float64bits(v)
	}
	return math.Float64bits(float64(iv)*(1/pe)) == math.Float64bits(v)
}

// alpScoreExp evaluates one effective exponent d over s: returns the FOR min,
// bit width, exception count. Round-trip uses exact float64 equality.
func alpScoreExp(s []float64, d int) (forMin int64, width uint8, exc int, exact bool) {
	pe := alpPow10[d]
	ie := alpInv10[d]
	// Score both reconstructions in one pass. They accept DIFFERENT sets of
	// values, so each needs its own FOR range: pairing one op's range with the
	// other op's accepted set yields a width wider than necessary (measured
	// +167 B on the IoT benchmark when the ranges were shared).
	mnDiv, mxDiv := int64(math.MaxInt64), int64(math.MinInt64)
	mnMul, mxMul := int64(math.MaxInt64), int64(math.MinInt64)
	excDiv, excMul := 0, 0
	for _, v := range s {
		iv := int64(math.RoundToEven(v * pe))
		bv := math.Float64bits(v)
		// Bit-exact round-trip checks. Arithmetic == would treat -0.0 as equal
		// to +0.0 and silently drop the sign bit, and NaN != NaN; comparing
		// bits makes -0.0/NaN/±Inf land in the exception list as the spec
		// requires (exact float64 bit equality).
		if math.Float64bits(float64(iv)/pe) == bv {
			mnDiv = min(mnDiv, iv)
			mxDiv = max(mxDiv, iv)
		} else {
			excDiv++
		}
		if math.Float64bits(float64(iv)*ie) == bv {
			mnMul = min(mnMul, iv)
			mxMul = max(mxMul, iv)
		} else {
			excMul++
		}
	}
	// Choose the reconstruction by BYTES, not by exception ratio. The two ops
	// accept different value sets, so each implies its own FOR width as well
	// as its own exception count — scoring only the counts let a single
	// exception flip a 40% size difference, took the exact form for zero gain
	// on short blocks, and once picked a form 2.17x LARGER because the values
	// only division accepts forced the FOR width from 12 to 50 bits.
	//
	// An exception costs a varuint position plus the raw 8-byte value; 13 is
	// the same conservative charge alpPlanFloat64's estimate uses. Ties go to
	// multiplication: same bytes, slightly faster decode, and readable by
	// decoders that predate alpExactFlag.
	const excBytes = 13
	widthOf := func(lo, hi int64) int {
		if lo > hi {
			return 0
		}
		return bits.Len64(uint64(hi - lo))
	}
	wDiv, wMul := widthOf(mnDiv, mxDiv), widthOf(mnMul, mxMul)
	costDiv := (wDiv*len(s)+7)/8 + excDiv*excBytes
	costMul := (wMul*len(s)+7)/8 + excMul*excBytes
	exact = costDiv < costMul
	mn, mx, exc := mnMul, mxMul, excMul
	if exact {
		mn, mx, exc = mnDiv, mxDiv, excDiv
	}
	if mn > mx { // all exceptions
		return 0, 0, exc, exact
	}
	return mn, uint8(bits.Len64(uint64(mx - mn))), exc, exact
}

// alpChooseExp samples ~32 evenly-spaced values to pick the cheapest effective
// exponent in 0..alpMaxExpSearch by FOR-width + exception cost on the sample.
func alpChooseExp(s []float64) int {
	const sampleN = 32
	sample := s
	if len(s) > sampleN {
		var sampleBuf [sampleN]float64
		sample = sampleBuf[:0]
		step := len(s) / sampleN
		for i := 0; i < len(s); i += step {
			sample = append(sample, s[i])
		}
	}
	bestD, bestCost := 0, math.MaxInt // platform max sentinel (int, 32-bit safe)
	for d := 0; d <= alpMaxExpSearch; d++ {
		// alpScoreExp returns the cheaper of the two reconstructions for this
		// exponent, so costing its (w, exc) compares each exponent at its own
		// optimum — mixing ops across exponents made the curve incomparable
		// and once picked d=4 where d=2 was 52% smaller.
		_, w, exc, _ := alpScoreExp(sample, d)
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
	forMin, width, exc, exact := alpScoreExp(s, d)
	if width > qpackForMaxBits {
		return alpFloatPlan{}, 0, false
	}
	if exc >= n { // nothing packs; raw/Gorilla strictly better
		return alpFloatPlan{}, 0, false
	}
	plan = alpFloatPlan{d: d, forMin: forMin, width: width, exc: exc, exact: exact}
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
	expByte := byte(plan.d)
	if plan.exact {
		expByte |= alpExactFlag
	}
	out = append(out, expByte)
	out = appendUvarint(out, zigzagEncode64(plan.forMin))
	out = append(out, plan.width)

	pe := alpPow10[plan.d]

	// Acquire exception-index scratch. Populated during the pack pass so the
	// exception-emit phase below never needs a second O(n) re-scan of s.
	excIdx := e.alpExcScratch[:0]

	if plan.width > 0 {
		// Pack I_i - forMin; exceptions occupy a 0 slot. Reuse pooled scratch
		// instead of a per-call make; clear is REQUIRED because exception slots
		// are never written in the loop below and must read back as 0 (a reused
		// buffer would otherwise leak a prior column's mantissa into them).
		if cap(e.alpScratch) < n {
			e.alpScratch = make([]uint64, n)
		}
		packed := e.alpScratch[:n]
		clear(packed)
		for i, v := range s {
			I := int64(math.RoundToEven(v * pe))
			if alpRoundTrips(I, v, pe, plan.exact) {
				packed[i] = uint64(I - plan.forMin)
			} else {
				excIdx = append(excIdx, i) // collect exception index for emit below
			}
		}
		bodyBytes := (int(plan.width)*n + 7) / 8
		base := len(out)
		// Extend into the capacity slices.Grow already reserved; bitpack.Pack
		// overwrites every one of bodyBytes bytes, so no zero-fill is needed (a
		// make([]byte, bodyBytes) here would alloc + zero a throwaway slice).
		out = out[:base+bodyBytes]
		bitpack.Pack(out[base:base+bodyBytes], packed, int(plan.width))
	} else if plan.exc > 0 {
		// plan.width == 0: all non-exception values share the same mantissa
		// (constant column). No pack body, but still collect exception indices
		// for the emit pass below.
		for i, v := range s {
			I := int64(math.RoundToEven(v * pe))
			if !alpRoundTrips(I, v, pe, plan.exact) {
				excIdx = append(excIdx, i)
			}
		}
	}

	// Save scratch back for the next call (avoids re-alloc when the exception
	// count is stable across batches).
	e.alpExcScratch = excIdx

	// Exception list. When there are no exceptions (the common case for clean
	// quantized telemetry) the guard skips the emit entirely. Otherwise emit
	// from the collected index scratch — no second O(n) re-scan of s.
	out = appendUvarint(out, uint64(plan.exc))
	if len(excIdx) > 0 {
		for _, i := range excIdx {
			out = appendUvarint(out, uint64(i))
			out = appendU64(out, math.Float64bits(s[i]))
		}
	}
	e.buf = out
}

// --- float32 ALP (qpackKindFloat32) ---
//
// Identical scheme to the float64 path: integer mantissas I = round(v * 10^d)
// FOR-bit-packed, plus an exception list for any value that does not reconstruct
// bit-for-bit. The mantissa is computed in float64 (exact for the integers ALP
// targets) and the round-trip is checked against the float32 bits; exception
// values are stored as 4-byte float32 patterns, not 8. The 23-bit float32
// mantissa makes the typical FOR width narrower than float64.

// alpMantissaF32 returns the integer mantissa round(v*10^d) (pe=10^d) and whether
// it reconstructs v bit-for-bit as a float32 (ie=10^-d) — i.e. whether the value
// packs as a mantissa or falls to the exception list. The mantissa is computed
// through float64 so it stays exact for values needing more than float32's 24-bit
// significand (a float32-only round would mis-round those into extra exceptions).
// Bit comparison (not ==) keeps -0.0/NaN/±Inf as exceptions. Centralises the one
// float32↔float64↔int64 chain the score / pack / exception passes all share.
func alpMantissaF32(v float32, pe, ie float64) (mant int64, exact bool) {
	I := int64(math.RoundToEven(float64(v) * pe))
	return I, math.Float32bits(float32(float64(I)*ie)) == math.Float32bits(v)
}

// alpScoreExpF32 evaluates one effective exponent d over s (float32 bit-exact
// round-trip). Mirrors alpScoreExp.
func alpScoreExpF32(s []float32, d int) (forMin int64, width uint8, exc int) {
	pe := alpPow10[d]
	ie := alpInv10[d]
	mn := int64(math.MaxInt64)
	mx := int64(math.MinInt64)
	for _, v := range s {
		I, ok := alpMantissaF32(v, pe, ie)
		if !ok {
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
	if mn > mx {
		return 0, 0, exc
	}
	return mn, uint8(bits.Len64(uint64(mx - mn))), exc
}

// alpChooseExpF32 samples ~32 values to pick the cheapest effective exponent.
func alpChooseExpF32(s []float32) int {
	const sampleN = 32
	sample := s
	if len(s) > sampleN {
		var sampleBuf [sampleN]float32
		sample = sampleBuf[:0]
		step := len(s) / sampleN
		for i := 0; i < len(s); i += step {
			sample = append(sample, s[i])
		}
	}
	bestD, bestCost := 0, math.MaxInt // platform max sentinel (int, 32-bit safe)
	for d := 0; d <= alpMaxExpSearch; d++ {
		_, w, exc := alpScoreExpF32(sample, d)
		cost := (int(w)*len(sample)+7)/8 + exc*6 // exc value is 4 bytes here
		if cost < bestCost {
			bestCost, bestD = cost, d
		}
	}
	return bestD
}

// alpPlanFloat32 chooses the exponent, scores the block, and returns a
// conservative upper bound on the encoded size (exception value = 4 bytes).
func alpPlanFloat32(s []float32) (plan alpFloatPlan, estBytes int, ok bool) {
	n := len(s)
	if n == 0 {
		return alpFloatPlan{}, 2 + 1, true
	}
	if n > alpMaxElems {
		return alpFloatPlan{}, 0, false
	}
	d := alpChooseExpF32(s)
	forMin, width, exc := alpScoreExpF32(s, d)
	if width > qpackForMaxBits {
		return alpFloatPlan{}, 0, false
	}
	if exc >= n {
		return alpFloatPlan{}, 0, false
	}
	// float32 keeps the original multiply reconstruction: its 24-bit
	// mantissa loses far less to the inverse approximation than float64's 53,
	// and leaving it alone keeps every f32 blob byte-identical.
	plan = alpFloatPlan{d: d, forMin: forMin, width: width, exc: exc}
	body := (int(width)*n + 7) / 8
	estBytes = 2 + uvarintLen(uint64(n)) + 1 + uvarintLen(zigzagEncode64(forMin)) +
		1 + body + uvarintLen(uint64(exc)) + exc*(5+4)
	return plan, estBytes, true
}

// writePackedALPFloat32Slice emits s under ALP using a pre-computed plan.
func (e *Encoder) writePackedALPFloat32Slice(s []float32, plan alpFloatPlan) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+1+10+1+(int(plan.width)*n+7)/8+10+plan.exc*(5+4))
	out = append(out, tagPackALP, qpackKindFloat32)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	// float32 always reconstructs by multiplication (alpMantissaF32 is the
	// only predicate its pack loop uses), so the exponent byte never carries
	// alpExactFlag — stamping it here would let the reader divide while the
	// writer multiplied.
	out = append(out, byte(plan.d))
	out = appendUvarint(out, zigzagEncode64(plan.forMin))
	out = append(out, plan.width)

	pe := alpPow10[plan.d]
	ie := alpInv10[plan.d]

	// Acquire exception-index scratch (shared with float64 path; reset to [:0]).
	excIdx := e.alpExcScratch[:0]

	if plan.width > 0 {
		if cap(e.alpScratch) < n {
			e.alpScratch = make([]uint64, n)
		}
		packed := e.alpScratch[:n]
		clear(packed) // exception slots stay 0 (never written in the loop)
		for i, v := range s {
			if I, ok := alpMantissaF32(v, pe, ie); ok {
				packed[i] = uint64(I - plan.forMin)
			} else {
				excIdx = append(excIdx, i) // collect exception index for emit below
			}
		}
		bodyBytes := (int(plan.width)*n + 7) / 8
		base := len(out)
		// Extend into the capacity slices.Grow already reserved; bitpack.Pack
		// overwrites every one of bodyBytes bytes, so no zero-fill is needed (a
		// make([]byte, bodyBytes) here would alloc + zero a throwaway slice).
		out = out[:base+bodyBytes]
		bitpack.Pack(out[base:base+bodyBytes], packed, int(plan.width))
	} else if plan.exc > 0 {
		// plan.width == 0: constant column with exceptions — collect indices.
		for i, v := range s {
			if _, ok := alpMantissaF32(v, pe, ie); !ok {
				excIdx = append(excIdx, i)
			}
		}
	}

	// Save scratch back for the next call.
	e.alpExcScratch = excIdx

	// Exception list: emit from collected index scratch — no second O(n) re-scan.
	out = appendUvarint(out, uint64(plan.exc))
	if len(excIdx) > 0 {
		for _, i := range excIdx {
			out = appendUvarint(out, uint64(i))
			out = appendU32(out, math.Float32bits(s[i]))
		}
	}
	e.buf = out
}

// readPackedALPFloat32Slice decodes a float32 ALP payload. The cursor (d.i) must
// point at the kind byte. Bounds mirror readPackedALPFloat64Slice; exception
// values are 4 bytes.
func (d *Decoder) readPackedALPFloat32Slice() ([]float32, error) {
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	if d.buf[d.i] != qpackKindFloat32 {
		return nil, ErrTypeMismatch
	}
	d.i++
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return nil, ErrInvalidLength
	}
	d.i += nr
	if n64 == 0 {
		return []float32{}, nil
	}
	if n64 > alpMaxElems {
		return nil, ErrInvalidLength
	}
	// Bound by the columnar row count before make([]float32, n) (see the float64
	// reader); no-op for standalone decode (colMaxLen==0).
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	rawExp := d.buf[d.i]
	exact := rawExp&alpExactFlag != 0
	expD := int(rawExp &^ alpExactFlag)
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
	rem := uint64(len(d.buf) - d.i)
	if width > 0 && n64 > rem*8/uint64(width) {
		return nil, ErrShortBuffer
	}
	n := int(n64)
	ie := alpInv10[expD]
	pe := alpPow10[expD]
	out := make([]float32, n)
	if width > 0 {
		bodyBytes := int((n64*uint64(width) + 7) / 8)
		if cap(d.deltaScratch) < n {
			d.deltaScratch = make([]uint64, n)
		}
		packed := d.deltaScratch[:n]
		bitpack.Unpack(packed, d.buf[d.i:d.i+bodyBytes], width)
		d.i += bodyBytes
		if exact { // see the float64 twin: branch hoisted out of the loop
			for i := range out {
				v := int64(packed[i])
				if (forMin > 0 && v > math.MaxInt64-forMin) || (forMin < 0 && v < math.MinInt64-forMin) {
					return nil, ErrInvalidLength
				}
				out[i] = float32(float64(v+forMin) / pe)
			}
		} else {
			for i := range out {
				v := int64(packed[i])
				if (forMin > 0 && v > math.MaxInt64-forMin) || (forMin < 0 && v < math.MinInt64-forMin) {
					return nil, ErrInvalidLength
				}
				out[i] = float32(float64(v+forMin) * ie)
			}
		}
	} else {
		fv := float32(float64(forMin) * ie)
		if exact {
			fv = float32(float64(forMin) / pe)
		}
		for i := range out {
			out[i] = fv
		}
	}
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
		if d.i+4 > len(d.buf) {
			return nil, ErrShortBuffer
		}
		out[pos64] = math.Float32frombits(readU32(d.buf[d.i:]))
		d.i += 4
	}
	return out, nil
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
	// Inside a columnar float column the element count must equal the row count;
	// gate before the make([]float64, n) below (mirrors readPackedGorillaHeader).
	// The constant (width==0) path carries no per-element body, so without this a
	// tiny header could claim alpMaxElems rows and force a ~128 MB allocation that
	// only the post-decode len(s)!=n check would reject. Standalone decode has
	// colMaxLen==0, so colLenOK is a no-op there (the alpMaxElems cap still holds).
	if !d.colLenOK(n64) {
		return nil, ErrInvalidLength
	}
	// Header: d(1), forMin zigzag-varuint, width(1).
	if d.i >= len(d.buf) {
		return nil, ErrShortBuffer
	}
	rawExp := d.buf[d.i]
	exact := rawExp&alpExactFlag != 0
	expD := int(rawExp &^ alpExactFlag)
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
	pe := alpPow10[expD]
	out := make([]float64, n)
	if width > 0 {
		bodyBytes := int((n64*uint64(width) + 7) / 8)
		// Reuse the shared transient unpack scratch (as readPackedDictUint64Slice
		// does): packed is fully written by Unpack before any read, only consumed
		// into out, never aliased into the returned slice. This branch runs only
		// when width > 0, so Unpack always writes all n slots — the width == 0
		// constant path below never touches deltaScratch, so no stale-tail leak.
		if cap(d.deltaScratch) < n {
			d.deltaScratch = make([]uint64, n)
		}
		packed := d.deltaScratch[:n]
		bitpack.Unpack(packed, d.buf[d.i:d.i+bodyBytes], width)
		d.i += bodyBytes
		// Two loops rather than a branch per element: the reconstruction op is
		// fixed for the whole block, and testing it inside the loop measured
		// +13% on decode.
		if exact {
			for i := range out {
				v := int64(packed[i])
				if (forMin > 0 && v > math.MaxInt64-forMin) || (forMin < 0 && v < math.MinInt64-forMin) {
					return nil, ErrInvalidLength
				}
				out[i] = float64(v+forMin) / pe
			}
		} else {
			for i := range out {
				v := int64(packed[i])
				if (forMin > 0 && v > math.MaxInt64-forMin) || (forMin < 0 && v < math.MinInt64-forMin) {
					return nil, ErrInvalidLength
				}
				out[i] = float64(v+forMin) * ie
			}
		}
	} else {
		fv := float64(forMin) * ie
		if exact {
			fv = float64(forMin) / pe
		}
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
