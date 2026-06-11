package rans

import "encoding/binary"

// encodeStreamStridedInto rANS-encodes the substream src[k], src[k+n], … into
// the caller-provided buf (which must have len >= m+16, where m is the element
// count) and returns buf[pos:] = [4-byte LE final state][renorm bytes]. Writing
// into a caller buffer lets appendInterleaved place all N substreams in one
// allocation, avoiding the per-substream make() (and its size-class rounding).
func encodeStreamStridedInto(buf, src []byte, k, n int, freq *[256]uint32, cum *[257]uint32) []byte {
	m := max((len(src)-k+n-1)/n, 0)
	pos := len(buf)
	x := uint32(ransByteL)
	for j := m - 1; j >= 0; j-- {
		s := src[k+j*n]
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

// maxInterleaveN bounds the stack-resident state/region scratch in
// appendInterleaved. The shipped stream count is interleaveN (4); the cap leaves
// headroom without forcing those small arrays to the heap.
const maxInterleaveN = 8

// appendInterleaved appends an interleaved-N body to dst (NOT including the tag
// or the shared freq table, which Encode writes): N 4-byte final states, then
// N-1 uvarint substream byte-lengths (the last substream is the remainder), then
// the concatenated substream byte regions. Each substream is an independent
// single-state rANS over the SHARED freq/cum table. All N substreams encode into
// one scratch allocation, partitioned into per-substream regions.
func appendInterleaved(dst, src []byte, freq *[256]uint32, cum *[257]uint32, n int) []byte {
	var statesArr [maxInterleaveN]uint32
	var subsArr [maxInterleaveN][]byte
	states := statesArr[:n]
	subs := subsArr[:n]
	// One allocation for every substream's output. Region k spans m_k+16 bytes
	// (the encodeStream worst case); the regions sum to len(src)+n*16.
	scratch := make([]byte, len(src)+n*16)
	off := 0
	for k := 0; k < n; k++ {
		m := max((len(src)-k+n-1)/n, 0)
		region := scratch[off : off+m+16]
		blob := encodeStreamStridedInto(region, src, k, n, freq, cum) // [4-byte state][bytes]; always >= 4 bytes
		states[k] = binary.LittleEndian.Uint32(blob[:4])
		subs[k] = blob[4:]
		off += m + 16
	}
	var s4 [4]byte
	for k := 0; k < n; k++ {
		binary.LittleEndian.PutUint32(s4[:], states[k])
		dst = append(dst, s4[:]...)
	}
	for k := 0; k < n-1; k++ { // last region length is implied by remainder
		dst = appendUvarint(dst, uint64(len(subs[k])))
	}
	for k := 0; k < n; k++ {
		dst = append(dst, subs[k]...)
	}
	return dst
}

const ransMask = scale - 1

// parseInterleavedRegions reads N states + (N-1) lengths from src, then splits
// the remaining bytes into N substream regions. Returns the states and regions.
func parseInterleavedRegions(src []byte, n int) (states []uint32, regions [][]byte, err error) {
	if len(src) < n*4 {
		return nil, nil, ErrCorrupt
	}
	states = make([]uint32, n)
	for k := 0; k < n; k++ {
		states[k] = binary.LittleEndian.Uint32(src[k*4:])
	}
	src = src[n*4:]
	lens := make([]int, n)
	total := 0
	for k := 0; k < n-1; k++ {
		v, used := uvarint(src)
		if used <= 0 {
			return nil, nil, ErrCorrupt
		}
		src = src[used:]
		if v > uint64(len(src)) {
			return nil, nil, ErrCorrupt
		}
		lens[k] = int(v)
		total += int(v)
	}
	if total > len(src) {
		return nil, nil, ErrCorrupt
	}
	lens[n-1] = len(src) - total // remainder
	regions = make([][]byte, n)
	off := 0
	for k := 0; k < n; k++ {
		regions[k] = src[off : off+lens[k]]
		off += lens[k]
	}
	return states, regions, nil
}

// decodeInterleaved4 decodes four interleaved rANS substreams that share one
// freq/cum table: out[i] comes from substream i%4. The main loop is unrolled by
// 4; the remainder loop continues at the same index and dispatches index i to
// substream i&3 so leftover indices land in the correct substream.
func decodeInterleaved4(states []uint32, regions [][]byte, freq *[256]uint32, slot *[scale]byte, cum *[257]uint32, n int) ([]byte, error) {
	out := make([]byte, n)
	var xs [4]uint32
	var rs [4][]byte
	var ps [4]int
	for k := 0; k < 4; k++ {
		xs[k] = states[k]
		rs[k] = regions[k]
	}
	i := 0
	for ; i+3 < n; i += 4 {
		s0 := slot[xs[0]&ransMask]
		s1 := slot[xs[1]&ransMask]
		s2 := slot[xs[2]&ransMask]
		s3 := slot[xs[3]&ransMask]
		out[i] = s0
		out[i+1] = s1
		out[i+2] = s2
		out[i+3] = s3
		xs[0] = freq[s0]*(xs[0]>>scaleBits) + (xs[0] & ransMask) - cum[s0]
		xs[1] = freq[s1]*(xs[1]>>scaleBits) + (xs[1] & ransMask) - cum[s1]
		xs[2] = freq[s2]*(xs[2]>>scaleBits) + (xs[2] & ransMask) - cum[s2]
		xs[3] = freq[s3]*(xs[3]>>scaleBits) + (xs[3] & ransMask) - cum[s3]
		for xs[0] < ransByteL {
			if ps[0] >= len(rs[0]) {
				return nil, ErrCorrupt
			}
			xs[0] = (xs[0] << 8) | uint32(rs[0][ps[0]])
			ps[0]++
		}
		for xs[1] < ransByteL {
			if ps[1] >= len(rs[1]) {
				return nil, ErrCorrupt
			}
			xs[1] = (xs[1] << 8) | uint32(rs[1][ps[1]])
			ps[1]++
		}
		for xs[2] < ransByteL {
			if ps[2] >= len(rs[2]) {
				return nil, ErrCorrupt
			}
			xs[2] = (xs[2] << 8) | uint32(rs[2][ps[2]])
			ps[2]++
		}
		for xs[3] < ransByteL {
			if ps[3] >= len(rs[3]) {
				return nil, ErrCorrupt
			}
			xs[3] = (xs[3] << 8) | uint32(rs[3][ps[3]])
			ps[3]++
		}
	}
	// remainder: index i maps to substream i&3 (continues from where the main
	// loop stopped, so the modular mapping must use i, not a fresh counter).
	for ; i < n; i++ {
		k := i & 3
		x := xs[k]
		s := slot[x&ransMask]
		out[i] = s
		x = freq[s]*(x>>scaleBits) + (x & ransMask) - cum[s]
		for x < ransByteL {
			if ps[k] >= len(rs[k]) {
				return nil, ErrCorrupt
			}
			x = (x << 8) | uint32(rs[k][ps[k]])
			ps[k]++
		}
		xs[k] = x
	}
	return out, nil
}
