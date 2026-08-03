// Package tans implements a table-based Finite State Entropy (FSE / tANS) coder.
// It compresses a byte slice using the same wire format as internal/rans — a shared
// normalized frequency table followed by the coded stream — but replaces arithmetic
// renormalization with a single array lookup per symbol, achieving 3–5× throughput.
//
// Wire format (byte stream written by Encode):
//
//	[TagSingle | TagInter4]  1 byte  sub-tag distinguishes tANS from legacy rANS (0/4)
//	[freq table]             ≤512 B  256 uvarint-encoded normalized frequencies (sum=4096)
//	[stream body]            ≤n+16 B compressed payload (single or interleaved-4)
package tans

import (
	"errors"
	"slices"
)

const (
	// TableLog is log2(TableSize). Matches internal/rans scaleBits=12.
	TableLog = 12
	// TableSize is the number of entries in the FSE decode table (4096 = 4 KiB slots).
	// At 4 bytes per entry the decode table occupies 16 KiB — fits in typical L1 cache.
	TableSize = 1 << TableLog

	// TagSingle tags a single-stream tANS blob. Distinct from rans.ransTagSingle=0.
	TagSingle = uint8(1)
	// TagInter4 tags a 4-stream interleaved tANS blob. Distinct from rans.ransTagInter4=4.
	TagInter4 = uint8(5)

	// interleaveMinBytes mirrors rans.interleaveMinBytes: bodies at or above this
	// size use the interleaved-4 path; smaller bodies use single-stream.
	interleaveMinBytes = 4096
)

// IsTag reports whether b is a tANS blob tag. rANS tags (0, 4) and tANS tags
// (1, 5) are disjoint, so mixed pipelines dispatch on the first blob byte.
func IsTag(b byte) bool {
	return b == TagSingle || b == TagInter4
}

// ErrBadTable is returned when a decoded frequency table is malformed.
var ErrBadTable = errors.New("tans: invalid frequency table")

// ErrCorrupt is returned when the tANS stream is truncated or inconsistent.
var ErrCorrupt = errors.New("tans: corrupt stream")

// buildFreqs builds the normalized frequency table (sum == TableSize) from src.
// Every symbol that occurs in src gets freq >= 1. Identical algorithm to rans.buildFreqs.
func buildFreqs(src []byte) [256]uint32 {
	var freq [256]uint32
	if len(src) == 0 {
		return freq
	}
	// 4-way split histogram: skewed inputs hammer one counter; splitting
	// breaks the store-to-load forwarding dependency between iterations.
	var h [4][256]uint32
	i := 0
	for ; i+3 < len(src); i += 4 {
		h[0][src[i]]++
		h[1][src[i+1]]++
		h[2][src[i+2]]++
		h[3][src[i+3]]++
	}
	for ; i < len(src); i++ {
		h[0][src[i]]++
	}
	var raw [256]uint32
	for s := range 256 {
		raw[s] = h[0][s] + h[1][s] + h[2][s] + h[3][s]
	}
	n := uint64(len(src))
	var total uint32
	for s := range 256 {
		if raw[s] == 0 {
			continue
		}
		f := uint32((uint64(raw[s])*TableSize + n/2) / n)
		if f == 0 {
			f = 1
		}
		freq[s] = f
		total += f
	}
	for total > TableSize {
		bi, bf, found := 0, uint32(0), false
		for s := range 256 {
			if freq[s] > 1 && freq[s] > bf {
				bf, bi, found = freq[s], s, true
			}
		}
		if !found {
			break
		}
		freq[bi]--
		total--
	}
	for total < TableSize {
		bi, bf, found := 0, uint32(0), false
		for s := range 256 {
			if freq[s] > bf {
				bf, bi, found = freq[s], s, true
			}
		}
		if !found {
			break
		}
		freq[bi]++
		total++
	}
	return freq
}

// appendTable serializes the 256 normalized frequencies as uvarints.
// Wire format identical to rans.appendTable.
func appendTable(dst []byte, freq *[256]uint32) []byte {
	for s := range 256 {
		dst = appendUvarint(dst, uint64(freq[s]))
	}
	return dst
}

// parseTable reads 256 uvarint frequencies, validates sum == TableSize.
// Returns freq table and bytes consumed.
func parseTable(src []byte) (freq [256]uint32, used int, err error) {
	pos := 0
	var total uint32
	for s := range 256 {
		v, k := uvarint(src[pos:])
		if k <= 0 {
			return freq, 0, ErrBadTable
		}
		if v > TableSize {
			return freq, 0, ErrBadTable
		}
		freq[s] = uint32(v)
		total += uint32(v)
		pos += k
	}
	if total != TableSize {
		return freq, 0, ErrBadTable
	}
	return freq, pos, nil
}

// Encode appends the tANS encoding of src to dst and returns it.
// For bodies >= interleaveMinBytes uses 4-stream interleaved; smaller use single-stream.
func Encode(dst, src []byte) []byte {
	freq := buildFreqs(src)
	// One up-front grow: tag + freq table (≤512B) + worst-case body
	// (TableLog/8 = 1.5 bytes/symbol) + stream headers/slack.
	dst = slices.Grow(dst, 1+512+len(src)*TableLog/8+64)
	if len(src) >= interleaveMinBytes {
		dst = append(dst, TagInter4)
		dst = appendTable(dst, &freq)
		return appendInterleaved4(dst, src, &freq)
	}
	dst = append(dst, TagSingle)
	dst = appendTable(dst, &freq)
	return encodeStream(dst, src, &freq)
}

// Decode parses the tANS blob in src and decompresses exactly n bytes.
// src[0] must be TagSingle or TagInter4 — this function does NOT handle legacy
// rANS tags (0, 4); those are handled by the caller (decoder.go).
func Decode(src []byte, n int) ([]byte, error) {
	if n == 0 {
		return []byte{}, nil
	}
	if n < 0 {
		return nil, ErrCorrupt
	}
	if len(src) < 1 {
		return nil, ErrCorrupt
	}
	tag := src[0]
	src = src[1:]
	freq, used, err := parseTable(src)
	if err != nil {
		return nil, err
	}
	src = src[used:]
	switch tag {
	case TagSingle:
		return decodeStream(src, &freq, n)
	case TagInter4:
		return decodeInterleaved4(src, &freq, n)
	default:
		return nil, ErrCorrupt
	}
}

// appendUvarint encodes v as a LEB128 varint and appends to b.
func appendUvarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

// uvarint decodes a LEB128 varint from b. Returns (value, bytes_consumed).
// Returns (0, -1) on overflow; (0, 0) on empty input.
func uvarint(b []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, c := range b {
		if i > 9 {
			return 0, -1
		}
		if c < 0x80 {
			if i == 9 && c > 1 {
				return 0, -1
			}
			return x | uint64(c)<<s, i + 1
		}
		x |= uint64(c&0x7f) << s
		s += 7
	}
	return 0, 0
}
