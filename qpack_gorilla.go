package qdf

import (
	"math"
	"math/bits"
	"slices"
)

// Gorilla XOR codec for float slices, from
// "Gorilla: A Fast, Scalable, In-Memory Time Series Database" (FB, VLDB
// 2015). For slow-varying floats (sensor reads, metrics, monotonic
// timestamps cast to float) it crushes the per-sample cost from 64 bits
// down to ~1-12 bits without lossy compression: NaN and ±Inf survive
// unchanged because the codec operates on the IEEE-754 bit pattern.
//
// Per sample:
//   - xor = bits(curr) ^ bits(prev)
//   - if xor == 0: write a single 0 bit
//   - else: write a 1 bit, then either
//       0  + meaningful bits inside the previous lz/tz window, or
//       1  + 5-bit lz + 6-bit (mbLen-1) + meaningful bits.
//
// Bit ordering is MSB-first within each byte to match the original paper
// and make the bit-stream comparable to other Gorilla implementations.

// bitWriter / bitReader are MSB-first scratchpads. They are local to the
// gorilla codec and do not allocate on the hot path.
type bitWriter struct {
	buf   []byte
	nbits int // total bits written, independent of buf's base offset
	cur   byte
	used  uint8 // bits used in cur (0..7)
}

func (bw *bitWriter) writeBit(v bool) {
	if v {
		bw.cur |= 1 << (7 - bw.used)
	}
	bw.used++
	bw.nbits++
	if bw.used == 8 {
		bw.buf = append(bw.buf, bw.cur)
		bw.cur = 0
		bw.used = 0
	}
}

func (bw *bitWriter) writeBits(v uint64, count uint8) {
	for i := count; i > 0; i-- {
		bw.writeBit((v>>(i-1))&1 == 1)
	}
}

// flush finalises any partial byte. Returns the total bit count written. The
// count is tracked incrementally (nbits) rather than derived from len(buf) so
// the writer can append directly into a caller buffer that already holds a
// header prefix.
func (bw *bitWriter) flush() int {
	total := bw.nbits
	if bw.used > 0 {
		bw.buf = append(bw.buf, bw.cur)
		bw.cur = 0
		bw.used = 0
	}
	return total
}

// finishGorillaBody finalises a Gorilla body that was written in-place into out
// starting at bodyStart (no separate scratch buffer), prepending its
// uvarint(totalBits) length via a single memmove of the body. This avoids the
// per-call scratch []byte that grew from nil and the final copy into out.
func finishGorillaBody(out []byte, bodyStart, totalBits int) []byte {
	bodyEnd := len(out)
	ul := uvarintLen(uint64(totalBits))
	out = slices.Grow(out, ul)
	out = out[:bodyEnd+ul]
	copy(out[bodyStart+ul:], out[bodyStart:bodyEnd])
	// Write the uvarint into the gap we just opened; cap is guaranteed, so
	// appendUvarint fills out[bodyStart:bodyStart+ul] without reallocating.
	_ = appendUvarint(out[bodyStart:bodyStart], uint64(totalBits))
	return out
}

type bitReader struct {
	buf  []byte
	pos  int   // byte index
	used uint8 // bits consumed in buf[pos] (0..7)
}

func (br *bitReader) readBit() (bool, bool) {
	if br.pos >= len(br.buf) {
		return false, false
	}
	v := (br.buf[br.pos] >> (7 - br.used)) & 1
	br.used++
	if br.used == 8 {
		br.used = 0
		br.pos++
	}
	return v == 1, true
}

func (br *bitReader) readBits(count uint8) (uint64, bool) {
	var out uint64
	for range count {
		b, ok := br.readBit()
		if !ok {
			return 0, false
		}
		out = (out << 1) | btoU64(b)
	}
	return out, true
}

func btoU64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// writePackedGorillaFloat64Slice emits a Gorilla XOR-coded []float64.
// n=0 writes only the tag/kind/n; n=1 also writes the first value but no
// XOR body.
func (e *Encoder) writePackedGorillaFloat64Slice(s []float64) {
	e.writeHeader()
	n := len(s)
	// Reserve generously: 4 byte body per sample is a comfortable upper
	// bound for any non-pathological input; the slow path appends if it
	// exceeds.
	out := slices.Grow(e.buf, 2+10+8+10+4*n)
	out = append(out, tagPackGorilla, qpackKindFloat64)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	first := math.Float64bits(s[0])
	out = appendU64(out, first)
	if n == 1 {
		out = appendUvarint(out, 0)
		e.buf = out
		return
	}
	bodyStart := len(out)
	bw := bitWriter{buf: out}
	prev := first
	// prevLZ == 64 marks "no previous window".
	prevLZ := uint8(64)
	prevTZ := uint8(0)
	for i := 1; i < n; i++ {
		cur := math.Float64bits(s[i])
		x := cur ^ prev
		if x == 0 {
			bw.writeBit(false)
			prev = cur
			continue
		}
		bw.writeBit(true)
		lz := uint8(bits.LeadingZeros64(x))
		tz := uint8(bits.TrailingZeros64(x))
		if lz > 31 {
			lz = 31
		}
		if prevLZ != 64 && lz >= prevLZ && tz >= prevTZ {
			bw.writeBit(false)
			mb := 64 - int(prevLZ) - int(prevTZ)
			bw.writeBits(x>>prevTZ, uint8(mb))
		} else {
			bw.writeBit(true)
			bw.writeBits(uint64(lz), 5)
			mb := 64 - int(lz) - int(tz)
			bw.writeBits(uint64(mb-1), 6)
			bw.writeBits(x>>tz, uint8(mb))
			prevLZ = lz
			prevTZ = tz
		}
		prev = cur
	}
	totalBits := bw.flush()
	out = finishGorillaBody(bw.buf, bodyStart, totalBits)
	e.buf = out
}

