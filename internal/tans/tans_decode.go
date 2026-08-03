package tans

import (
	"encoding/binary"
	"math/bits"
)

// DecEntry packs symbol, nbBits, and newBase into 4 bytes so the 4096-entry
// decode table (16 KB) fits in L1 cache.
//
//	bits  [7:0]  = symbol byte
//	bits [15:8]  = nbBits
//	bits [31:16] = newBase  (= nextStateNumber << nbBits, ∈ [TableSize, 2*TableSize))
type DecEntry uint32

func (e DecEntry) Symbol() byte    { return byte(e) }
func (e DecEntry) NbBits() uint8   { return uint8(e >> 8) }
func (e DecEntry) NewBase() uint32 { return uint32(e >> 16) }

// buildDecTable fills decOut with the TableSize FSE decode entries.
// Symbol s occupies contiguous slots [cumul[s], cumul[s]+freq[s]) — required by
// the encoder's DeltaFindState = cumul[s] - freq[s] formula.
func buildDecTable(freq *[256]uint32, decOut *[TableSize]DecEntry) {
	var cumul [256]uint32
	var c uint32
	for s := range 256 {
		cumul[s] = c
		c += freq[s]
	}

	for s := range 256 {
		f := freq[s]
		if f == 0 {
			continue
		}
		base := cumul[s]
		for j := range f {
			y := f + j // ∈ [freq[s], 2*freq[s])
			nb := uint32(TableLog+1) - bitLen32(y)
			newBase := y << nb
			decOut[base+j] = DecEntry(s) | DecEntry(nb)<<8 | DecEntry(newBase)<<16
		}
	}
}

// The decoder mirrors zstd's BIT_DStream: the 64-bit container always holds
// the 8 payload bytes at [pos, pos+8) loaded little-endian, and `consumed`
// counts bits already eaten from the TOP of that window (the top of the
// window = the newest-written bits, which decode first).
//
//	read nb bits:  v = (container << consumed) >> (64 - nb)
//	reload:        pos -= consumed/8; consumed &= 7; container = LE64(payload[pos:])
//	               (clamped at pos==0 near the stream start — `consumed` then
//	               keeps growing instead; Go shifts >= 64 yield 0, so reads
//	               past the front of a corrupt stream return zero bits and
//	               never panic.)
//
// Payloads shorter than 8 bytes are right-aligned into a zero-padded 8-byte
// buffer by the caller so the window load is always in bounds.

// initBitReader positions the reader on the close() byte at the END of
// payload. len(payload) must be >= 8. Returns (container, consumed, pos, err).
func initBitReader(payload []byte) (container uint64, consumed uint, pos int, err error) {
	last := payload[len(payload)-1]
	if last == 0 {
		return 0, 0, 0, ErrCorrupt // no end mark
	}
	pos = len(payload) - 8
	container = binary.LittleEndian.Uint64(payload[pos:])
	// End mark sits at bit 56 + (Len8(last)-1); everything above it is consumed.
	consumed = uint(9 - bits.Len8(last))
	return container, consumed, pos, nil
}

// padPayload right-aligns a short payload into pad, returning an 8-byte view.
func padPayload(payload []byte, pad *[8]byte) []byte {
	if len(payload) >= 8 {
		return payload
	}
	copy(pad[8-len(payload):], payload)
	return pad[:]
}

