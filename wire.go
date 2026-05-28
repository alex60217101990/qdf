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
	tagStateColRepeat = 0xEE // Column-conditional repeat for Dense shape
	//                    field values. Inside a tagMapShape struct, an
	//                    interned string/bytes value equal to the last
	//                    value emitted in the same (shapeID, fieldIdx)
	//                    column is encoded as this single byte (no
	//                    payload). Decoder resolves it from its per-column
	//                    last-value mirror. Only emitted under
	//                    OptDense+OptShapeIntern+OptPairPred.
	// 0xEF reserved.

	tagExt8      = 0xF0
	tagExt16     = 0xF1
	tagExt32     = 0xF2
	tagTimestamp = 0xF3
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