func (e *Encoder) writePackedGorillaFloat32Slice(s []float32) {
	e.writeHeader()
	n := len(s)
	out := slices.Grow(e.buf, 2+10+4+10+2*n)
	out = append(out, tagPackGorilla, qpackKindFloat32)
	out = appendUvarint(out, uint64(n))
	if n == 0 {
		e.buf = out
		return
	}
	first := math.Float32bits(s[0])
	out = appendU32(out, first)
	if n == 1 {
		out = appendUvarint(out, 0)
		e.buf = out
		return
	}
	bodyStart := len(out)
	bw := bitWriter{buf: out}
	prev := first
	prevLZ := uint8(32)
	prevTZ := uint8(0)
	for i := 1; i < n; i++ {
		cur := math.Float32bits(s[i])
		x := cur ^ prev
		if x == 0 {
			bw.writeBit(false)
			prev = cur
			continue
		}
		bw.writeBit(true)
		lz := uint8(bits.LeadingZeros32(x))
		tz := uint8(bits.TrailingZeros32(x))
		if lz > 15 {
			lz = 15
		}
		if prevLZ != 32 && lz >= prevLZ && tz >= prevTZ {
			bw.writeBit(false)
			mb := 32 - int(prevLZ) - int(prevTZ)
			bw.writeBits(uint64(x>>prevTZ), uint8(mb))
		} else {
			bw.writeBit(true)
			bw.writeBits(uint64(lz), 4)
			mb := 32 - int(lz) - int(tz)
			bw.writeBits(uint64(mb-1), 5)
			bw.writeBits(uint64(x>>tz), uint8(mb))
			prevLZ = lz
			prevTZ = tz
		}
		prev = cur
	}
	totalBits := bw.flush()
	out = finishGorillaBody(bw.buf, bodyStart, totalBits)
	e.buf = out
}

// readPackedGorillaHeader consumes kind, n, firstVal, numBits, body.
// The tag itself must already be consumed. The numBits varuint is
// validated and consumed but not returned — it is used only to compute
// bodyBytes = ceil(numBits/8) and slice the body from the wire buffer;
// the bitReader is then bounded by that byte-sliced body length, not by
// a tracked bit count. Surfacing numBits made it unused (caught by unparam).
func (d *Decoder) readPackedGorillaHeader(expectKind byte) (n int, firstU64 uint64, body []byte, err error) {
	if d.i >= len(d.buf) {
		return 0, 0, nil, ErrShortBuffer
	}
	k := d.buf[d.i]
	d.i++
	if k != expectKind {
		return 0, 0, nil, ErrTypeMismatch
	}
	n64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	// Bound the element count before make([]float, n). In a columnar column
	// it must equal the row count (colLenOK); standalone, every element past
	// the first costs >=1 control bit in the body, so a valid n cannot exceed
	// the remaining bits — this stops a tiny payload from claiming billions of
	// elements and triggering an OOM (the body-bytes check below does not
	// bound n, which is read independently).
	if !d.colLenOK(n64) {
		return 0, 0, nil, ErrInvalidLength
	}
	if rem := uint64(len(d.buf) - d.i); n64 > rem*8+1 {
		return 0, 0, nil, ErrShortBuffer
	}
	n = int(n64)
	if n == 0 {
		return n, 0, nil, nil
	}
	w := qpackRawWidthBytes(k)
	if d.i+w > len(d.buf) {
		return 0, 0, nil, ErrShortBuffer
	}
	switch w {
	case 4:
		firstU64 = uint64(readU32(d.buf[d.i:]))
	case 8:
		firstU64 = readU64(d.buf[d.i:])
	default:
		return 0, 0, nil, ErrBadTag
	}
	d.i += w
	if n == 1 {
		// numBits must be 0 but still encoded.
		nb64, nr := readUvarint(d.buf[d.i:])
		if nr <= 0 {
			return 0, 0, nil, ErrInvalidLength
		}
		d.i += nr
		if nb64 != 0 {
			return 0, 0, nil, ErrBadTag
		}
		return n, firstU64, nil, nil
	}
	nb64, nr := readUvarint(d.buf[d.i:])
	if nr <= 0 {
		return 0, 0, nil, ErrInvalidLength
	}
	d.i += nr
	// uint64 bounds check before signed cast.
	rem := uint64(len(d.buf) - d.i)
	if nb64 > rem*8 {
		return 0, 0, nil, ErrShortBuffer
	}
	bodyBytes := int((nb64 + 7) / 8)
	body = d.buf[d.i : d.i+bodyBytes]
	d.i += bodyBytes
	return n, firstU64, body, nil
}

