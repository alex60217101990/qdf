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
// and writes complete bytes FORWARD into the buffer.
//
// Convention (must match decoder in tans_decode.go):
//   - addBits: container |= (v&mask) << bitPos. New bits stack ABOVE old ones,
//     so the low bits are final once written and can be flushed immediately.
//   - flush:   writes LOW bytes forward; earlier-written bits are at LOWER addresses.
//   - close:   inserts 1-bit end mark at position bitPos, writes the final byte,
//     returns total payload length.
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
	bitPos       uint // bits valid in bitContainer (0..63)
}

func newBitWriter(buf []byte) bitWriter {
	return bitWriter{buf: buf}
}

//go:nosplit
func (bw *bitWriter) addBits(v uint64, nb uint) {
	bw.bitContainer |= (v & ((1 << nb) - 1)) << bw.bitPos
	bw.bitPos += nb
}

// flush writes complete bytes forward, one byte at a time.
//
//go:nosplit
func (bw *bitWriter) flush() {
	for bw.bitPos >= 8 {
		bw.buf[bw.pos] = byte(bw.bitContainer)
		bw.pos++
		bw.bitContainer >>= 8
		bw.bitPos -= 8
	}
}

// close inserts a 1-bit end mark, writes the final byte, and returns the
// payload length (number of bytes written).
func (bw *bitWriter) close() int {
	// End mark at position bitPos; bitPos is in [0,7] after flush().
	bw.bitContainer |= (1 << bw.bitPos)
	bw.buf[bw.pos] = byte(bw.bitContainer)
	bw.pos++
	return bw.pos
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

	// Worst-case: TableLog bits/symbol + 1 end-mark byte + slack.
	upperBound := (n*TableLog+7)/8 + 2 + 16
	bp := bufpool.Get(upperBound)
	buf := (*bp)[:upperBound]
	defer bufpool.Put(bp)

	bw := newBitWriter(buf)
	state := uint32(TableSize)

	for i := n - 1; i >= 0; i-- {
		s := src[i]
		sym := &encTable[s]
		nbBitsOut := uint((state + sym.DeltaNbBits) >> 16)
		bw.addBits(uint64(state), nbBitsOut)
		bw.flush()
		state = uint32(int32(state>>nbBitsOut) + sym.DeltaFindState + int32(TableSize))
	}
	bw.flush()
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

	var statesArr [4]uint32
	var subLens [4]int

	for k := range 4 {
		statesArr[k] = uint32(TableSize)
		bw := newBitWriter(scratch[k*subMax : (k+1)*subMax])
		m := (n - k + 3) / 4
		for j := m - 1; j >= 0; j-- {
			s := src[k+j*4]
			sym := &encTable[s]
			nbBitsOut := uint((statesArr[k] + sym.DeltaNbBits) >> 16)
			bw.addBits(uint64(statesArr[k]), nbBitsOut)
			bw.flush()
			statesArr[k] = uint32(int32(statesArr[k]>>nbBitsOut) + sym.DeltaFindState + int32(TableSize))
		}
		bw.flush()
		subLens[k] = bw.close()
	}

	// 4 final states.
	var s4 [4]byte
	for k := range 4 {
		binary.LittleEndian.PutUint32(s4[:], statesArr[k])
		dst = append(dst, s4[:]...)
	}
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
