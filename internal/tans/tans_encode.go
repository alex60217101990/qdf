package tans

import (
	"encoding/binary"
	"unsafe"

	"github.com/alex60217101990/qdf/internal/bufpool"
)

// EncSymbol holds the two encode constants for one symbol in the FSE encode table.
// The encode table is [256]EncSymbol (2 KB) — kept on the goroutine stack.
type EncSymbol struct {
	// DeltaFindState: added to (state >> nbBitsOut) to get the new state index.
	// Equals cumul[s] - freq[s], where cumul[s] is the symbol's base in the decode table.
	DeltaFindState int32
	// DeltaNbBits encodes the bits-to-flush threshold:
	// nbBitsOut = uint((state + DeltaNbBits) >> 16).
	// Equals (maxBitsOut << 16) - (freq[s] << maxBitsOut).
	DeltaNbBits uint32
}

// buildEncTable fills encOut from a normalized frequency table.
// encOut must be a caller-stack [256]EncSymbol — not heap-allocated.
func buildEncTable(freq *[256]uint32, encOut *[256]EncSymbol) {
	// Compute cumulative table.
	var cumul [256]uint32
	var c uint32
	for s := range 256 {
		cumul[s] = c
		c += freq[s]
	}
	// Fill encode table.
	for s := range 256 {
		f := freq[s]
		if f == 0 {
			continue
		}
		// maxBitsOut: number of bits to flush for this symbol at maximum state.
		// FSE spec: maxBitsOut = TableLog - BIT_highbit32(f-1)
		//   where BIT_highbit32(v) = floor(log2(v)) for v>0, else 0.
		// BIT_highbit32 = bitLen32 - 1 for v>0; bitLen32(0) == 0 handles f==1.
		hl := bitLen32(f - 1)
		if hl > 0 {
			hl--
		}
		maxBitsOut := uint32(TableLog) - hl
		encOut[s].DeltaNbBits = (maxBitsOut << 16) - (f << maxBitsOut)
		encOut[s].DeltaFindState = int32(cumul[s]) - int32(f)
	}
}

// bitLen32 returns the number of bits required to represent v (floor(log2(v))+1),
// or 0 for v == 0. Equivalent to bits.Len32 without importing math/bits.
func bitLen32(v uint32) uint32 {
	if v == 0 {
		return 0
	}
	var n uint32
	for v >= (1 << 16) {
		v >>= 16
		n += 16
	}
	for v >= (1 << 8) {
		v >>= 8
		n += 8
	}
	for v >= (1 << 4) {
		v >>= 4
		n += 4
	}
	for v >= (1 << 2) {
		v >>= 2
		n += 2
	}
	for v >= 2 {
		v >>= 1
		n++
	}
	return n + 1
}

// bitWriter writes bits backward into a pre-allocated buffer using a 64-bit container.
// Flush uses a single 8-byte unsafe.Pointer store (one STUR on ARM64, one MOV on amd64).
type bitWriter struct {
	buf          []byte
	pos          int    // write position (decrements from cap end to start)
	bitContainer uint64
	bitPos       uint   // bits accumulated in bitContainer (0..63)
}

func newBitWriter(buf []byte) bitWriter {
	return bitWriter{buf: buf, pos: len(buf)}
}

//go:nosplit
func (bw *bitWriter) addBits(v uint64, nb uint) {
	bw.bitContainer |= v << bw.bitPos
	bw.bitPos += nb
}

// flush writes all full bytes from bitContainer to buf (backward) using one 8-byte store.
// Must be called when bitPos <= 57 so the store does not overflow the container.
// No-op when fewer than 8 bits are accumulated (e.g. after a 0-bit symbol emission).
//
//go:nosplit
func (bw *bitWriter) flush() {
	nbBytes := bw.bitPos >> 3
	if nbBytes == 0 {
		return
	}
	bw.pos -= int(nbBytes)
	// Single 8-byte write: correct bytes land at buf[bw.pos..bw.pos+nbBytes);
	// the remaining 8-nbBytes bytes are overwritten by the next flush.
	*(*uint64)(unsafe.Pointer(&bw.buf[bw.pos])) = bw.bitContainer
	bw.bitContainer >>= nbBytes * 8
	bw.bitPos -= nbBytes * 8
}

// close flushes remaining bits byte-by-byte and returns the start offset of the
// compressed data within buf. The caller appends buf[close():].
func (bw *bitWriter) close() int {
	for bw.bitPos > 0 {
		bw.pos--
		bw.buf[bw.pos] = byte(bw.bitContainer)
		bw.bitContainer >>= 8
		if bw.bitPos >= 8 {
			bw.bitPos -= 8
		} else {
			bw.bitPos = 0
		}
	}
	return bw.pos
}

