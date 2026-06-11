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

// symbol holds up to 8 bytes of a table entry. bytes/len drive decode; val/mask
// are the same bytes packed little-endian (plus a length mask) so the hot
// match path is a single masked uint64 compare instead of a byte loop.
type symbol struct {
	bytes [8]byte
	val   uint64 // bytes packed little-endian
	mask  uint64 // (1<<(8*len))-1
	len   uint8
}

// packVal returns the little-endian uint64 of b and its length mask.
func packVal(b []byte) (val, mask uint64) {
	for i := range b {
		val |= uint64(b[i]) << (8 * i)
	}
	if len(b) >= 8 {
		mask = ^uint64(0)
	} else {
		mask = (uint64(1) << (8 * len(b))) - 1
	}
	return
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
		s.val, s.mask = packVal(b)
		code := uint8(len(t.symbols))
		t.symbols = append(t.symbols, s)
		t.byFirst[b[0]] = append(t.byFirst[b[0]], code)
	}
	t.buildIndex()
	return t
}

// buildIndex sorts each first-byte bucket longest-symbol-first (tie-break by
// code, for determinism) so match's first prefix hit is the longest.
func (t *SymbolTable) buildIndex() {
	for fb := range t.byFirst {
		cs := t.byFirst[fb]
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
}

// match returns the code and length of the longest symbol that is a prefix of
// s, or (0,0) if none matches. Hot path: load up to 8 input bytes once and test
// each candidate (longest-first) with a single masked uint64 compare.
func (t *SymbolTable) match(s []byte) (uint8, int) {
	var x uint64
	if len(s) >= 8 {
		x = binary.LittleEndian.Uint64(s)
	} else {
		for i := range s {
			x |= uint64(s[i]) << (8 * i)
		}
	}
	avail := len(s)
	for _, code := range t.byFirst[byte(x)] {
		sym := &t.symbols[code]
		if int(sym.len) <= avail && x&sym.mask == sym.val {
			return code, int(sym.len)
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

// CompressedLen returns the number of bytes Compress would emit for src,
// without materializing the output. The columnar size probe only needs the
// coded length to compare codecs, so this avoids allocating (and growing) a
// throwaway destination buffer per sampled string.
func (t *SymbolTable) CompressedLen(src []byte) int {
	n, i := 0, 0
	for i < len(src) {
		_, m := t.match(src[i:])
		if m == 0 {
			n += 2 // escapeCode + literal byte
			i++
			continue
		}
		n++ // single code byte
		i += m
	}
	return n
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

// DecompressN appends the decoding of codes to dst but never lets len(dst)
// exceed limit, returning ok=false the moment a symbol or literal would
// overflow. The decode path pre-sizes the destination slab to limit and passes
// limit here, so a malformed block cannot drive an append past the slab's
// capacity (no transient over-allocation / heap-exhaustion). Never panics.
func (t *SymbolTable) DecompressN(codes, dst []byte, limit int) ([]byte, bool) {
	i := 0
	for i < len(codes) {
		c := codes[i]
		i++
		if c == escapeCode {
			if i >= len(codes) {
				break
			}
			if len(dst) >= limit {
				return dst, false
			}
			dst = append(dst, codes[i])
			i++
			continue
		}
		if int(c) >= len(t.symbols) {
			continue
		}
		s := &t.symbols[c]
		if len(dst)+int(s.len) > limit {
			return dst, false
		}
		dst = append(dst, s.bytes[:s.len]...)
	}
	return dst, true
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
	for range cnt {
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
