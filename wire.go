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
	tagMapShape = 0xEC // Struct/map shape interning for Dense mode.
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
)

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
