package qdf

// Wire format constants. The 5-byte file header is 'QDF' + 1-byte version
// + 1-byte flags. Tag space layout is documented inline below.

const (
	Magic0   byte = 'Q'
	Magic1   byte = 'D'
	Magic2   byte = 'F'
	Version1 byte = 0x01
)

// Flag bits in the 5th header byte.
const (
	FlagDense byte = 1 << 0
	// FlagQPack signals that the encoder may emit QPack codec tags
	// (0xE3..0xEF). A decoder that does not recognize the tags will fail
	// with ErrBadTag on the first packed slice; the flag is an early hint
	// so callers can refuse the buffer up front.
	FlagQPack byte = 1 << 1

	// FlagRANS signals that the body (everything after the 5-byte header) is
	// rANS-compressed: varuint(origLen) + 256-entry frequency table +
	// rANS stream. The decoder reverses it before reading tags. Set only
	// when the rANS form is strictly smaller than the plain body.
	FlagRANS byte = 1 << 2

	// FlagColIndex marks a tagColStruct (columnar []struct) payload that
	// carries a fixed-width column-length table right after the shape
	// declaration and before the column bodies. The table is K little-endian
	// uint32 entries (one per column, K = number of columns), each the byte
	// length of the corresponding column body. It lets a reader skip columns
	// it does not need without decoding them. Opt-in (~4 bytes per column);
	// only meaningful when the value is columnar — for any other top-level
	// shape the bit is set but no index block exists.
	FlagColIndex byte = 1 << 3
)