// decodeStream decompresses a single-stream tANS blob.
// src = [4-byte LE final state][compressed bytes].
func decodeStream(src []byte, freq *[256]uint32, n int) ([]byte, error) {
	if len(src) < 4 {
		return nil, ErrCorrupt
	}
	state := binary.LittleEndian.Uint32(src[:4])
	if len(src) == 4 {
		return nil, ErrCorrupt // n > 0 always emits at least the close byte
	}
	var pad [8]byte
	payload := padPayload(src[4:], &pad)

	var decTable [TableSize]DecEntry
	buildDecTable(freq, &decTable)

	container, consumed, pos, err := initBitReader(payload)
	if err != nil {
		return nil, err
	}

	out := make([]byte, n)

	// Unrolled x4: init leaves consumed <= 8, each read adds <= TableLog=12,
	// so 8 + 4*12 = 56 <= 63 between reloads. Reads never need shifts > 63
	// on valid streams; corrupt streams degrade to zero bits (Go-defined).
	i := 0
	for ; i+3 < n; i += 4 {
		e0 := decTable[state&(TableSize-1)]
		nb0 := uint(e0.NbBits())
		state = e0.NewBase() + uint32((container<<consumed)>>(64-nb0))
		consumed += nb0

		e1 := decTable[state&(TableSize-1)]
		nb1 := uint(e1.NbBits())
		state = e1.NewBase() + uint32((container<<consumed)>>(64-nb1))
		consumed += nb1

		e2 := decTable[state&(TableSize-1)]
		nb2 := uint(e2.NbBits())
		state = e2.NewBase() + uint32((container<<consumed)>>(64-nb2))
		consumed += nb2

		e3 := decTable[state&(TableSize-1)]
		nb3 := uint(e3.NbBits())
		state = e3.NewBase() + uint32((container<<consumed)>>(64-nb3))
		consumed += nb3

		out[i] = e0.Symbol()
		out[i+1] = e1.Symbol()
		out[i+2] = e2.Symbol()
		out[i+3] = e3.Symbol()

		pos -= int(consumed >> 3)
		consumed &= 7
		if pos < 0 {
			consumed += uint(-pos) << 3
			pos = 0
		}
		container = binary.LittleEndian.Uint64(payload[pos:])
	}
	for ; i < n; i++ {
		e := decTable[state&(TableSize-1)]
		nb := uint(e.NbBits())
		state = e.NewBase() + uint32((container<<consumed)>>(64-nb))
		consumed += nb
		out[i] = e.Symbol()

		pos -= int(consumed >> 3)
		consumed &= 7
		if pos < 0 {
			consumed += uint(-pos) << 3
			pos = 0
		}
		container = binary.LittleEndian.Uint64(payload[pos:])
	}

	// A valid stream ends exactly where encoding began.
	if state != TableSize {
		return nil, ErrCorrupt
	}
	return out, nil
}

