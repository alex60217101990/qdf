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

// buildDecTable fills decOut with the 4096 FSE decode entries.
// Symbol s occupies contiguous slots [cumul[s], cumul[s]+freq[s]) — required by
// the encoder's DeltaFindState = cumul[s] - freq[s] formula.
func buildDecTable(freq *[256]uint32, encOut *[256]EncSymbol, decOut *[TableSize]DecEntry) {
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
		for j := range f {
			i := cumul[s] + j
			y := f + j // ∈ [freq[s], 2*freq[s])
			nb := uint32(TableLog+1) - bitLen32(y)
			newBase := y << nb
			decOut[i] = DecEntry(s) | DecEntry(nb)<<8 | DecEntry(newBase)<<16
		}
	}

	for s := range 256 {
		f := freq[s]
		if f == 0 {
			continue
		}
		hl := bitLen32(f - 1)
		if hl > 0 {
			hl--
		}
		maxBitsOut := uint32(TableLog) - hl
		encOut[s].DeltaNbBits = (maxBitsOut << 16) - (f << maxBitsOut)
		encOut[s].DeltaFindState = int32(cumul[s]) - int32(f)
	}
}

// initBitReader initializes a BACKWARD bit reader over a forward-written
// payload. The LAST byte of payload is the encoder's close() byte: end mark at
// its highest set bit, data bits below it. Bits are consumed from the TOP of
// the container (newest-written first); refill prepends older (lower-address)
// bytes below via container<<8 | byte.
// Returns (bitContainer, bitPos, pos, err); pos is the next byte to read,
// moving toward -1.
func initBitReader(payload []byte) (bitContainer uint64, bitPos uint, pos int, err error) {
	pos = len(payload) - 1
	if pos < 0 {
		return 0, 0, -1, nil
	}
	closeByte := payload[pos]
	if closeByte == 0 {
		return 0, 0, 0, ErrCorrupt
	}
	pos--
	bitPos = uint(bits.Len8(closeByte)) - 1
	bitContainer = uint64(closeByte) & ((1 << bitPos) - 1)
	// Pre-fill remaining capacity.
	for bitPos <= 56 && pos >= 0 {
		bitContainer = bitContainer<<8 | uint64(payload[pos])
		bitPos += 8
		pos--
	}
	return bitContainer, bitPos, pos, nil
}

// decodeStream decompresses a single-stream tANS blob.
// src = [4-byte LE final state][compressed bytes].
func decodeStream(src []byte, freq *[256]uint32, n int) ([]byte, error) {
	if len(src) < 4 {
		return nil, ErrCorrupt
	}
	state := uint32(binary.LittleEndian.Uint32(src[:4]))
	payload := src[4:]

	var decTable [TableSize]DecEntry
	var encTable [256]EncSymbol
	buildDecTable(freq, &encTable, &decTable)

	out := make([]byte, n)

	bitContainer, bitPos, pos, err := initBitReader(payload)
	if err != nil {
		return nil, err
	}

	refill := func() {
		for bitPos <= 56 && pos >= 0 {
			bitContainer = bitContainer<<8 | uint64(payload[pos])
			bitPos += 8
			pos--
		}
	}

	for i := range n {
		e := decTable[state&(TableSize-1)]
		out[i] = e.Symbol()
		nb := uint(e.NbBits())
		bitPos -= nb
		state = e.NewBase() + uint32(bitContainer>>bitPos)&uint32((1<<nb)-1)
		if bitPos <= 24 {
			refill()
		}
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
	var encTable [256]EncSymbol
	buildDecTable(freq, &encTable, &decTable)

	var xs [4]uint32
	for k := range 4 {
		xs[k] = binary.LittleEndian.Uint32(src[k*4:])
	}
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

	var regions [4][]byte
	off := 0
	for k := range 4 {
		regions[k] = src[off : off+lens[k]]
		off += lens[k]
	}

	out := make([]byte, n)
	var bcs [4]uint64
	var bps [4]uint
	var rpos [4]int

	for k := range 4 {
		var err error
		bcs[k], bps[k], rpos[k], err = initBitReader(regions[k])
		if err != nil {
			return nil, err
		}
	}

	refill := func(k int) {
		for bps[k] <= 56 && rpos[k] >= 0 {
			bcs[k] = bcs[k]<<8 | uint64(regions[k][rpos[k]])
			bps[k] += 8
			rpos[k]--
		}
	}

	i := 0
	for ; i+3 < n; i += 4 {
		e0 := decTable[xs[0]&(TableSize-1)]
		e1 := decTable[xs[1]&(TableSize-1)]
		e2 := decTable[xs[2]&(TableSize-1)]
		e3 := decTable[xs[3]&(TableSize-1)]
		out[i] = e0.Symbol()
		out[i+1] = e1.Symbol()
		out[i+2] = e2.Symbol()
		out[i+3] = e3.Symbol()
		nb0 := uint(e0.NbBits())
		nb1 := uint(e1.NbBits())
		nb2 := uint(e2.NbBits())
		nb3 := uint(e3.NbBits())
		bps[0] -= nb0
		bps[1] -= nb1
		bps[2] -= nb2
		bps[3] -= nb3
		xs[0] = e0.NewBase() + uint32(bcs[0]>>bps[0])&uint32((1<<nb0)-1)
		xs[1] = e1.NewBase() + uint32(bcs[1]>>bps[1])&uint32((1<<nb1)-1)
		xs[2] = e2.NewBase() + uint32(bcs[2]>>bps[2])&uint32((1<<nb2)-1)
		xs[3] = e3.NewBase() + uint32(bcs[3]>>bps[3])&uint32((1<<nb3)-1)
		if bps[0] <= 24 {
			refill(0)
		}
		if bps[1] <= 24 {
			refill(1)
		}
		if bps[2] <= 24 {
			refill(2)
		}
		if bps[3] <= 24 {
			refill(3)
		}
	}
	for ; i < n; i++ {
		k := i & 3
		e := decTable[xs[k]&(TableSize-1)]
		out[i] = e.Symbol()
		nb := uint(e.NbBits())
		bps[k] -= nb
		xs[k] = e.NewBase() + uint32(bcs[k]>>bps[k])&uint32((1<<nb)-1)
		if bps[k] <= 24 {
			refill(k)
		}
	}
	return out, nil
}
