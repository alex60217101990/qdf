package vecquant

import (
	"encoding/binary"
	"errors"

	"github.com/alex60217101990/qdf/internal/rans"
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

// Stream layout: mode byte (0=raw varint, 1=rANS) | varuint(rawLen) | body.
func encodeCoords(q []int32) []byte {
	raw := appendVarintZigzag(nil, q)
	packed := rans.Encode(nil, raw)
	// rans.Encode returns the raw bytes when rANS would not be smaller; detect
	// the win by size and pick the smaller framing.
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
		dec, err := rans.Decode(body, int(rawLen))
		if err != nil {
			return nil, err
		}
		raw = dec
	default:
		return nil, errors.New("vecquant: bad mode")
	}
	return readVarintZigzag(raw, count)
}

// appendCosets writes a length-prefixed raw coset-bit byte stream.
func appendCosets(dst, cosets []byte) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(cosets)))
	return append(dst, cosets...)
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