// decodeInterleaved4 decompresses a 4-stream interleaved tANS blob.
// src = [4×uint32 LE states][3×uvarint substream lengths][substream 0..3].
func decodeInterleaved4(src []byte, freq *[256]uint32, n int) ([]byte, error) {
	if len(src) < 16 {
		return nil, ErrCorrupt
	}

	var decTable [TableSize]DecEntry
	buildDecTable(freq, &decTable)

	x0 := binary.LittleEndian.Uint32(src[0:])
	x1 := binary.LittleEndian.Uint32(src[4:])
	x2 := binary.LittleEndian.Uint32(src[8:])
	x3 := binary.LittleEndian.Uint32(src[12:])
	src = src[16:]

	var lens [4]int
	var total uint64
	for k := range 3 {
		v, used := uvarint(src)
		if used <= 0 || v > uint64(len(src)) {
			return nil, ErrCorrupt
		}
		src = src[used:]
		lens[k] = int(v)
		total += v
	}
	if total > uint64(len(src)) {
		return nil, ErrCorrupt
	}
	lens[3] = len(src) - int(total)

	var pads [4][8]byte
	var regions [4][]byte
	off := 0
	for k := range 4 {
		regions[k] = padPayload(src[off:off+lens[k]], &pads[k])
		off += lens[k]
	}
	r0, r1, r2, r3 := regions[0], regions[1], regions[2], regions[3]

	c0, u0, p0, err := initBitReader(r0)
	if err != nil {
		return nil, err
	}
	c1, u1, p1, err := initBitReader(r1)
	if err != nil {
		return nil, err
	}
	c2, u2, p2, err := initBitReader(r2)
	if err != nil {
		return nil, err
	}
	c3, u3, p3, err := initBitReader(r3)
	if err != nil {
		return nil, err
	}

	out := make([]byte, n)

	// Blocked main loop: 4 rows (16 symbols) per reload. Entering a block,
	// consumed <= 8; four reads/lane add <= 4*TableLog = 48, so consumed
	// stays <= 56 <= 63. The four serial state chains are independent, so
	// the CPU overlaps them (ILP).
	i := 0
	for ; i+15 < n; i += 16 {
		e0 := decTable[x0&(TableSize-1)]
		e1 := decTable[x1&(TableSize-1)]
		e2 := decTable[x2&(TableSize-1)]
		e3 := decTable[x3&(TableSize-1)]
		nb0 := uint(e0.NbBits())
		nb1 := uint(e1.NbBits())
		nb2 := uint(e2.NbBits())
		nb3 := uint(e3.NbBits())
		x0 = e0.NewBase() + uint32((c0<<u0)>>(64-nb0))
		x1 = e1.NewBase() + uint32((c1<<u1)>>(64-nb1))
		x2 = e2.NewBase() + uint32((c2<<u2)>>(64-nb2))
		x3 = e3.NewBase() + uint32((c3<<u3)>>(64-nb3))
		u0 += nb0
		u1 += nb1
		u2 += nb2
		u3 += nb3
		out[i] = e0.Symbol()
		out[i+1] = e1.Symbol()
		out[i+2] = e2.Symbol()
		out[i+3] = e3.Symbol()

		e0 = decTable[x0&(TableSize-1)]
		e1 = decTable[x1&(TableSize-1)]
		e2 = decTable[x2&(TableSize-1)]
		e3 = decTable[x3&(TableSize-1)]
		nb0 = uint(e0.NbBits())
		nb1 = uint(e1.NbBits())
		nb2 = uint(e2.NbBits())
		nb3 = uint(e3.NbBits())
		x0 = e0.NewBase() + uint32((c0<<u0)>>(64-nb0))
		x1 = e1.NewBase() + uint32((c1<<u1)>>(64-nb1))
		x2 = e2.NewBase() + uint32((c2<<u2)>>(64-nb2))
		x3 = e3.NewBase() + uint32((c3<<u3)>>(64-nb3))
		u0 += nb0
		u1 += nb1
		u2 += nb2
		u3 += nb3
		out[i+4] = e0.Symbol()
		out[i+5] = e1.Symbol()
		out[i+6] = e2.Symbol()
		out[i+7] = e3.Symbol()

		e0 = decTable[x0&(TableSize-1)]
		e1 = decTable[x1&(TableSize-1)]
		e2 = decTable[x2&(TableSize-1)]
		e3 = decTable[x3&(TableSize-1)]
		nb0 = uint(e0.NbBits())
		nb1 = uint(e1.NbBits())
		nb2 = uint(e2.NbBits())
		nb3 = uint(e3.NbBits())
		x0 = e0.NewBase() + uint32((c0<<u0)>>(64-nb0))
		x1 = e1.NewBase() + uint32((c1<<u1)>>(64-nb1))
		x2 = e2.NewBase() + uint32((c2<<u2)>>(64-nb2))
		x3 = e3.NewBase() + uint32((c3<<u3)>>(64-nb3))
		u0 += nb0
		u1 += nb1
		u2 += nb2
		u3 += nb3
		out[i+8] = e0.Symbol()
		out[i+9] = e1.Symbol()
		out[i+10] = e2.Symbol()
		out[i+11] = e3.Symbol()

		e0 = decTable[x0&(TableSize-1)]
		e1 = decTable[x1&(TableSize-1)]
		e2 = decTable[x2&(TableSize-1)]
		e3 = decTable[x3&(TableSize-1)]
		nb0 = uint(e0.NbBits())
		nb1 = uint(e1.NbBits())
		nb2 = uint(e2.NbBits())
		nb3 = uint(e3.NbBits())
		x0 = e0.NewBase() + uint32((c0<<u0)>>(64-nb0))
		x1 = e1.NewBase() + uint32((c1<<u1)>>(64-nb1))
		x2 = e2.NewBase() + uint32((c2<<u2)>>(64-nb2))
		x3 = e3.NewBase() + uint32((c3<<u3)>>(64-nb3))
		u0 += nb0
		u1 += nb1
		u2 += nb2
		u3 += nb3
		out[i+12] = e0.Symbol()
		out[i+13] = e1.Symbol()
		out[i+14] = e2.Symbol()
		out[i+15] = e3.Symbol()

		p0 -= int(u0 >> 3)
		u0 &= 7
		if p0 < 0 {
			u0 += uint(-p0) << 3
			p0 = 0
		}
		c0 = binary.LittleEndian.Uint64(r0[p0:])

		p1 -= int(u1 >> 3)
		u1 &= 7
		if p1 < 0 {
			u1 += uint(-p1) << 3
			p1 = 0
		}
		c1 = binary.LittleEndian.Uint64(r1[p1:])

		p2 -= int(u2 >> 3)
		u2 &= 7
		if p2 < 0 {
			u2 += uint(-p2) << 3
			p2 = 0
		}
		c2 = binary.LittleEndian.Uint64(r2[p2:])

		p3 -= int(u3 >> 3)
		u3 &= 7
		if p3 < 0 {
			u3 += uint(-p3) << 3
			p3 = 0
		}
		c3 = binary.LittleEndian.Uint64(r3[p3:])
	}

	// Remainder rows (< 4): one reload per lane per row.
	for ; i+3 < n; i += 4 {
		e0 := decTable[x0&(TableSize-1)]
		e1 := decTable[x1&(TableSize-1)]
		e2 := decTable[x2&(TableSize-1)]
		e3 := decTable[x3&(TableSize-1)]
		nb0 := uint(e0.NbBits())
		nb1 := uint(e1.NbBits())
		nb2 := uint(e2.NbBits())
		nb3 := uint(e3.NbBits())
		x0 = e0.NewBase() + uint32((c0<<u0)>>(64-nb0))
		x1 = e1.NewBase() + uint32((c1<<u1)>>(64-nb1))
		x2 = e2.NewBase() + uint32((c2<<u2)>>(64-nb2))
		x3 = e3.NewBase() + uint32((c3<<u3)>>(64-nb3))
		u0 += nb0
		u1 += nb1
		u2 += nb2
		u3 += nb3
		out[i] = e0.Symbol()
		out[i+1] = e1.Symbol()
		out[i+2] = e2.Symbol()
		out[i+3] = e3.Symbol()

		p0 -= int(u0 >> 3)
		u0 &= 7
		if p0 < 0 {
			u0 += uint(-p0) << 3
			p0 = 0
		}
		c0 = binary.LittleEndian.Uint64(r0[p0:])

		p1 -= int(u1 >> 3)
		u1 &= 7
		if p1 < 0 {
			u1 += uint(-p1) << 3
			p1 = 0
		}
		c1 = binary.LittleEndian.Uint64(r1[p1:])

		p2 -= int(u2 >> 3)
		u2 &= 7
		if p2 < 0 {
			u2 += uint(-p2) << 3
			p2 = 0
		}
		c2 = binary.LittleEndian.Uint64(r2[p2:])

		p3 -= int(u3 >> 3)
		u3 &= 7
		if p3 < 0 {
			u3 += uint(-p3) << 3
			p3 = 0
		}
		c3 = binary.LittleEndian.Uint64(r3[p3:])
	}

	// Tail: <= 3 symbols, one per distinct lane, each reads <= 12 more bits
	// on top of consumed <= 20 — no reload needed.
	xs := [4]uint32{x0, x1, x2, x3}
	cs := [4]uint64{c0, c1, c2, c3}
	us := [4]uint{u0, u1, u2, u3}
	for ; i < n; i++ {
		k := i & 3
		e := decTable[xs[k]&(TableSize-1)]
		nb := uint(e.NbBits())
		xs[k] = e.NewBase() + uint32((cs[k]<<us[k])>>(64-nb))
		us[k] += nb
		out[i] = e.Symbol()
	}

	// Every lane of a valid stream ends exactly where encoding began.
	if xs[0] != TableSize || xs[1] != TableSize || xs[2] != TableSize || xs[3] != TableSize {
		return nil, ErrCorrupt
	}
	return out, nil
}
