// Package fsst implements FSST (Fast Static Symbol Table) string compression:
// a learned table of up to 255 substrings (each 1..8 bytes) replaced by 1-byte
// codes, with code 255 reserved as an escape for a following literal byte.
// Decompression is a table lookup; the table is stored on the wire.
package fsst

import (
	"encoding/binary"
	"errors"
)

const (
	escapeCode = 0xFF // 255: next byte is an emitted literal
	maxSymbols = 255  // codes 0..254
	maxSymLen  = 8
)

// symbol holds up to 8 bytes of a table entry.
type symbol struct {
	bytes [8]byte
	len   uint8
}

// SymbolTable maps codes 0..254 to symbols and indexes them for compression.
type SymbolTable struct {
	symbols []symbol     // index == code
	byFirst [256][]uint8 // candidate codes per first byte, longest symbol first
}

// newSymbolTable builds a table from raw symbol byte-strings (≤255, each 1..8
// bytes). Symbols longer than maxSymLen or empty are skipped; excess past 255
// is truncated. The byFirst index is sorted longest-first so Compress's first
// hit is the longest match.
func newSymbolTable(raw [][]byte) *SymbolTable {
	t := &SymbolTable{}
	for _, b := range raw {
		if len(b) == 0 || len(b) > maxSymLen || len(t.symbols) >= maxSymbols {
			continue
		}
		var s symbol
		s.len = uint8(len(b))
		copy(s.bytes[:], b)
		code := uint8(len(t.symbols))
		t.symbols = append(t.symbols, s)
		t.byFirst[b[0]] = append(t.byFirst[b[0]], code)
	}
	for fb := range t.byFirst {
		cs := t.byFirst[fb]
		// stable longest-first; tie-break by code for determinism
		for i := 1; i < len(cs); i++ {
			for j := i; j > 0; j-- {
				a, b := cs[j-1], cs[j]
				if t.symbols[b].len > t.symbols[a].len ||
					(t.symbols[b].len == t.symbols[a].len && b < a) {
					cs[j-1], cs[j] = cs[j], cs[j-1]
				} else {
					break
				}
			}
		}
	}
	return t
}

// match returns the code and length of the longest symbol that is a prefix of
// s, or (0,0) if none matches.
func (t *SymbolTable) match(s []byte) (uint8, int) {
	for _, code := range t.byFirst[s[0]] {
		L := int(t.symbols[code].len)
		if L <= len(s) && string(t.symbols[code].bytes[:L]) == string(s[:L]) {
			return code, L
		}
	}
	return 0, 0
}

// Compress appends the FSST encoding of src to dst and returns dst.
func (t *SymbolTable) Compress(src, dst []byte) []byte {
	i := 0
	for i < len(src) {
		code, n := t.match(src[i:])
		if n == 0 {
			dst = append(dst, escapeCode, src[i])
			i++
			continue
		}
		dst = append(dst, code)
		i += n
	}
	return dst
}

// Decompress appends the decoding of codes to dst and returns dst. It is
// defensive: a truncated trailing escape stops cleanly rather than panicking,
// so it is safe to call on arbitrary input.
func (t *SymbolTable) Decompress(codes, dst []byte) []byte {
	i := 0
	for i < len(codes) {
		c := codes[i]
		i++
		if c == escapeCode {
			if i >= len(codes) {
				break
			}
			dst = append(dst, codes[i])
			i++
			continue
		}
		if int(c) >= len(t.symbols) {
			// unknown code on arbitrary input: skip (Decompress never panics).
			continue
		}
		s := &t.symbols[c]
		dst = append(dst, s.bytes[:s.len]...)
	}
	return dst
}

// MarshalTo appends the serialized symbol table to dst:
//
//	uvarint(count) then count × [ len:1B(1..8) | bytes ].
func (t *SymbolTable) MarshalTo(dst []byte) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(t.symbols)))
	for i := range t.symbols {
		dst = append(dst, t.symbols[i].len)
		dst = append(dst, t.symbols[i].bytes[:t.symbols[i].len]...)
	}
	return dst
}

// SerializedSize returns the number of bytes MarshalTo would write, without
// allocating. Used by the columnar probe to estimate FSST cost cheaply.
func (t *SymbolTable) SerializedSize() int {
	sz := uvarintSize(uint64(len(t.symbols)))
	for i := range t.symbols {
		sz += 1 + int(t.symbols[i].len)
	}
	return sz
}

// uvarintSize returns the encoded length of x as a uvarint (no allocation).
func uvarintSize(x uint64) int {
	n := 1
	for x >= 0x80 {
		x >>= 7
		n++
	}
	return n
}

var errBadTable = errors.New("fsst: malformed symbol table")

// UnmarshalSymbolTable parses a table from b and returns it plus the number of
// bytes consumed. All lengths are validated against b before any use; never
// panics on malformed input.
func UnmarshalSymbolTable(b []byte) (*SymbolTable, int, error) {
	cnt, n := binary.Uvarint(b)
	if n <= 0 || cnt > maxSymbols {
		return nil, 0, errBadTable
	}
	off := n
	raw := make([][]byte, 0, cnt)
	for i := uint64(0); i < cnt; i++ {
		if off >= len(b) {
			return nil, 0, errBadTable
		}
		l := int(b[off])
		off++
		if l < 1 || l > maxSymLen || off+l > len(b) {
			return nil, 0, errBadTable
		}
		raw = append(raw, b[off:off+l])
		off += l
	}
	return newSymbolTable(raw), off, nil
}
