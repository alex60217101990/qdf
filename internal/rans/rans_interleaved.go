package rans

import "encoding/binary"

// stridedSubstream returns src[k], src[k+N], src[k+2N], … (a fresh slice).
func stridedSubstream(src []byte, k, n int) []byte {
	sz := (len(src) - k + n - 1) / n
	if sz <= 0 {
		return nil
	}
	out := make([]byte, 0, sz)
	for i := k; i < len(src); i += n {
		out = append(out, src[i])
	}
	return out
}

// appendInterleaved appends an interleaved-N body to dst (NOT including the tag
// or the shared freq table, which Encode writes): N 4-byte final states, then
// N-1 uvarint substream byte-lengths (the last substream is the remainder), then
// the concatenated substream byte regions. Each substream is an independent
// single-state rANS over the SHARED freq/cum table.
func appendInterleaved(dst, src []byte, freq *[256]uint32, cum *[257]uint32, n int) []byte {
	states := make([]uint32, n)
	subs := make([][]byte, n)
	for k := 0; k < n; k++ {
		sub := stridedSubstream(src, k, n)
		blob := encodeStream(sub, freq, cum) // [4-byte state][bytes]; always >= 4 bytes
		states[k] = binary.LittleEndian.Uint32(blob[:4])
		subs[k] = blob[4:]
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