// encodeStream FSE-encodes src in a single stream and appends to dst.
// Format: [4-byte LE final state][compressed bytes (backward)].
// Uses bufpool for scratch; encode table on goroutine stack.
func encodeStream(dst, src []byte, freq *[256]uint32) []byte {
	n := len(src)
	if n == 0 {
		var zero [4]byte
		return append(dst, zero[:]...)
	}

	// Stack-resident encode table: 2 KB.
	var encTable [256]EncSymbol
	buildEncTable(freq, &encTable)

	// Worst-case: TableLog bits per symbol + 8-byte unsafe flush slack + 4-byte state.
	upperBound := (n*TableLog+7)/8 + 12 + 16
	bp := bufpool.Get(upperBound)
	buf := (*bp)[:upperBound]
	defer bufpool.Put(bp)

	bw := newBitWriter(buf)
	state := uint32(TableSize)

	// Encode backward: last symbol first.
	for i := n - 1; i >= 0; i-- {
		s := src[i]
		sym := &encTable[s]
		nbBitsOut := uint((state + sym.DeltaNbBits) >> 16)
		bw.addBits(uint64(state), nbBitsOut)
		bw.flush()
		state = uint32(int32(state>>nbBitsOut) + sym.DeltaFindState + int32(TableSize))
	}
	bw.flush()
	startPos := bw.close()

	var stateBuf [4]byte
	binary.LittleEndian.PutUint32(stateBuf[:], state)
	dst = append(dst, stateBuf[:]...)
	dst = append(dst, buf[startPos:upperBound]...)
	return dst
}

// decodeStream is a placeholder; Task 3 implements the real decoder in tans_decode.go.
func decodeStream(_ []byte, _ *[256]uint32, _ int) ([]byte, error) {
	return nil, ErrCorrupt
}

// decodeInterleaved4 is a placeholder; Task 3 implements the real decoder in tans_decode.go.
func decodeInterleaved4(_ []byte, _ *[256]uint32, _ int) ([]byte, error) {
	return nil, ErrCorrupt
}

// appendInterleaved4 FSE-encodes src as 4 interleaved substreams and appends to dst.
// Format: [4×uint32 LE final states][3×uvarint substream lengths][substream 0]..[substream 3].
// All substreams share the same freq table. Uses bufpool for scratch; tables on stack.
func appendInterleaved4(dst, src []byte, freq *[256]uint32) []byte {
	n := len(src)
	if n == 0 {
		// 4 zero states + 3 zero lengths.
		return append(dst, make([]byte, 4*4+3)...)
	}

	var encTable [256]EncSymbol
	buildEncTable(freq, &encTable)

	// Each substream: ceil(n/4) symbols. Worst case: TableLog bits/sym + slack.
	subMax := ((n+3)/4*TableLog+7)/8 + 20
	bp := bufpool.Get(subMax * 4)
	scratch := (*bp)[:subMax*4]
	defer bufpool.Put(bp)

	var statesArr [4]uint32
	var startPositions [4]int

	for k := range 4 {
		statesArr[k] = uint32(TableSize)
		bw := newBitWriter(scratch[k*subMax : (k+1)*subMax])
		m := (n - k + 3) / 4 // elements in substream k
		for j := m - 1; j >= 0; j-- {
			s := src[k+j*4]
			sym := &encTable[s]
			nbBitsOut := uint((statesArr[k] + sym.DeltaNbBits) >> 16)
			bw.addBits(uint64(statesArr[k]), nbBitsOut)
			bw.flush()
			statesArr[k] = uint32(int32(statesArr[k]>>nbBitsOut) + sym.DeltaFindState + int32(TableSize))
		}
		bw.flush()
		startPositions[k] = bw.close()
	}

	// Write 4 final states.
	var s4 [4]byte
	for k := range 4 {
		binary.LittleEndian.PutUint32(s4[:], statesArr[k])
		dst = append(dst, s4[:]...)
	}
	// Write 3 substream lengths (4th is implied by remainder).
	for k := range 3 {
		subLen := subMax - startPositions[k]
		dst = appendUvarint(dst, uint64(subLen))
	}
	// Write 4 substream bodies.
	for k := range 4 {
		dst = append(dst, scratch[k*subMax+startPositions[k]:(k+1)*subMax]...)
	}
	return dst
}
