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

// cand is a compression-index entry: the match-relevant fields of a symbol
// inlined so the hot match loop scans a contiguous, cache-friendly slice instead
// of gathering each symbol out of t.symbols by code. val/mask drive the single
// masked uint64 compare; code/length are the result.
type cand struct {
	val, mask uint64
	code      uint8
	length    uint8
}

// SymbolTable maps codes 0..254 to symbols and indexes them for compression.
type SymbolTable struct {
	symbols []symbol // index == code
	// Compression index: all candidates in one flat slice, grouped by first
	// byte and sorted longest-symbol-first within each group. cands[first[b] :
	// first[b+1]] is byte b's group. One backing allocation (reused across
	// rebuilds) and sequential scan — no 256 sub-slices, no gather into symbols.
	cands []cand
	first [257]int32
}

// newSymbolTable builds a table from raw symbol byte-strings (≤255, each 1..8
// bytes). Symbols longer than maxSymLen or empty are skipped; excess past 255
// is truncated. buildIndex then groups them first-byte, longest-first so
// Compress's first hit is the longest match.
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
		t.symbols = append(t.symbols, s)
	}
	t.buildIndex()
	return t
}

// buildIndex rebuilds the flat first-byte-grouped candidate index from
// t.symbols. Each group is sorted longest-symbol-first (tie-break by code, for
// determinism) so match's first prefix hit is the longest. The cands backing
// array is reused when it already has the capacity (the pooled-builder path
// rebuilds this every training round).
func (t *SymbolTable) buildIndex() {
	var cnt [256]int32
	for i := range t.symbols {
		cnt[t.symbols[i].bytes[0]]++
	}
	var off int32
	for b := range 256 {
		t.first[b] = off
		off += cnt[b]
	}
	t.first[256] = off
	if cap(t.cands) < int(off) {
		t.cands = make([]cand, off)
	} else {
		t.cands = t.cands[:off]
	}
	var cur [256]int32
	copy(cur[:], t.first[:256])
	for i := range t.symbols {
		s := &t.symbols[i]
		b0 := s.bytes[0]
		t.cands[cur[b0]] = cand{val: s.val, mask: s.mask, code: uint8(i), length: s.len}
		cur[b0]++
	}
	for b := range 256 {
		lo, hi := t.first[b], t.first[b+1]
		for i := lo + 1; i < hi; i++ {
			for j := i; j > lo; j-- {
				a, c := &t.cands[j-1], &t.cands[j]
				if c.length > a.length || (c.length == a.length && c.code < a.code) {
					*a, *c = *c, *a
				} else {
					break
				}
			}
		}
	}
}

// match returns the code and length of the longest symbol that is a prefix of
// s, or (0,0) if none matches. Hot path: load up to 8 input bytes once and test
// each candidate (longest-first) with a single masked uint64 compare over the
// contiguous first-byte group (no per-candidate gather into t.symbols).
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
	b0 := int(byte(x))
	cs := t.cands[t.first[b0]:t.first[b0+1]]
	for k := range cs {
		c := &cs[k]
		if int(c.length) <= avail && x&c.mask == c.val {
			return c.code, int(c.length)
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
