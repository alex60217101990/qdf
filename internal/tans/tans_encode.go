package tans

import (
	"encoding/binary"
	"math/bits"

	"github.com/alex60217101990/qdf/internal/bufpool"
)

// EncSymbol holds the two encode constants for one symbol in the FSE encode table.
type EncSymbol struct {
	DeltaFindState int32
	DeltaNbBits    uint32
}

// buildEncTable fills encOut from a normalized frequency table.
func buildEncTable(freq *[256]uint32, encOut *[256]EncSymbol) {
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
		hl := bitLen32(f - 1)
		if hl > 0 {
			hl--
		}
		maxBitsOut := uint32(TableLog) - hl
		encOut[s].DeltaNbBits = (maxBitsOut << 16) - (f << maxBitsOut)
		encOut[s].DeltaFindState = int32(cumul[s]) - int32(f)
	}
}

// bitLen32 returns floor(log2(v))+1, or 0 for v==0. Uses math/bits.Len32.
func bitLen32(v uint32) uint32 {
	return uint32(bits.Len32(v))
}

// bitWriter accumulates bits at HIGH positions (canonical zstd FSE convention)
// and flushes the whole 64-bit container FORWARD with a single 8-byte store.
//
// Convention (must match decoder in tans_decode.go):
//   - addBits:   container |= (v&mask) << bitPos. New bits stack ABOVE old ones,
//     so the low bits are final once written and can be flushed immediately.
//   - flushBits: stores the container as 8 LE bytes at pos, advances pos by the
//     number of COMPLETE bytes, keeps the 0..7 remainder bits. The bytes above
//     the remainder are garbage on the wire until overwritten by the next
//     store — the buffer must always have >= 8 bytes of slack past pos.
//   - close:     inserts a 1-bit end mark at position bitPos, writes the final
//     byte, returns total payload length.
//
// End-mark layout in the close byte (= LAST byte, first byte the decoder reads):
//
//	bits [0 .. bitPos-1] = data (oldest remaining bits at lowest positions)
//	bit  [bitPos]        = end mark = 1
//	bits [bitPos+1 .. 7] = padding zeros
//
// Decoder finds data extent: endMarkPos = bits.Len8(closeByte)-1, then reads
// the stream BACKWARD (newest bits first, from the top).
type bitWriter struct {
	buf          []byte
	pos          int // write cursor, starts at 0, increments
	bitContainer uint64
	bitPos       uint // bits valid in bitContainer (0..63; <=55 before addBits)
}

func newBitWriter(buf []byte) bitWriter {
	return bitWriter{buf: buf}
}

func (bw *bitWriter) addBits(v uint64, nb uint) {
	bw.bitContainer |= (v & (1<<nb - 1)) << bw.bitPos
	bw.bitPos += nb
}

// flushBits writes the container with one 8-byte store and advances by the
// complete bytes. Leaves bitPos in [0,7].
func (bw *bitWriter) flushBits() {
	binary.LittleEndian.PutUint64(bw.buf[bw.pos:], bw.bitContainer)
	nb := bw.bitPos >> 3
	bw.pos += int(nb)
	bw.bitPos &= 7
	bw.bitContainer >>= nb << 3
}

// close inserts a 1-bit end mark, writes the final byte, and returns the
// payload length. Must be called after flushBits (bitPos in [0,7]).
func (bw *bitWriter) close() int {
	bw.bitContainer |= 1 << bw.bitPos
	bw.buf[bw.pos] = byte(bw.bitContainer)
	return bw.pos + 1
}

// encodeSym encodes one symbol transition: emits nbBits of state, returns next state.
func encodeSym(bw *bitWriter, state uint32, sym *EncSymbol) uint32 {
	nb := uint((state + sym.DeltaNbBits) >> 16)
	bw.addBits(uint64(state), nb)
	return uint32(int32(state>>nb) + sym.DeltaFindState + int32(TableSize))
}

// encodeStream FSE-encodes src in a single stream and appends to dst.
// Wire: [4-byte LE final state][compressed bytes].
func encodeStream(dst, src []byte, freq *[256]uint32) []byte {
	n := len(src)
	if n == 0 {
		var zero [4]byte
		return append(dst, zero[:]...)
	}

	var encTable [256]EncSymbol
	buildEncTable(freq, &encTable)

	// Worst-case: TableLog bits/symbol + close byte + 8-byte flush slack.
	upperBound := (n*TableLog+7)/8 + 2 + 16
	bp := bufpool.Get(upperBound)
	buf := (*bp)[:upperBound]
	defer bufpool.Put(bp)

	bw := newBitWriter(buf)
	state := uint32(TableSize)

	// Unrolled x4: each symbol adds <= TableLog=12 bits; 7 + 4*12 = 55 <= 63,
	// so one flush per 4 symbols is safe.
	i := n - 1
	for ; i >= 3; i -= 4 {
		state = encodeSym(&bw, state, &encTable[src[i]])
		state = encodeSym(&bw, state, &encTable[src[i-1]])
		state = encodeSym(&bw, state, &encTable[src[i-2]])
		state = encodeSym(&bw, state, &encTable[src[i-3]])
		bw.flushBits()
	}
	for ; i >= 0; i-- {
		state = encodeSym(&bw, state, &encTable[src[i]])
		bw.flushBits()
	}
	length := bw.close()

	// Prepend 4-byte state then compressed bytes.
	var stateBuf [4]byte
	binary.LittleEndian.PutUint32(stateBuf[:], state)
	dst = append(dst, stateBuf[:]...)
	dst = append(dst, buf[:length]...)
	return dst
}