// Tag bytes.
const (
	// 0x00..0x7F  positive fixint
	tagFixintMax = 0x7F

	// 0x80..0x9F  fixstr (len 0..31)
	tagFixstr     = 0x80
	tagFixstrMask = 0x1F

	// 0xA0..0xBF  fixarr (len 0..31)
	tagFixarr     = 0xA0
	tagFixarrMask = 0x1F

	tagNil     = 0xC0
	tagFalse   = 0xC1
	tagTrue    = 0xC2
	tagUint8   = 0xC3
	tagUint16  = 0xC4
	tagUint32  = 0xC5
	tagUint64  = 0xC6
	tagInt8    = 0xC7
	tagInt16   = 0xC8
	tagInt32   = 0xC9
	tagInt64   = 0xCA
	tagFloat32 = 0xCB
	tagFloat64 = 0xCC
	tagStr8    = 0xCD
	tagStr16   = 0xCE
	tagStr32   = 0xCF
	tagBin8    = 0xD0
	tagBin16   = 0xD1
	tagBin32   = 0xD2
	tagArr16   = 0xD3
	tagArr32   = 0xD4
	tagMap8    = 0xD5
	tagMap16   = 0xD6
	tagMap32   = 0xD7

	// 0xD8..0xDF  negfixint (-1..-8) — 3-bit body
	tagNegfixint     = 0xD8
	tagNegfixintMask = 0x07
	negfixintMaxAbs  = 8 // values -1..-8 fit

	tagInternStr = 0xE0
	tagStateRef  = 0xE1
	tagInternBin = 0xE2

	// QPack codec tags (0xE3..0xEF). Each opens a self-described payload
	// that replaces the per-element tag stream for a single slice.
	tagPackBool = 0xE3 // bitpacked []bool: tag, varuint(n), ceil(n/8) bytes (LSB-first)
	tagPackRaw  = 0xE4 // raw-LE numeric: tag, kind byte, varuint(n), n*width bytes (LE)
	tagPackFor  = 0xE5 // Frame-of-Reference bitpacked integer slice:
	//                    tag, kind, bits (0..56), min varuint (zigzag for signed),
	//                    varuint(n), ceil(n*bits/8) bytes (LSB-first).
	tagStateRepeat = 0xE8 // Markov-0 predictor for Dense state-refs:
	//                    a state-ref whose ID equals the immediately
	//                    previous state-ref emission is encoded as a
	//                    single byte (no varuint payload).
	tagStateMTF = 0xE9 // Move-To-Front coding for Dense state-refs:
	//                    encodes the MTF rank (position in the LRU
	//                    list of recently-emitted IDs) instead of the
	//                    raw intern ID. Used when uvarintLen(rank) <
	//                    uvarintLen(id) so the wire never grows over
	//                    the plain tagStateRef encoding.
	tagPackGorilla = 0xE7 // Gorilla XOR-coded float slice:
	//                    tag, kind (qpackKindFloat32/64), varuint(n),
	//                    first value (4 or 8 LE bytes), varuint(numBits),
	//                    ceil(numBits/8) bytes of MSB-first XOR-delta bit-stream.
	tagPackDeltaFor = 0xE6 // Delta + zigzag + Frame-of-Reference integer slice:
	//                    tag, kind, bits (0..56),
	//                    first value (varuint or zigzag varuint),
	//                    minDelta (zigzag varuint),
	//                    varuint(n),
	//                    ceil((n-1)*bits/8) bytes (LSB-first) of (Δᵢ - minDelta).
	tagPackRLE = 0xEB // Run-length encoded integer slice. Wins on
	//                    high-repeat telemetry columns (Status, Level
	//                    enum-likes, sparse counters). Wire form:
	//                       tag, kind (qpackKindUint64/qpackKindInt64),
	//                       varuint(n),
	//                       runs: pairs of (value-varuint, runLen-varuint).
	//                    Signed kinds zigzag-encode the value. Total of
	//                    runLens equals n; the decoder unrolls until n
	//                    elements are produced.
	tagPackDict = 0xED // Dictionary-coded integer slice. Wins on
	//                    enum-like columns where distinct cardinality
	//                    is small (≤ 16) but values are spread out
	//                    enough that Frame-of-Reference can't bit-pack
	//                    them cheaply. Wire form:
	//                       tag, kind, varuint(distinct),
	//                       distinct values (each varuint, zigzag for
	//                         signed kinds),
	//                       varuint(n),
	//                       ceil(n * ceil(log2(distinct)) / 8) bytes
	//                         (LSB-first bit stream of indices).
	//                    bitsPer is implicit (derivable from distinct);
	//                    distinct == 1 emits a zero-width body and the
	//                    decoder broadcasts the single value.
	tagPackPFor = 0xEE // Patched Frame-of-Reference integer slice. The FOR
	//                    body is bit-packed at a reduced width b chosen so the
	//                    few values that don't fit (outliers) go to an
	//                    exception list instead of widening every slot. Wire:
	//                       tag, kind, varuint(n), b(1 byte), <min>,
	//                       body (n*b bits, LSB-first, (delta & mask)),
	//                       varuint(excN),
	//                       excN x ( varuint(dPos), varuint(delta) ).
	//                    <min> is varuint (unsigned kinds) or zigzag-varuint
	//                    (signed). Selected only when strictly smaller than
	//                    every other codec, so the wire never grows.
	tagStatePair = 0xEA // Markov-1 predictor for Dense state-refs.
	//                    Conditioned on the previously emitted state-ref ID
	//                    (lastID), the encoder maintains a small ring of the
	//                    most recent successors per prev. If the current ID
	//                    is in the ring, emit tagStatePair + varuint(rank).
	//                    Ring size pairPredK; rank ∈ [0, pairPredK). The
	//                    encoder picks tagStatePair only when its byte cost
	//                    is strictly smaller than every other state-ref
	//                    variant (Repeat, MTF, raw), so the wire never grows.
	tagMapShape = 0xEC // Struct shape interning (OptShapeIntern) and, for
	//                    string-keyed maps, map key-set interning
	//                    (OptMapShape). Shared sequential shape-ID space.
	//                    Wire form:
	//                       0xEC + varuint(shapeID)
	//                    shapeID == 0 declares a new shape inline:
	//                       0xEC, 0, varuint(N), [N x key-emit], [N x value]
	//                    The decoder assigns the new shape the next
	//                    sequential ID (starting at 1) for reuse.
	//                    shapeID > 0 reuses a previously declared shape:
	//                       0xEC, varuint(id), [N x value]
	//                    N is recovered from the shape table.
	tagColStruct = 0xEF // Columnar container for a slice of homogeneous flat
	//                    structs. Wire: 0xEF, varuint(M), varuint(shapeID)
	//                    (shapeID==0 declares: varuint(K) + K×(name, kind
	//                    byte)), then K columns in field order. Numeric/bool
	//                    columns are a QPack slice payload; string/[]byte
	//                    columns are M values emitted consecutively through the
	//                    intern path. Chosen per-array by a probe; the encoder
	//                    falls back to row-major when columnar would not win.
	//                    A nullable column (kind byte has bit 0x80 set) emits
	//                    ceil(M/8) LSB-first presence-mask bytes followed by
	//                    the dense non-nil values encoded with the base kind's
	//                    codec.

	// 0xF0..0xF2 unassigned (reserved for a future MessagePack-style ext type).
	tagTimestamp = 0xF3

	// ALP (Adaptive Lossless floating-Point, CWI 2023), decimal path,
	// for []float64 under OptCompression. Self-describing:
	//   0xF4, qpackKindFloat64, varuint(n),
	//   d(1), zigzag-varuint(forMin), width(1),
	//   ceil(n*width/8) LSB-first body (absent when width==0),
	//   varuint(excN), excN×(varuint(pos), 8 LE raw float64).
	// Chosen by the float64 picker only when strictly smaller than raw
	// and the Gorilla projection, so it never grows the wire.
	tagPackALP = 0xF4

	// Dictionary-coded string column inside a tagColStruct payload. Emitted
	// as the first byte where a string column's M values would otherwise be
	// written consecutively, so it is self-describing: a decoder peeks the
	// byte and either takes this path or reads M per-value strings. Wire:
	//   0xF5, varuint(d), d×(varuint(len), len bytes),  // distinct table
	//   varuint(M), ceil(M*ceil(log2 d)/8) LSB-first index body (absent
	//   when d==1). Chosen by the column emitter only when the bitpacked
	//   index body beats the per-value run cost, so it never grows the wire.
	tagColStrDict = 0xF5
	tagColStrFSST = 0xF6 // FSST-coded string column (inside tagColStruct)

	// tagHybridColStruct is a columnar container for a slice of MIXED structs:
	// the columnar-eligible scalar/string fields are transposed into columns
	// (same per-column encoding as tagColStruct) while the remaining fields
	// (maps, non-[]byte slices, nested structs, interfaces) are kept row-major
	// in a per-row residual block. Wire:
	//   0xF7, varuint(N),
	//   varuint(shapeID)  // 0 = declare inline: varuint(K), K×(WriteString(name),
	//                     //   byte(colKind | 0xFF=residual)); else reuse by ID
	//   [K_eligible column bodies, same layout as tagColStruct],
	//   N × (K_residual values in field order, existing per-value tags).
	// A decoder that does not implement it sees an unknown tag → ErrBadTag.
	tagHybridColStruct = 0xF7

	// tagColStrRaw is a bulk-materialized string column inside a tagColStruct
	// payload: the high-cardinality counterpart to tagColStrDict. A mostly-
	// distinct column (IDs, GUIDs, emails, free text) cannot dict-compress and,
	// written per value, costs one heap allocation per row on decode. tagColStrRaw
	// instead lays every value down once, length-prefixed, so the decoder
	// materializes the whole column into ONE backing slab (every row a sub-slice)
	// — distinct strings still allocate their bytes once (wire-neutral vs the
	// per-value form, which also stores each distinct value once) but the decode
	// drops from n allocations to one. Chosen automatically (no option) by the
	// column emitter for high-cardinality columns where dict does not fire and
	// intern dedup cannot help. Wire:
	//   0xF8, varuint(n), varuint(total),
	//   n × (varuint(len), len bytes)            // values, interleaved
	// total is the summed value-byte count, so the decoder pre-sizes the slab in
	// one allocation. A decoder that does not implement it sees an unknown tag →
	// ErrBadTag.
	tagColStrRaw = 0xF8

	// tagColStrConst is a single-distinct (constant) string column inside a
	// tagColStruct payload: every row holds the SAME string. The dict codec
	// rejects this (it requires count >= 2 for a bounded index body) and the
	// per-value form stores the value n times (Dense state-repeats it, but the
	// codegen/Fast path does not), so a constant column is both wire-bloated and
	// decodes to n allocations there. tagColStrConst stores the value once plus
	// the row count; the decoder fills n shares of the single owned string (one
	// allocation). Chosen automatically when every value is identical. Wire:
	//   0xF9, varuint(len), len bytes, varuint(n)
	// n is checked against the columnar header's row count, which bounds it; a
	// decoder that does not implement it sees an unknown tag → ErrBadTag.
	tagColStrConst = 0xF9

	// tagColStrDictFC is a front-coded variant of tagColStrDict: the distinct
	// table is SORTED and incrementally (front-) coded — each entry stores the
	// length of the prefix it shares with the previous entry plus only its
	// suffix. The table order is the encoder's free choice (the per-row indices
	// point into it), so this needs no sorted input and is never larger: the
	// encoder emits it only when the front-coded table is strictly smaller than
	// the plain table (the index body is byte-identical between the two forms).
	// Big on prefix-shared medium-cardinality columns (SIDs, DNs, paths, URLs).
	// Wire (indices identical to tagColStrDict):
	//   0xFA, varuint(d),
	//        d×(varuint(sharedPrefixLen), varuint(suffixLen), suffix bytes),  // sorted
	//        varuint(n),
	//        ⌈n·ceil(log2 d)/8⌉ LSB-first bitpacked indices into the sorted table
	// sharedPrefixLen <= len(previous entry); entry 0 has prefixLen 0.
	tagColStrDictFC = 0xFA

	// tagColStrAlpha is an alphabet-aware bit-packed string column inside a
	// tagColStruct payload: the high-cardinality, restricted-alphabet counterpart
	// to tagColStrRaw. When every byte of every value is drawn from a small
	// alphabet (|A| <= 64 — hex, base32, base64, decimal IDs: trace/span/request
	// IDs, hashes, GUIDs), each character is stored in ceil(log2 |A|) bits via
	// pure positional notation instead of 8: hex (|A|=16) halves the body. This
	// is the one class dict (high-card), front-coding (no shared prefix) and FSST
	// (high entropy, few shared substrings) all miss, and it is captured WITHOUT
	// the rANS CPU cost, so it wins on the Balanced (rANS-off) tier.
	//
	// Wire:
	//   0xFB,
	//        varuint(a), a alphabet bytes,            // code k -> alphabet[k], a in [2,64]
	//        varuint(n),                              // row count (cross-checked)
	//        flags byte (bit0 = fixed length),
	//        if fixedLen: varuint(L)  else: n varuint lengths,
	//        ⌈(sum of lengths)·ceil(log2 a)/8⌉ LSB-first bitpacked char codes.
	// Never-larger: emitted only when the packed body + header beats the raw
	// per-value floor. a >= 2 keeps ceil(log2 a) >= 1 so the packed body is
	// non-empty and the decode allocation is buffer-bounded.
	tagColStrAlpha = 0xFB

	// tagColStrDictQ is a plain dictionary string column (identical distinct
	// table to tagColStrDict) whose per-row index is QPack-coded (RLE / Dict /
	// FOR / DeltaFor — the integer-slice picker) instead of a flat ceil(log2 d)-
	// bit pack. A skewed (Zipf-like) low-cardinality column packs its index far
	// below the flat width via run-length / dictionary coding, without the rANS
	// CPU cost, so it wins on the Balanced (rANS-off) tier. The front-coded table
	// variant (tagColStrDictFC) is never paired with a QPack index: front-coding
	// fires on high-cardinality sorted-prefix data whose index is near-uniform,
	// where the picker cannot beat the flat pack.
	//
	// Wire:
	//   0xFC, varuint(d), d×(varuint(len), len bytes),  // distinct table (plain)
	//        <QPack uint64 block>                        // per-row index, kind uint64
	// Never-larger: emitted only when the picker's chosen-codec byte cost is
	// strictly below the flat ceil(log2 d)-bit index body; otherwise tagColStrDict
	// (flat) is written. The QPack block carries its own row count, cross-checked
	// against the columnar header and buffer-bounded before allocation.
	tagColStrDictQ = 0xFC
)

