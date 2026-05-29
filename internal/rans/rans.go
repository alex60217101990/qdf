// Package rans is a static order-0 byte range Asymmetric Numeral System
// coder (the canonical rans_byte: 32-bit state, byte renormalization, a
// 12-bit normalized frequency table). It compresses and decompresses a
// single byte slice and has no dependencies on the rest of qdf.
package rans

import (
	"encoding/binary"
	"errors"
)

const (
	scaleBits = 12
	scale     = 1 << scaleBits // 4096; frequencies are normalized to sum to this
	ransByteL = 1 << 23        // renormalization lower bound
)

// ErrBadTable is returned when a decoded frequency table is malformed
// (frequencies do not sum to scale, or a frequency exceeds scale).
var ErrBadTable = errors.New("rans: invalid frequency table")

// ErrCorrupt is returned when the rANS stream is truncated or inconsistent.
var ErrCorrupt = errors.New("rans: corrupt stream")

// buildFreqs builds the normalized frequency table (sum == scale) and the
// cumulative table from src. Every symbol that occurs in src gets freq >= 1.
func buildFreqs(src []byte) (freq [256]uint32, cum [257]uint32) {
	if len(src) == 0 {
		return freq, cum
	}
	var raw [256]uint32
	for _, b := range src {
		raw[b]++
	}
	n := uint64(len(src))
	var total uint32
	for s := range 256 {
		if raw[s] == 0 {
			continue
		}
		f := uint32((uint64(raw[s])*scale + n/2) / n)
		if f == 0 {
			f = 1
		}
		freq[s] = f
		total += f
	}
	// Correct rounding so the frequencies sum to exactly `scale`. Adjust the
	// current largest frequency each step; never drop a used symbol below 1.
	for total > scale {
		bi, bf, found := 0, uint32(0), false
		for s := range 256 {
			if freq[s] > 1 && freq[s] > bf {
				bf, bi, found = freq[s], s, true
			}
		}
		if !found {
			break // unreachable: total>scale implies some freq>1
		}
		freq[bi]--
		total--
	}
	for total < scale {
		bi, bf, found := 0, uint32(0), false
		for s := range 256 {
			if freq[s] > bf {
				bf, bi, found = freq[s], s, true
			}
		}
		if !found {
			break // unreachable: non-empty src has at least one freq>=1
		}
		freq[bi]++
		total++
	}
	var c uint32
	for s := range 256 {
		cum[s] = c
		c += freq[s]
	}
	cum[256] = c
	return freq, cum
}

// buildSlot returns the 4096-entry slot->symbol lookup for decode.
func buildSlot(cum *[257]uint32) *[scale]byte {
	var slot [scale]byte
	for s := range 256 {
		for c := cum[s]; c < cum[s+1]; c++ {
			slot[c] = byte(s)
		}
	}
	return &slot
}

// encodeStream rANS-encodes src and returns the stream: 4-byte little-endian
// final state followed by the renormalization bytes in decode order.
func encodeStream(src []byte, freq *[256]uint32, cum *[257]uint32) []byte {
	size := len(src) + 16
	buf := make([]byte, size)
	pos := size
	x := uint32(ransByteL)
	for i := len(src) - 1; i >= 0; i-- {
		s := src[i]
		f := freq[s]
		xMax := ((ransByteL >> scaleBits) << 8) * f
		for x >= xMax {
			pos--
			buf[pos] = byte(x)
			x >>= 8
		}
		x = ((x / f) << scaleBits) + (x % f) + cum[s]
	}
	pos -= 4
	binary.LittleEndian.PutUint32(buf[pos:], x)
	return buf[pos:]
}

// decodeStream reverses encodeStream, emitting exactly n bytes.
func decodeStream(stream []byte, freq *[256]uint32, slot *[scale]byte, cum *[257]uint32, n int) ([]byte, error) {
	if len(stream) < 4 {
		return nil, ErrCorrupt
	}
	x := binary.LittleEndian.Uint32(stream[:4])
	pos := 4
	out := make([]byte, n)
	for i := range n {
		s := slot[x&(scale-1)]
		out[i] = s
		x = freq[s]*(x>>scaleBits) + (x & (scale - 1)) - cum[s]
		for x < ransByteL {
			if pos >= len(stream) {
				return nil, ErrCorrupt
			}
			x = (x << 8) | uint32(stream[pos])
			pos++
		}
	}
	return out, nil
}

// appendTable serializes the 256 normalized frequencies as varuints.
func appendTable(dst []byte, freq *[256]uint32) []byte {
	for s := range 256 {
		dst = appendUvarint(dst, uint64(freq[s]))
	}
	return dst
}

// parseTable reads 256 varuint frequencies, validates them, and returns the
// freq/cum tables plus the number of bytes consumed.
func parseTable(src []byte) (freq [256]uint32, cum [257]uint32, used int, err error) {
	pos := 0
	var total uint32
	for s := range 256 {
		v, k := uvarint(src[pos:])
		if k <= 0 {
			return freq, cum, 0, ErrBadTable
		}
		if v > scale {
			return freq, cum, 0, ErrBadTable
		}
		freq[s] = uint32(v)
		total += uint32(v)
		pos += k
	}
	if total != scale {
		return freq, cum, 0, ErrBadTable
	}
	var c uint32
	for s := range 256 {
		cum[s] = c
		c += freq[s]
	}
	cum[256] = c
	return freq, cum, pos, nil
}

// Encode appends the order-0 rANS encoding of src (frequency table followed by
// the rANS stream) to dst and returns it. The caller stores the original
// length separately and passes it to Decode.
func Encode(dst, src []byte) []byte {
	freq, cum := buildFreqs(src)
	dst = appendTable(dst, &freq)
	return append(dst, encodeStream(src, &freq, &cum)...)
}

// Decode parses the table from src, rANS-decodes exactly n bytes, and returns
// them. It validates the table and stream and never panics on malformed input.
func Decode(src []byte, n int) ([]byte, error) {
	if n == 0 {
		return []byte{}, nil
	}
	freq, cum, used, err := parseTable(src)
	if err != nil {
		return nil, err
	}
	slot := buildSlot(&cum)
	return decodeStream(src[used:], &freq, slot, &cum, n)
}

// appendUvarint / uvarint are local LEB128 helpers (kept self-contained so the
// package has no qdf dependency).
func appendUvarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func uvarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		if i > 9 {
			return 0, -1
		}
		if c < 0x80 {
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, 0
}
