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

	tagExt8      = 0xF0
	tagExt16     = 0xF1
	tagExt32     = 0xF2
	tagTimestamp = 0xF3
)

// Varint (ULEB128) helpers. Used for state-table IDs and intern-payload
// lengths. The encoder always appends; the decoder returns the consumed
// length so the caller can advance its cursor.

func appendUvarint(b []byte, x uint64) []byte {
	for x >= 0x80 {
		b = append(b, byte(x)|0x80)
		x >>= 7
	}
	return append(b, byte(x))
}

// readUvarint decodes a ULEB128 and returns value, bytes-consumed. n==0 means
// not enough input; n<0 means overflow (>10 bytes).
func readUvarint(b []byte) (uint64, int) {
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
