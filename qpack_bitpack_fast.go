package qdf

import "encoding/binary"

// bitUnpackU64LEFast is a wider-window replacement for bitUnpackU64LE.
// It carries a 128-bit sliding window (hi||lo) and refills it with a
// single 64-bit little-endian load from the input rather than one byte
// at a time. For bitsPer in [1, 56] the inner loop reduces to:
//
//	out[i] = lo & mask
//	lo = (lo >> b) | (hi << (64-b))
//	hi >>= b
//
// which is 4 dependent ops per element with no inner refill branch on
// the hot iterations.
//
// Bit layout is unchanged: LSB-first within each byte. Output of
// bitPackU64LE is consumed by this function bit-for-bit.
func bitUnpackU64LEFast(out []uint64, in []byte, bitsPer int) {
	if bitsPer == 0 {
		clear(out)
		return
	}
	n := len(out)
	if n == 0 {
		return
	}
	mask := uint64(1)<<uint(bitsPer) - 1
	b := uint(bitsPer)

	var lo, hi uint64
	var have uint
	pos := 0
	end := len(in)

	for i := 0; i < n; i++ {
		// Refill: ensure 128-bit window has at least `b` valid bits.
		// Invariant: when we enter the refill, have < b <= 56, so the
		// remaining bits all live in lo and hi is zero.
		if have < b {
			if pos+8 <= end {
				v := binary.LittleEndian.Uint64(in[pos:])
				pos += 8
				lo |= v << have
				if have > 0 {
					hi = v >> (64 - have)
				}
				have += 64
			} else {
				// Tail: byte-by-byte top-up while bytes remain. When
				// `have` exceeds 56 a byte straddles the lo/hi
				// boundary, so split the loaded byte across both
				// halves to avoid losing the high overflow bits.
				for have < 64 && pos < end {
					v := uint64(in[pos])
					lo |= v << have
					if have > 56 {
						hi |= v >> (64 - have)
					}
					pos++
					have += 8
				}
				for have < b && pos < end {
					hi |= uint64(in[pos]) << (have - 64)
					pos++
					have += 8
				}
			}
		}
		out[i] = lo & mask
		lo = (lo >> b) | (hi << (64 - b))
		hi >>= b
		have -= b
	}
}