// isStringColumnBlockTag reports whether b begins a self-describing string
// column block (dict / dictFC / FSST / raw / const) rather than a per-value run.
// The columnar decoders peek this to choose the block reader (readStringColumn)
// over a ReadString loop.
func isStringColumnBlockTag(b byte) bool {
	return b == tagColStrDict || b == tagColStrDictFC || b == tagColStrFSST ||
		b == tagColStrRaw || b == tagColStrConst || b == tagColStrAlpha ||
		b == tagColStrDictQ
}

// Varint (ULEB128) helpers. Used for state-table IDs and intern-payload
// lengths. The encoder always appends; the decoder returns the consumed
// length so the caller can advance its cursor.

//go:nosplit
func appendUvarint(b []byte, x uint64) []byte {
	// 3-byte unrolled fast path. Covers values up to 2^21 = 2 097 151
	// which is well past the default maxStateEntries (16 384) and
	// covers every state-ref / shape ID / fixstr length in practical
	// payloads. Multi-byte values fall through to the loop.
	if x < 0x80 {
		return append(b, byte(x))
	}
	if x < 0x4000 {
		return append(b, byte(x)|0x80, byte(x>>7))
	}
	if x < 0x200000 {
		return append(b, byte(x)|0x80, byte(x>>7)|0x80, byte(x>>14))
	}
	for x >= 0x80 {
		b = append(b, byte(x)|0x80)
		x >>= 7
	}
	return append(b, byte(x))
}