// appendInterleaved4 FSE-encodes src as 4 interleaved substreams and appends to dst.
// Wire: [4×uint32 LE states][3×uvarint lengths][substream 0..3].
//
// The 4 lanes are encoded in ONE loop so their serial state chains overlap in
// the CPU pipeline (same ILP trick as the interleaved decoder).
func appendInterleaved4(dst, src []byte, freq *[256]uint32) []byte {
	n := len(src)
	if n == 0 {
		return append(dst, make([]byte, 4*4+3)...)
	}

	var encTable [256]EncSymbol
	buildEncTable(freq, &encTable)

	subMax := ((n+3)/4*TableLog+7)/8 + 2 + 16
	bp := bufpool.Get(subMax * 4)
	scratch := (*bp)[:subMax*4]
	defer bufpool.Put(bp)

	bw0 := newBitWriter(scratch[0:subMax])
	bw1 := newBitWriter(scratch[subMax : 2*subMax])
	bw2 := newBitWriter(scratch[2*subMax : 3*subMax])
	bw3 := newBitWriter(scratch[3*subMax : 4*subMax])
	x0, x1, x2, x3 := uint32(TableSize), uint32(TableSize), uint32(TableSize), uint32(TableSize)

	// Lane k holds symbols k, k+4, k+8, ...; lane k encodes them in reverse.
	// mmax = ceil(n/4); lanes 0..r-1 have mmax symbols, lanes r..3 have mmax-1.
	mmax := (n + 3) / 4
	r := n - 4*(mmax-1) // 1..4 lanes participate in the ragged top row

	j := mmax - 1
	base := j * 4
	switch r {
	case 4:
		x3 = encodeSym(&bw3, x3, &encTable[src[base+3]])
		fallthrough
	case 3:
		x2 = encodeSym(&bw2, x2, &encTable[src[base+2]])
		fallthrough
	case 2:
		x1 = encodeSym(&bw1, x1, &encTable[src[base+1]])
		fallthrough
	case 1:
		x0 = encodeSym(&bw0, x0, &encTable[src[base]])
	}
	bw0.flushBits()
	bw1.flushBits()
	bw2.flushBits()
	bw3.flushBits()

	// Blocked x4: each lane adds <= 4*TableLog = 48 bits on top of <= 7
	// remaining, 55 <= 63 — one flush per lane per 4 rows.
	j = mmax - 2
	for ; j >= 3; j -= 4 {
		base = j * 4
		x0 = encodeSym(&bw0, x0, &encTable[src[base]])
		x1 = encodeSym(&bw1, x1, &encTable[src[base+1]])
		x2 = encodeSym(&bw2, x2, &encTable[src[base+2]])
		x3 = encodeSym(&bw3, x3, &encTable[src[base+3]])
		x0 = encodeSym(&bw0, x0, &encTable[src[base-4]])
		x1 = encodeSym(&bw1, x1, &encTable[src[base-3]])
		x2 = encodeSym(&bw2, x2, &encTable[src[base-2]])
		x3 = encodeSym(&bw3, x3, &encTable[src[base-1]])
		x0 = encodeSym(&bw0, x0, &encTable[src[base-8]])
		x1 = encodeSym(&bw1, x1, &encTable[src[base-7]])
		x2 = encodeSym(&bw2, x2, &encTable[src[base-6]])
		x3 = encodeSym(&bw3, x3, &encTable[src[base-5]])
		x0 = encodeSym(&bw0, x0, &encTable[src[base-12]])
		x1 = encodeSym(&bw1, x1, &encTable[src[base-11]])
		x2 = encodeSym(&bw2, x2, &encTable[src[base-10]])
		x3 = encodeSym(&bw3, x3, &encTable[src[base-9]])
		bw0.flushBits()
		bw1.flushBits()
		bw2.flushBits()
		bw3.flushBits()
	}
	for ; j >= 0; j-- {
		base = j * 4
		x0 = encodeSym(&bw0, x0, &encTable[src[base]])
		x1 = encodeSym(&bw1, x1, &encTable[src[base+1]])
		x2 = encodeSym(&bw2, x2, &encTable[src[base+2]])
		x3 = encodeSym(&bw3, x3, &encTable[src[base+3]])
		bw0.flushBits()
		bw1.flushBits()
		bw2.flushBits()
		bw3.flushBits()
	}

	var subLens [4]int
	subLens[0] = bw0.close()
	subLens[1] = bw1.close()
	subLens[2] = bw2.close()
	subLens[3] = bw3.close()

	// 4 final states.
	var s4 [16]byte
	binary.LittleEndian.PutUint32(s4[0:], x0)
	binary.LittleEndian.PutUint32(s4[4:], x1)
	binary.LittleEndian.PutUint32(s4[8:], x2)
	binary.LittleEndian.PutUint32(s4[12:], x3)
	dst = append(dst, s4[:]...)
	// 3 substream lengths (4th implied by remaining).
	for k := range 3 {
		dst = appendUvarint(dst, uint64(subLens[k]))
	}
	// 4 substream bodies.
	for k := range 4 {
		dst = append(dst, scratch[k*subMax:k*subMax+subLens[k]]...)
	}
	return dst
}