func (d *Decoder) readPackedGorillaFloat64Slice() ([]float64, error) {
	n, firstU, body, err := d.readPackedGorillaHeader(qpackKindFloat64)
	if err != nil {
		return nil, err
	}
	out := make([]float64, n)
	if n == 0 {
		return out, nil
	}
	out[0] = math.Float64frombits(firstU)
	if n == 1 {
		return out, nil
	}
	br := bitReader{buf: body}
	prev := firstU
	prevLZ := uint8(64)
	prevTZ := uint8(0)
	for i := 1; i < n; i++ {
		ctl, ok := br.readBit()
		if !ok {
			return nil, ErrShortBuffer
		}
		if !ctl {
			out[i] = math.Float64frombits(prev)
			continue
		}
		ctl2, ok := br.readBit()
		if !ok {
			return nil, ErrShortBuffer
		}
		var x uint64
		if !ctl2 {
			mb := 64 - int(prevLZ) - int(prevTZ)
			mbBits, ok := br.readBits(uint8(mb))
			if !ok {
				return nil, ErrShortBuffer
			}
			x = mbBits << prevTZ
		} else {
			lz64, ok := br.readBits(5)
			if !ok {
				return nil, ErrShortBuffer
			}
			mbLen64, ok := br.readBits(6)
			if !ok {
				return nil, ErrShortBuffer
			}
			mb := mbLen64 + 1
			// A well-formed window always has lz + mb <= 64 (lz + mb + tz = 64).
			// Reject a hostile stream where it exceeds the word width before the
			// unsigned `64 - lz64 - mb` underflows: that would make the shift
			// below collapse to 0 (silent wrong value) and poison prevTZ for the
			// rest of the slice.
			if lz64+mb > 64 {
				return nil, ErrInvalidLength
			}
			tz := 64 - lz64 - mb
			mbBits, ok := br.readBits(uint8(mb))
			if !ok {
				return nil, ErrShortBuffer
			}
			x = mbBits << tz
			prevLZ = uint8(lz64)
			prevTZ = uint8(tz)
		}
		cur := prev ^ x
		out[i] = math.Float64frombits(cur)
		prev = cur
	}
	return out, nil
}

func (d *Decoder) readPackedGorillaFloat32Slice() ([]float32, error) {
	n, firstU, body, err := d.readPackedGorillaHeader(qpackKindFloat32)
	if err != nil {
		return nil, err
	}
	out := make([]float32, n)
	if n == 0 {
		return out, nil
	}
	out[0] = math.Float32frombits(uint32(firstU))
	if n == 1 {
		return out, nil
	}
	br := bitReader{buf: body}
	prev := uint32(firstU)
	prevLZ := uint8(32)
	prevTZ := uint8(0)
	for i := 1; i < n; i++ {
		ctl, ok := br.readBit()
		if !ok {
			return nil, ErrShortBuffer
		}
		if !ctl {
			out[i] = math.Float32frombits(prev)
			continue
		}
		ctl2, ok := br.readBit()
		if !ok {
			return nil, ErrShortBuffer
		}
		var x uint32
		if !ctl2 {
			mb := 32 - int(prevLZ) - int(prevTZ)
			mbBits, ok := br.readBits(uint8(mb))
			if !ok {
				return nil, ErrShortBuffer
			}
			x = uint32(mbBits) << prevTZ
		} else {
			lz64, ok := br.readBits(4)
			if !ok {
				return nil, ErrShortBuffer
			}
			mbLen64, ok := br.readBits(5)
			if !ok {
				return nil, ErrShortBuffer
			}
			mb := mbLen64 + 1
			// As in the float64 path: a valid window has lz + mb <= 32; reject a
			// hostile stream before the unsigned subtraction underflows.
			if lz64+mb > 32 {
				return nil, ErrInvalidLength
			}
			tz := 32 - lz64 - mb
			mbBits, ok := br.readBits(uint8(mb))
			if !ok {
				return nil, ErrShortBuffer
			}
			x = uint32(mbBits) << tz
			prevLZ = uint8(lz64)
			prevTZ = uint8(tz)
		}
		cur := prev ^ x
		out[i] = math.Float32frombits(cur)
		prev = cur
	}
	return out, nil
}