// readUvarint decodes a ULEB128 and returns value, bytes-consumed. n==0 means
// not enough input; n<0 means overflow (>10 bytes).
//
// The 1-byte branch stays inline (cost ≤ 80) — most varints in
// practice are state-ref ranks / shape IDs / small lengths < 128.
// Multi-byte values fall through to the loop below, which is
// extracted into a non-inlinable slow path on purpose so the hot
// path keeps its inline budget.
//
//go:nosplit
func readUvarint(b []byte) (uint64, int) {
	if len(b) > 0 && b[0] < 0x80 {
		return uint64(b[0]), 1
	}
	var x uint64
	var shift uint
	for i, c := range b {
		if i >= 10 {
			return 0, -1
		}
		if c < 0x80 {
			// Canonical-form guard, matching encoding/binary.Uvarint: the 10th
			// byte (i==9, shift==63) may only carry bit 63, so c>1 would set
			// bits above 63 — reject instead of silently truncating.
			if i == 9 && c > 1 {
				return 0, -1
			}
			return x | uint64(c)<<shift, i + 1
		}
		x |= uint64(c&0x7F) << shift
		shift += 7
	}
	return 0, 0
}

// uvarintLen returns the number of ULEB128 bytes needed to encode v.
// Inlined by the compiler; used in QPack size estimators and the columnar probe.
func uvarintLen(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

// zigzagEncode64 maps a signed int64 to an unsigned int64 with
// magnitude-preserving low-bit cost: |v| small => result small.
// Used by the FOR / Delta-FOR codecs and the columnar probe.
func zigzagEncode64(v int64) uint64 {
	return uint64((v << 1) ^ (v >> 63))
}

// zigzagDecode64 reverses zigzagEncode64.
func zigzagDecode64(u uint64) int64 {
	return int64((u >> 1) ^ -(u & 1))
}
