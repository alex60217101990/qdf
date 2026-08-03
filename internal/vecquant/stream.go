package vecquant

import (
	"encoding/binary"
	"errors"

	"github.com/alex60217101990/qdf/internal/rans"
	"github.com/alex60217101990/qdf/internal/tans"
)

func zigzag32(v int32) uint32   { return uint32((v << 1) ^ (v >> 31)) }
func unzigzag32(u uint32) int32 { return int32(u>>1) ^ -int32(u&1) }

// appendVarintZigzag writes each coordinate as a zigzag uvarint.
func appendVarintZigzag(dst []byte, q []int32) []byte {
	var tmp [binary.MaxVarintLen32]byte
	for _, v := range q {
		n := binary.PutUvarint(tmp[:], uint64(zigzag32(v)))
		dst = append(dst, tmp[:n]...)
	}
	return dst
}

// readVarintZigzag reads exactly n zigzag uvarints.
func readVarintZigzag(src []byte, n int) ([]int32, error) {
	out := make([]int32, 0, n)
	off := 0
	for range n {
		u, k := binary.Uvarint(src[off:])
		if k <= 0 {
			return nil, errors.New("vecquant: truncated coord stream")
		}
		off += k
		out = append(out, unzigzag32(uint32(u)))
	}
	return out, nil
}

// Stream layout: mode byte (0=raw varint, 1=entropy-coded) | varuint(rawLen) | body.
// Mode-1 bodies are tANS blobs (tags 1/5) since the tANS switch; legacy rANS
// blobs (tags 0/4) still decode via the tag dispatch in decodeCoords.
// encodeCoordsInto is the buffer-reusing form of encodeCoords: it zigzag-varint
// encodes q into the reused zig buffer, tANS-encodes into the reused ransDst
// buffer, and writes the never-larger framing into out (reused). Returns the
// coord block plus the grown staging buffers so the caller can retain them.
func encodeCoordsInto(q []int32, out, zig, ransDst []byte) (res, zigBack, ransBack []byte) {
	zig = appendVarintZigzag(zig[:0], q)
	ransDst = tans.Encode(ransDst[:0], zig)
	out = out[:0]
	var tmp [binary.MaxVarintLen64]byte
	hdr := binary.PutUvarint(tmp[:], uint64(len(zig)))
	if len(ransDst) < len(zig) {
		out = append(out, 1)
		out = append(out, tmp[:hdr]...)
		out = append(out, ransDst...)
	} else {
		out = append(out, 0)
		out = append(out, tmp[:hdr]...)
		out = append(out, zig...)
	}
	return out, zig, ransDst
}

func encodeCoords(q []int32) []byte {
	raw := appendVarintZigzag(nil, q)
	packed := tans.Encode(nil, raw)
	// tans.Encode always returns an entropy-coded blob; pick the smaller of it
	// and the raw zigzag bytes here (never-larger framing).
	var out []byte
	var tmp [binary.MaxVarintLen64]byte
	hdr := binary.PutUvarint(tmp[:], uint64(len(raw)))
	if len(packed) < len(raw) {
		out = append(out, 1)
		out = append(out, tmp[:hdr]...)
		out = append(out, packed...)
	} else {
		out = append(out, 0)
		out = append(out, tmp[:hdr]...)
		out = append(out, raw...)
	}
	return out
}

func decodeCoords(src []byte, count int) ([]int32, error) {
	if len(src) < 1 {
		return nil, errors.New("vecquant: empty coord block")
	}
	mode := src[0]
	rawLen, k := binary.Uvarint(src[1:])
	if k <= 0 {
		return nil, errors.New("vecquant: bad rawLen")
	}
	// Bound rawLen: each coordinate is at most binary.MaxVarintLen32 bytes.
	maxRaw := uint64(count) * uint64(binary.MaxVarintLen32)
	if rawLen > maxRaw {
		return nil, errors.New("vecquant: rawLen exceeds bound")
	}
	body := src[1+k:]
	var raw []byte
	switch mode {
	case 0:
		if uint64(len(body)) < rawLen {
			return nil, errors.New("vecquant: short raw body")
		}
		raw = body[:rawLen]
	case 1:
		var dec []byte
		var err error
		if len(body) > 0 && tans.IsTag(body[0]) {
			dec, err = tans.Decode(body, int(rawLen))
		} else {
			dec, err = rans.Decode(body, int(rawLen))
		}
		if err != nil {
			return nil, err
		}
		raw = dec
	default:
		return nil, errors.New("vecquant: bad mode")
	}
	return readVarintZigzag(raw, count)
}

// uvarintLen returns the number of bytes binary.PutUvarint would write for v.
func uvarintLen(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

// appendCosets writes a length-prefixed raw coset-bit byte stream.
func appendCosets(dst, cosets []byte) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(cosets)))
	return append(dst, cosets...)
}

// ReadCosets is the exported entry for the qdf wire layer; it reads a
// length-prefixed coset stream and verifies it equals wantBytes.
func ReadCosets(src []byte, wantBytes int) (cosets []byte, used int, err error) {
	return readCosets(src, wantBytes)
}

// readCosets reads a length-prefixed coset stream and verifies the length
// equals wantBytes (the only legal value for the block's shape).
func readCosets(src []byte, wantBytes int) (cosets []byte, used int, err error) {
	n, k := binary.Uvarint(src)
	if k <= 0 {
		return nil, 0, errors.New("vecquant: bad coset length")
	}
	if int(n) != wantBytes {
		return nil, 0, errors.New("vecquant: coset length mismatch")
	}
	if uint64(len(src)-k) < n {
		return nil, 0, errors.New("vecquant: short coset body")
	}
	return src[k : k+int(n)], k + int(n), nil
}
